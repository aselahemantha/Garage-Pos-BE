package server

import (
	"context"
	"net/http"
	"time"

	"garage-pos-be/internal/auth"
	"garage-pos-be/internal/pkg/response"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

type RouterParams struct {
	Echo        *echo.Echo
	Pool        *pgxpool.Pool
	AuthHandler *auth.Handler
	JWTSecret   string
}

func RegisterRoutes(p RouterParams) {
	// Healthcheck endpoint (no auth, checks DB connection)
	p.Echo.GET("/healthz", func(c echo.Context) error {
		ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Second)
		defer cancel()

		if err := p.Pool.Ping(ctx); err != nil {
			return response.JSONError(c, http.StatusServiceUnavailable, "database_unavailable", "Database ping failed", nil)
		}

		return c.JSON(http.StatusOK, map[string]string{
			"status": "ok",
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
	})

	// API v1 group
	v1 := p.Echo.Group("/api/v1")

	// Auth routes
	authGroup := v1.Group("/auth")
	authGroup.POST("/login", p.AuthHandler.Login)
	authGroup.POST("/refresh", p.AuthHandler.Refresh)
	authGroup.POST("/logout", p.AuthHandler.Logout)
	authGroup.GET("/me", p.AuthHandler.Me, AuthMiddleware(p.JWTSecret))
}
