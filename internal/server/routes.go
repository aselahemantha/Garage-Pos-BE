package server

import (
	"context"
	"net/http"
	"time"

	"garage-pos-be/internal/auth"
	"garage-pos-be/internal/billing"
	"garage-pos-be/internal/catalog"
	"garage-pos-be/internal/customer"
	"garage-pos-be/internal/inventory"
	"garage-pos-be/internal/job"
	"garage-pos-be/internal/pkg/response"
	"garage-pos-be/internal/report"
	"garage-pos-be/internal/settings"
	"garage-pos-be/internal/vehicle"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

type RouterParams struct {
	Echo             *echo.Echo
	Pool             *pgxpool.Pool
	AuthHandler      *auth.Handler
	CustomerHandler  *customer.Handler
	VehicleHandler   *vehicle.Handler
	CatalogHandler   *catalog.Handler
	SettingsHandler  *settings.Handler
	InventoryHandler *inventory.Handler
	JobHandler       *job.Handler
	BillingHandler   *billing.Handler
	ReportHandler    *report.Handler
	JWTSecret        string
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

	// 5. Inventory routes (parts master & stock)
	partsGroup := protected.Group("/parts")
	partsGroup.GET("", p.InventoryHandler.List)
	partsGroup.GET("/:id", p.InventoryHandler.GetByID)
	partsGroup.POST("", p.InventoryHandler.Create, RequireRoles("owner"))
	partsGroup.POST("/:id/stock-in", p.InventoryHandler.StockIn, RequireRoles("owner", "advisor"))

	// 6. Job Cards routes
	jobsGroup := protected.Group("/jobs")
	jobsGroup.GET("/board", p.JobHandler.ListBoard)
	jobsGroup.POST("", p.JobHandler.Create, RequireRoles("owner", "advisor"))
	jobsGroup.GET("/:id", p.JobHandler.GetByID)
	jobsGroup.PATCH("/:id", p.JobHandler.UpdateJob, RequireRoles("owner", "advisor"))
	jobsGroup.POST("/:id/status", p.JobHandler.UpdateStatus)
	jobsGroup.POST("/:id/items", p.JobHandler.AddItem)
	jobsGroup.DELETE("/:id/items/:itemId", p.JobHandler.RemoveItem)

	// 7. Settings routes
	settingsGroup := protected.Group("/settings")
	settingsGroup.GET("", p.SettingsHandler.GetAll)
	settingsGroup.PUT("", p.SettingsHandler.UpdateAll, RequireRoles("owner"))

	// 8. Billing & Invoices routes
	invGroup := protected.Group("/invoices")
	invGroup.GET("", p.BillingHandler.ListInvoices)
	invGroup.POST("", p.BillingHandler.IssueInvoice, RequireRoles("owner", "advisor"))
	invGroup.GET("/:id", p.BillingHandler.GetInvoice)
	invGroup.POST("/:id/payments", p.BillingHandler.RecordPayment, RequireRoles("owner", "advisor"))
	invGroup.POST("/:id/void", p.BillingHandler.VoidInvoice, RequireRoles("owner"))

	// 9. Cash Sessions routes
	cashGroup := protected.Group("/cash-sessions")
	cashGroup.GET("", p.BillingHandler.ListCashSessions)
	cashGroup.GET("/active", p.BillingHandler.GetActiveCashSession)
	cashGroup.POST("", p.BillingHandler.OpenCashSession, RequireRoles("owner", "advisor"))
	cashGroup.POST("/:id/close", p.BillingHandler.CloseCashSession, RequireRoles("owner", "advisor"))

	// 10. Reports & Analytics routes (owner, advisor)
	reportsGroup := protected.Group("/reports")
	reportsGroup.Use(RequireRoles("owner", "advisor"))
	reportsGroup.GET("/summary", p.ReportHandler.GetSummary)
	reportsGroup.GET("/revenue-trend", p.ReportHandler.GetRevenueTrend)
	reportsGroup.GET("/payment-methods", p.ReportHandler.GetPaymentMethods)
	reportsGroup.GET("/top-items", p.ReportHandler.GetTopItems)
	reportsGroup.GET("/end-of-day", p.ReportHandler.GetEndOfDayReport)
}
