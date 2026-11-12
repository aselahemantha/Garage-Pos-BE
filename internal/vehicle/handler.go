package vehicle

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

func (h *Handler) Create(c echo.Context) error {
	var req CreateVehicleRequest
	if err := c.Bind(&req); err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_request", "Invalid request body", nil)
	}

	if err := h.validator.Struct(&req); err != nil {
		return response.JSONError(c, http.StatusBadRequest, "validation_failed", err.Error(), nil)
	}

	v, err := h.service.Create(c.Request().Context(), req)
	if err != nil {
		if errors.Is(err, ErrDuplicatePlate) {
			return response.JSONError(c, http.StatusConflict, "duplicate_plate", err.Error(), nil)
		}
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to create vehicle", nil)
	}

	return response.JSONSuccess(c, http.StatusCreated, v)
}

func (h *Handler) GetByID(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_id", "Invalid vehicle ID", nil)
	}

	v, err := h.service.GetByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, ErrVehicleNotFound) {
			return response.JSONError(c, http.StatusNotFound, "not_found", "Vehicle not found", nil)
		}
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to get vehicle", nil)
	}

	return response.JSONSuccess(c, http.StatusOK, v)
}

func (h *Handler) GetHistory(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_id", "Invalid vehicle ID", nil)
	}

	hist, err := h.service.GetHistory(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, ErrVehicleNotFound) {
			return response.JSONError(c, http.StatusNotFound, "not_found", "Vehicle not found", nil)
		}
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to get vehicle history", nil)
	}

	return response.JSONSuccess(c, http.StatusOK, hist)
}

func (h *Handler) Search(c echo.Context) error {
	query := c.QueryParam("q")
	plate := c.QueryParam("plate")
	if plate != "" {
		v, err := h.service.GetByPlate(c.Request().Context(), plate)
		if err != nil {
			if errors.Is(err, ErrVehicleNotFound) {
				return response.JSONError(c, http.StatusNotFound, "not_found", "Vehicle plate not found", nil)
			}
			return response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to find vehicle", nil)
		}
		return response.JSONSuccess(c, http.StatusOK, v)
	}

	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))

	vehicles, total, err := h.service.Search(c.Request().Context(), query, page, pageSize)
	if err != nil {
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to search vehicles", nil)
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	return response.JSONList(c, http.StatusOK, vehicles, response.Meta{
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	})
}

func (h *Handler) Update(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_id", "Invalid vehicle ID", nil)
	}

	var req UpdateVehicleRequest
	if err := c.Bind(&req); err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_request", "Invalid request body", nil)
	}

	v, err := h.service.Update(c.Request().Context(), id, req)
	if err != nil {
		if errors.Is(err, ErrVehicleNotFound) {
			return response.JSONError(c, http.StatusNotFound, "not_found", "Vehicle not found", nil)
		}
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to update vehicle", nil)
	}

	return response.JSONSuccess(c, http.StatusOK, v)
}
