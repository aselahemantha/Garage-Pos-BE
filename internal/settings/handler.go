package settings

import (
	"net/http"

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

func (h *Handler) GetAll(c echo.Context) error {
	res, err := h.service.GetAll(c.Request().Context())
	if err != nil {
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to get settings", nil)
	}
	return response.JSONSuccess(c, http.StatusOK, res)
}

func (h *Handler) UpdateAll(c echo.Context) error {
	var req UpdateSettingsRequest
	if err := c.Bind(&req); err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_request", "Invalid request body", nil)
	}

	res, err := h.service.UpdateAll(c.Request().Context(), req)
	if err != nil {
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to update settings", nil)
	}

	return response.JSONSuccess(c, http.StatusOK, res)
}
