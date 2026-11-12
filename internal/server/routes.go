package server

import (
	"context"
	"net/http"
	"time"

	"garage-pos-be/internal/auth"
	"garage-pos-be/internal/catalog"
	"garage-pos-be/internal/customer"
	"garage-pos-be/internal/pkg/response"
	"garage-pos-be/internal/settings"
	"garage-pos-be/internal/vehicle"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

type RouterParams struct {
	Echo            *echo.Echo
	Pool            *pgxpool.Pool
	AuthHandler     *auth.Handler
	CustomerHandler *customer.Handler
	VehicleHandler  *vehicle.Handler
	CatalogHandler  *catalog.Handler
	SettingsHandler *settings.Handler
	JWTSecret       string
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

	// 1. Auth routes (public)
	authGroup := v1.Group("/auth")
	authGroup.POST("/login", p.AuthHandler.Login)
	authGroup.POST("/refresh", p.AuthHandler.Refresh)
	authGroup.POST("/logout", p.AuthHandler.Logout)
	authGroup.GET("/me", p.AuthHandler.Me, AuthMiddleware(p.JWTSecret))

	// Protected API group
	protected := v1.Group("")
	protected.Use(AuthMiddleware(p.JWTSecret))

	// 2. Customers routes (owner, advisor)
	custGroup := protected.Group("/customers")
	custGroup.Use(RequireRoles("owner", "advisor"))
	custGroup.GET("", p.CustomerHandler.Search)
	custGroup.POST("", p.CustomerHandler.Create)
	custGroup.GET("/:id", p.CustomerHandler.GetByID)
	custGroup.PATCH("/:id", p.CustomerHandler.Update)

	// 3. Vehicles routes (owner, advisor, mechanic for read)
	vehGroup := protected.Group("/vehicles")
	vehGroup.GET("", p.VehicleHandler.Search)
	vehGroup.GET("/:id", p.VehicleHandler.GetByID)
	vehGroup.POST("", p.VehicleHandler.Create, RequireRoles("owner", "advisor"))
	vehGroup.PATCH("/:id", p.VehicleHandler.Update, RequireRoles("owner", "advisor"))

	// 4. Catalog routes
	catGroup := protected.Group("/service-categories")
	catGroup.GET("", p.CatalogHandler.ListCategories)
	catGroup.POST("", p.CatalogHandler.CreateCategory, RequireRoles("owner"))

	svcGroup := protected.Group("/services")
	svcGroup.GET("", p.CatalogHandler.ListServices)
	svcGroup.POST("", p.CatalogHandler.CreateService, RequireRoles("owner"))
	svcGroup.PATCH("/:id", p.CatalogHandler.UpdateService, RequireRoles("owner"))

	// 5. Settings routes
	settingsGroup := protected.Group("/settings")
	settingsGroup.GET("", p.SettingsHandler.GetAll)
	settingsGroup.PUT("", p.SettingsHandler.UpdateAll, RequireRoles("owner"))
}
