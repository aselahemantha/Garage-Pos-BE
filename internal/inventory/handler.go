package inventory

import (
	"errors"
	"net/http"
	"strconv"

	"garage-pos-be/internal/auth"
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

func (h *Handler) List(c echo.Context) error {
	search := c.QueryParam("q")
	var category *string
	if c.QueryParam("category") != "" {
		cat := c.QueryParam("category")
		category = &cat
	}

	var activeOnly *bool
	if c.QueryParam("active") != "" {
		b := c.QueryParam("active") == "true"
		activeOnly = &b
	}

	parts, err := h.service.List(c.Request().Context(), search, category, activeOnly)
	if err != nil {
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to list parts", nil)
	}

	return response.JSONSuccess(c, http.StatusOK, parts)
}

func (h *Handler) GetByID(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_id", "Invalid part ID", nil)
	}

	part, err := h.service.GetByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, ErrPartNotFound) {
			return response.JSONError(c, http.StatusNotFound, "not_found", "Part not found", nil)
		}
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to get part", nil)
	}

	return response.JSONSuccess(c, http.StatusOK, part)
}

func (h *Handler) Create(c echo.Context) error {
	var req CreatePartRequest
	if err := c.Bind(&req); err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_request", "Invalid request body", nil)
	}

	if err := h.validator.Struct(&req); err != nil {
		return response.JSONError(c, http.StatusBadRequest, "validation_failed", err.Error(), nil)
	}

	part, err := h.service.Create(c.Request().Context(), req)
	if err != nil {
		if errors.Is(err, ErrDuplicateSKU) {
			return response.JSONError(c, http.StatusConflict, "duplicate_sku", err.Error(), nil)
		}
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to create part", nil)
	}

	return response.JSONSuccess(c, http.StatusCreated, part)
}

func (h *Handler) StockIn(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_id", "Invalid part ID", nil)
	}

	var req StockInRequest
	if err := c.Bind(&req); err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_request", "Invalid request body", nil)
	}

	if err := h.validator.Struct(&req); err != nil {
		return response.JSONError(c, http.StatusBadRequest, "validation_failed", err.Error(), nil)
	}

	claims, _ := c.Get("user").(*auth.JWTClaims)
	var userID int64 = 1
	if claims != nil {
		userID = claims.UserID
	}

	if err := h.service.StockIn(c.Request().Context(), id, userID, req); err != nil {
		if errors.Is(err, ErrPartNotFound) {
			return response.JSONError(c, http.StatusNotFound, "not_found", "Part not found", nil)
		}
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to add stock", nil)
	}

	return response.JSONSuccess(c, http.StatusOK, map[string]string{"message": "Stock added successfully"})
}
