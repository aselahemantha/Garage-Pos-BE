package report

import (
	"net/http"
	"strconv"

	"garage-pos-be/internal/pkg/response"

	"github.com/labstack/echo/v4"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}

func (h *Handler) GetSummary(c echo.Context) error {
	period := c.QueryParam("period")
	startDate := c.QueryParam("start_date")
	endDate := c.QueryParam("end_date")

	summary, err := h.service.GetSummary(c.Request().Context(), period, startDate, endDate)
	if err != nil {
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", err.Error(), nil)
	}

	return response.JSONSuccess(c, http.StatusOK, summary)
}

func (h *Handler) GetRevenueTrend(c echo.Context) error {
	period := c.QueryParam("period")
	startDate := c.QueryParam("start_date")
	endDate := c.QueryParam("end_date")

	trend, err := h.service.GetRevenueTrend(c.Request().Context(), period, startDate, endDate)
	if err != nil {
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", err.Error(), nil)
	}

	return response.JSONSuccess(c, http.StatusOK, trend)
}

func (h *Handler) GetPaymentMethods(c echo.Context) error {
	period := c.QueryParam("period")
	startDate := c.QueryParam("start_date")
	endDate := c.QueryParam("end_date")

	methods, err := h.service.GetPaymentMethods(c.Request().Context(), period, startDate, endDate)
	if err != nil {
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", err.Error(), nil)
	}

	return response.JSONSuccess(c, http.StatusOK, methods)
}

func (h *Handler) GetTopItems(c echo.Context) error {
	period := c.QueryParam("period")
	startDate := c.QueryParam("start_date")
	endDate := c.QueryParam("end_date")
	limit := int32(5)
	if l := c.QueryParam("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = int32(parsed)
		}
	}

	items, err := h.service.GetTopItems(c.Request().Context(), period, startDate, endDate, limit)
	if err != nil {
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", err.Error(), nil)
	}

	return response.JSONSuccess(c, http.StatusOK, items)
}

func (h *Handler) GetEndOfDayReport(c echo.Context) error {
	date := c.QueryParam("date")

	report, err := h.service.GetEndOfDayReport(c.Request().Context(), date)
	if err != nil {
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", err.Error(), nil)
	}

	return response.JSONSuccess(c, http.StatusOK, report)
}
