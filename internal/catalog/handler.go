package catalog

import (
	"errors"
	"net/http"
	"strconv"

	"garage-pos-be/internal/pkg/response"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

type Handler struct {
	service   *Service
	validator *validator.Validate
}

func NewHandler(service *Service, v *validator.Validate) *Handler {
	return &Handler{
		service:   service,
		validator: v,
	}
}

func (h *Handler) ListCategories(c echo.Context) error {
	cats, err := h.service.ListCategories(c.Request().Context())
	if err != nil {
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to list categories", nil)
	}
	return response.JSONSuccess(c, http.StatusOK, cats)
}

func (h *Handler) CreateCategory(c echo.Context) error {
	var req CreateCategoryRequest
	if err := c.Bind(&req); err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_request", "Invalid request body", nil)
	}

	if err := h.validator.Struct(&req); err != nil {
		return response.JSONError(c, http.StatusBadRequest, "validation_failed", err.Error(), nil)
	}

	cat, err := h.service.CreateCategory(c.Request().Context(), req)
	if err != nil {
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to create category", nil)
	}

	return response.JSONSuccess(c, http.StatusCreated, cat)
}

func (h *Handler) ListServices(c echo.Context) error {
	var catID *int64
	if c.QueryParam("category_id") != "" {
		id, err := strconv.ParseInt(c.QueryParam("category_id"), 10, 64)
		if err == nil {
			catID = &id
		}
	}

	var activeOnly *bool
	if c.QueryParam("active") != "" {
		b := c.QueryParam("active") == "true"
		activeOnly = &b
	}

	services, err := h.service.ListServices(c.Request().Context(), catID, activeOnly)
	if err != nil {
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to list services", nil)
	}

	return response.JSONSuccess(c, http.StatusOK, services)
}

func (h *Handler) CreateService(c echo.Context) error {
	var req CreateServiceRequest
	if err := c.Bind(&req); err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_request", "Invalid request body", nil)
	}

	if err := h.validator.Struct(&req); err != nil {
		return response.JSONError(c, http.StatusBadRequest, "validation_failed", err.Error(), nil)
	}

	svc, err := h.service.CreateService(c.Request().Context(), req)
	if err != nil {
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to create service", nil)
	}

	return response.JSONSuccess(c, http.StatusCreated, svc)
}

func (h *Handler) UpdateService(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_id", "Invalid service ID", nil)
	}

	var req UpdateServiceRequest
	if err := c.Bind(&req); err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_request", "Invalid request body", nil)
	}

	svc, err := h.service.UpdateService(c.Request().Context(), id, req)
	if err != nil {
		if errors.Is(err, ErrServiceNotFound) {
			return response.JSONError(c, http.StatusNotFound, "not_found", "Service not found", nil)
		}
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to update service", nil)
	}

	return response.JSONSuccess(c, http.StatusOK, svc)
}
