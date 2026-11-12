package user

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
	users, err := h.service.List(c.Request().Context())
	if err != nil {
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to list users", nil)
	}
	return response.JSONSuccess(c, http.StatusOK, users)
}

func (h *Handler) GetByID(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_id", "Invalid user ID", nil)
	}

	u, err := h.service.GetByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return response.JSONError(c, http.StatusNotFound, "user_not_found", "User not found", nil)
		}
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to get user", nil)
	}

	return response.JSONSuccess(c, http.StatusOK, u)
}

func (h *Handler) Create(c echo.Context) error {
	var req CreateUserRequest
	if err := c.Bind(&req); err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_request", "Invalid request body", nil)
	}

	if err := h.validator.Struct(&req); err != nil {
		return response.JSONError(c, http.StatusBadRequest, "validation_failed", err.Error(), nil)
	}

	u, err := h.service.Create(c.Request().Context(), req)
	if err != nil {
		if errors.Is(err, ErrUsernameTaken) {
			return response.JSONError(c, http.StatusConflict, "username_taken", "Username is already taken", nil)
		}
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to create user", nil)
	}

	return response.JSONSuccess(c, http.StatusCreated, u)
}

func (h *Handler) Update(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_id", "Invalid user ID", nil)
	}

	var req UpdateUserRequest
	if err := c.Bind(&req); err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_request", "Invalid request body", nil)
	}

	if err := h.validator.Struct(&req); err != nil {
		return response.JSONError(c, http.StatusBadRequest, "validation_failed", err.Error(), nil)
	}

	// Prevent user from deactivating themselves
	if req.IsActive != nil && !*req.IsActive {
		if claims, ok := c.Get("user").(*auth.JWTClaims); ok && claims.UserID == id {
			return response.JSONError(c, http.StatusBadRequest, "cannot_deactivate_self", "You cannot deactivate your own account", nil)
		}
	}

	u, err := h.service.Update(c.Request().Context(), id, req)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return response.JSONError(c, http.StatusNotFound, "user_not_found", "User not found", nil)
		}
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to update user", nil)
	}

	return response.JSONSuccess(c, http.StatusOK, u)
}

func (h *Handler) ChangePassword(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_id", "Invalid user ID", nil)
	}

	var req ChangePasswordRequest
	if err := c.Bind(&req); err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_request", "Invalid request body", nil)
	}

	if err := h.validator.Struct(&req); err != nil {
		return response.JSONError(c, http.StatusBadRequest, "validation_failed", err.Error(), nil)
	}

	if err := h.service.ChangePassword(c.Request().Context(), id, req.NewPassword); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return response.JSONError(c, http.StatusNotFound, "user_not_found", "User not found", nil)
		}
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to change password", nil)
	}

	return response.JSONSuccess(c, http.StatusOK, map[string]string{
		"message": "Password updated successfully",
	})
}
