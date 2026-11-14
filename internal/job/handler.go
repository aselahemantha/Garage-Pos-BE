package job

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

func (h *Handler) Create(c echo.Context) error {
	var req CreateJobCardRequest
	if err := c.Bind(&req); err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_request", "Invalid request body", nil)
	}

	if err := h.validator.Struct(&req); err != nil {
		return response.JSONError(c, http.StatusBadRequest, "validation_failed", err.Error(), nil)
	}

	claims, _ := c.Get("user").(*auth.JWTClaims)
	var advisorID int64 = 1
	if claims != nil {
		advisorID = claims.UserID
	}

	jobDetail, err := h.service.Create(c.Request().Context(), advisorID, req)
	if err != nil {
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", err.Error(), nil)
	}

	return response.JSONSuccess(c, http.StatusCreated, jobDetail)
}

func (h *Handler) GetByID(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_id", "Invalid job card ID", nil)
	}

	jobDetail, err := h.service.GetByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, ErrJobNotFound) {
			return response.JSONError(c, http.StatusNotFound, "not_found", "Job card not found", nil)
		}
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to get job card", nil)
	}

	return response.JSONSuccess(c, http.StatusOK, jobDetail)
}

func (h *Handler) ListBoard(c echo.Context) error {
	boardJobs, err := h.service.ListBoard(c.Request().Context())
	if err != nil {
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to load job board", nil)
	}

	return response.JSONSuccess(c, http.StatusOK, boardJobs)
}

func (h *Handler) UpdateStatus(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_id", "Invalid job card ID", nil)
	}

	var req UpdateJobStatusRequest
	if err := c.Bind(&req); err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_request", "Invalid request body", nil)
	}

	if err := h.validator.Struct(&req); err != nil {
		return response.JSONError(c, http.StatusBadRequest, "validation_failed", err.Error(), nil)
	}

	updated, err := h.service.UpdateStatus(c.Request().Context(), id, req.Status)
	if err != nil {
		if errors.Is(err, ErrInvalidTransition) {
			return response.JSONError(c, http.StatusConflict, "invalid_transition", err.Error(), nil)
		}
		if errors.Is(err, ErrJobNotFound) {
			return response.JSONError(c, http.StatusNotFound, "not_found", "Job card not found", nil)
		}
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", err.Error(), nil)
	}

	return response.JSONSuccess(c, http.StatusOK, updated)
}

func (h *Handler) AddItem(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_id", "Invalid job card ID", nil)
	}

	var req CreateJobItemInput
	if err := c.Bind(&req); err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_request", "Invalid request body", nil)
	}

	if err := h.validator.Struct(&req); err != nil {
		return response.JSONError(c, http.StatusBadRequest, "validation_failed", err.Error(), nil)
	}

	detail, err := h.service.AddItem(c.Request().Context(), id, req)
	if err != nil {
		if errors.Is(err, ErrJobNotFound) {
			return response.JSONError(c, http.StatusNotFound, "not_found", "Job card not found", nil)
		}
		if errors.Is(err, ErrJobAlreadyCompleted) {
			return response.JSONError(c, http.StatusConflict, "job_already_completed", err.Error(), nil)
		}
		return response.JSONError(c, http.StatusBadRequest, "add_item_failed", err.Error(), nil)
	}

	return response.JSONSuccess(c, http.StatusCreated, detail)
}

func (h *Handler) RemoveItem(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_id", "Invalid job card ID", nil)
	}

	itemID, err := strconv.ParseInt(c.Param("itemId"), 10, 64)
	if err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_item_id", "Invalid item ID", nil)
	}

	detail, err := h.service.RemoveItem(c.Request().Context(), id, itemID)
	if err != nil {
		if errors.Is(err, ErrJobNotFound) {
			return response.JSONError(c, http.StatusNotFound, "not_found", "Job card not found", nil)
		}
		if errors.Is(err, ErrJobAlreadyCompleted) {
			return response.JSONError(c, http.StatusConflict, "job_already_completed", err.Error(), nil)
		}
		return response.JSONError(c, http.StatusBadRequest, "remove_item_failed", err.Error(), nil)
	}

	return response.JSONSuccess(c, http.StatusOK, detail)
}

func (h *Handler) UpdateJob(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_id", "Invalid job card ID", nil)
	}

	var req UpdateJobCardRequest
	if err := c.Bind(&req); err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_request", "Invalid request body", nil)
	}

	detail, err := h.service.UpdateJob(c.Request().Context(), id, req)
	if err != nil {
		if errors.Is(err, ErrJobNotFound) {
			return response.JSONError(c, http.StatusNotFound, "not_found", "Job card not found", nil)
		}
		return response.JSONError(c, http.StatusBadRequest, "update_failed", err.Error(), nil)
	}

	return response.JSONSuccess(c, http.StatusOK, detail)
}

func (h *Handler) GetPublicTracking(c echo.Context) error {
	token := c.Param("token")
	tracking, err := h.service.GetPublicTracking(c.Request().Context(), token)
	if err != nil {
		if errors.Is(err, ErrJobNotFound) {
			return response.JSONError(c, http.StatusNotFound, "not_found", "Tracking link not found or expired", nil)
		}
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to load job tracking details", nil)
	}

	return response.JSONSuccess(c, http.StatusOK, tracking)
}
