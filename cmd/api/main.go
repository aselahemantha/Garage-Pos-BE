package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"garage-pos-be/db/migrations"
	"garage-pos-be/internal/auth"
	"garage-pos-be/internal/catalog"
	"garage-pos-be/internal/config"
	"garage-pos-be/internal/customer"
	"garage-pos-be/internal/pkg/response"
	"garage-pos-be/internal/server"
	"garage-pos-be/internal/settings"
	"garage-pos-be/internal/store"
	"garage-pos-be/internal/vehicle"

	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/pressly/goose/v3"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.Load()

	slog.Info("starting garage pos api", "env", cfg.AppEnv, "addr", cfg.HTTPAddr)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Connect to PostgreSQL
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to parse database config", "error", err)
		os.Exit(1)
	}
	poolCfg.MaxConns = 25
	poolCfg.MinConns = 2
	poolCfg.MaxConnLifetime = 1 * time.Hour

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		slog.Error("failed to create database pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Warn("database connection not ready, waiting...", "error", err)
	}

	// 2. Run database migrations using goose
	db := stdlib.OpenDBFromPool(pool)
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		slog.Error("failed to set goose dialect", "error", err)
		os.Exit(1)
	}
	slog.Info("running database migrations...")
	if err := goose.Up(db, "."); err != nil {
		slog.Error("failed to run database migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("database migrations completed successfully")

	// 3. Initialize queries & services
	queries := store.New(pool)
	v := validator.New()

	authService := auth.NewService(queries, pool, cfg)
	vehicleService := vehicle.NewService(queries, pool)
	customerService := customer.NewService(queries, pool, vehicleService)
	catalogService := catalog.NewService(queries, pool)
	settingsService := settings.NewService(queries, pool)

	// Seed initial owner account and catalog defaults
	if err := authService.SeedOwner(ctx); err != nil {
		slog.Warn("failed to seed initial owner", "error", err)
	} else {
		slog.Info("initial owner seed checked", "username", cfg.SeedOwnerUsername)
	}

	if err := catalogService.SeedDefaults(ctx); err != nil {
		slog.Warn("failed to seed default catalog", "error", err)
	}

	// Handlers
	authHandler := auth.NewHandler(authService, v, cfg.AppEnv == "production")
	vehicleHandler := vehicle.NewHandler(vehicleService, v)
	customerHandler := customer.NewHandler(customerService, v)
	catalogHandler := catalog.NewHandler(catalogService, v)
	settingsHandler := settings.NewHandler(settingsService)

	// 4. Initialize HTTP server
	e := echo.New()
	e.HideBanner = true
	e.HTTPErrorHandler = response.CentralErrorHandler

	// Middlewares
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: `{"time":"${time_rfc3339}","id":"${id}","remote_ip":"${remote_ip}",` +
			`"method":"${method}","uri":"${uri}","status":${status},"latency_ms":${latency}}` + "\n",
	}))
	e.Use(middleware.BodyLimit("1M"))
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"http://localhost:5173", "http://127.0.0.1:5173"},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete, http.MethodPut, http.MethodOptions},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
		AllowCredentials: true,
	}))

	server.RegisterRoutes(server.RouterParams{
		Echo:            e,
		Pool:            pool,
		AuthHandler:     authHandler,
		CustomerHandler: customerHandler,
		VehicleHandler:  vehicleHandler,
		CatalogHandler:  catalogHandler,
		SettingsHandler: settingsHandler,
		JWTSecret:       cfg.JWTSecret,
	})

	// 5. Graceful shutdown
	go func() {
		if err := e.Start(cfg.HTTPAddr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server gracefully...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}
	slog.Info("server exited cleanly")
}
