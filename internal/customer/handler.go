package customer

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
	var req CreateCustomerRequest
	if err := c.Bind(&req); err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_request", "Invalid request body", nil)
	}

	if err := h.validator.Struct(&req); err != nil {
		return response.JSONError(c, http.StatusBadRequest, "validation_failed", err.Error(), nil)
	}

	cust, err := h.service.Create(c.Request().Context(), req)
	if err != nil {
		if errors.Is(err, ErrDuplicatePhone) {
			return response.JSONError(c, http.StatusConflict, "duplicate_phone", err.Error(), nil)
		}
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to create customer", nil)
	}

	return response.JSONSuccess(c, http.StatusCreated, cust)
}

func (h *Handler) GetByID(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_id", "Invalid customer ID", nil)
	}

	cust, err := h.service.GetByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, ErrCustomerNotFound) {
			return response.JSONError(c, http.StatusNotFound, "not_found", "Customer not found", nil)
		}
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to get customer", nil)
	}

	return response.JSONSuccess(c, http.StatusOK, cust)
}

func (h *Handler) Search(c echo.Context) error {
	phone := c.QueryParam("phone")
	if phone != "" {
		cust, err := h.service.GetByPhone(c.Request().Context(), phone)
		if err != nil {
			if errors.Is(err, ErrCustomerNotFound) {
				return response.JSONError(c, http.StatusNotFound, "not_found", "Customer with this phone not found", nil)
			}
			return response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to find customer", nil)
		}
		return response.JSONSuccess(c, http.StatusOK, cust)
	}

	query := c.QueryParam("q")
	page, _ := strconv.Atoi(c.QueryParam("page"))
	pageSize, _ := strconv.Atoi(c.QueryParam("page_size"))

	customers, total, err := h.service.Search(c.Request().Context(), query, page, pageSize)
	if err != nil {
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to search customers", nil)
	}

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	return response.JSONList(c, http.StatusOK, customers, response.Meta{
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	})
}

func (h *Handler) Update(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_id", "Invalid customer ID", nil)
	}

	var req UpdateCustomerRequest
	if err := c.Bind(&req); err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_request", "Invalid request body", nil)
	}

	cust, err := h.service.Update(c.Request().Context(), id, req)
	if err != nil {
		if errors.Is(err, ErrCustomerNotFound) {
			return response.JSONError(c, http.StatusNotFound, "not_found", "Customer not found", nil)
		}
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to update customer", nil)
	}

	return response.JSONSuccess(c, http.StatusOK, cust)
}
