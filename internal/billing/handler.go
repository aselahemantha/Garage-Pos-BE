package billing

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

func (h *Handler) IssueInvoice(c echo.Context) error {
	var req IssueInvoiceRequest
	if err := c.Bind(&req); err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_request", "Invalid request body", nil)
	}

	claims, _ := c.Get("user").(*auth.JWTClaims)
	var userID int64 = 1
	if claims != nil {
		userID = claims.UserID
	}

	detail, err := h.service.IssueInvoice(c.Request().Context(), userID, req)
	if err != nil {
		if errors.Is(err, ErrInvoiceAlreadyIssued) {
			return response.JSONError(c, http.StatusConflict, "invoice_exists", err.Error(), nil)
		}
		return response.JSONError(c, http.StatusBadRequest, "issue_failed", err.Error(), nil)
	}

	return response.JSONSuccess(c, http.StatusCreated, detail)
}

func (h *Handler) GetInvoice(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_id", "Invalid invoice ID", nil)
	}

	detail, err := h.service.GetInvoice(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, ErrInvoiceNotFound) {
			return response.JSONError(c, http.StatusNotFound, "not_found", "Invoice not found", nil)
		}
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", err.Error(), nil)
	}

	return response.JSONSuccess(c, http.StatusOK, detail)
}

func (h *Handler) ListInvoices(c echo.Context) error {
	var statusPtr *string
	if s := c.QueryParam("status"); s != "" {
		statusPtr = &s
	}
	var searchPtr *string
	if q := c.QueryParam("search"); q != "" {
		searchPtr = &q
	}

	limit := int32(20)
	if l := c.QueryParam("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = int32(parsed)
		}
	}
	offset := int32(0)
	if o := c.QueryParam("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = int32(parsed)
		}
	}

	resp, err := h.service.ListInvoices(c.Request().Context(), statusPtr, searchPtr, limit, offset)
	if err != nil {
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", err.Error(), nil)
	}

	return response.JSONSuccess(c, http.StatusOK, resp)
}

func (h *Handler) RecordPayment(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_id", "Invalid invoice ID", nil)
	}

	var req RecordPaymentRequest
	if err := c.Bind(&req); err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_request", "Invalid request body", nil)
	}

	claims, _ := c.Get("user").(*auth.JWTClaims)
	var userID int64 = 1
	if claims != nil {
		userID = claims.UserID
	}

	detail, err := h.service.RecordPayment(c.Request().Context(), userID, id, req)
	if err != nil {
		if errors.Is(err, ErrInvoiceNotFound) {
			return response.JSONError(c, http.StatusNotFound, "not_found", "Invoice not found", nil)
		}
		if errors.Is(err, ErrInsufficientTendered) || errors.Is(err, ErrInvalidPaymentAmount) || errors.Is(err, ErrInvoiceAlreadyPaid) || errors.Is(err, ErrInvoiceVoided) {
			return response.JSONError(c, http.StatusBadRequest, "payment_failed", err.Error(), nil)
		}
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", err.Error(), nil)
	}

	return response.JSONSuccess(c, http.StatusOK, detail)
}

func (h *Handler) VoidInvoice(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_id", "Invalid invoice ID", nil)
	}

	var req VoidInvoiceRequest
	if err := c.Bind(&req); err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_request", "Invalid request body", nil)
	}

	claims, _ := c.Get("user").(*auth.JWTClaims)
	var userID int64 = 1
	userRole := "owner"
	if claims != nil {
		userID = claims.UserID
		userRole = claims.Role
	}

	detail, err := h.service.VoidInvoice(c.Request().Context(), userID, userRole, id, req)
	if err != nil {
		if errors.Is(err, ErrUnauthorizedRole) {
			return response.JSONError(c, http.StatusForbidden, "unauthorized", err.Error(), nil)
		}
		if errors.Is(err, ErrInvoiceNotFound) {
			return response.JSONError(c, http.StatusNotFound, "not_found", "Invoice not found", nil)
		}
		return response.JSONError(c, http.StatusBadRequest, "void_failed", err.Error(), nil)
	}

	return response.JSONSuccess(c, http.StatusOK, detail)
}

func (h *Handler) ApplyDiscount(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_id", "Invalid invoice ID", nil)
	}

	var req ApplyDiscountRequest
	if err := c.Bind(&req); err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_request", "Invalid request body", nil)
	}

	claims, _ := c.Get("user").(*auth.JWTClaims)
	var userID int64 = 1
	userRole := "owner"
	if claims != nil {
		userID = claims.UserID
		userRole = claims.Role
	}

	detail, err := h.service.ApplyDiscount(c.Request().Context(), userID, userRole, id, req)
	if err != nil {
		if errors.Is(err, ErrUnauthorizedRole) {
			return response.JSONError(c, http.StatusForbidden, "unauthorized", err.Error(), nil)
		}
		if errors.Is(err, ErrInvoiceNotFound) {
			return response.JSONError(c, http.StatusNotFound, "not_found", "Invoice not found", nil)
		}
		return response.JSONError(c, http.StatusBadRequest, "apply_discount_failed", err.Error(), nil)
	}

	return response.JSONSuccess(c, http.StatusOK, detail)
}

// Cash Session Endpoints

func (h *Handler) GetActiveCashSession(c echo.Context) error {
	session, err := h.service.GetActiveCashSession(c.Request().Context())
	if err != nil {
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", err.Error(), nil)
	}
	return response.JSONSuccess(c, http.StatusOK, session)
}

func (h *Handler) OpenCashSession(c echo.Context) error {
	var req OpenCashSessionRequest
	if err := c.Bind(&req); err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_request", "Invalid request body", nil)
	}

	claims, _ := c.Get("user").(*auth.JWTClaims)
	var userID int64 = 1
	if claims != nil {
		userID = claims.UserID
	}

	session, err := h.service.OpenCashSession(c.Request().Context(), userID, req)
	if err != nil {
		if errors.Is(err, ErrSessionAlreadyOpen) {
			return response.JSONError(c, http.StatusConflict, "session_already_open", err.Error(), nil)
		}
		return response.JSONError(c, http.StatusBadRequest, "open_failed", err.Error(), nil)
	}

	return response.JSONSuccess(c, http.StatusCreated, session)
}

func (h *Handler) CloseCashSession(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_id", "Invalid cash session ID", nil)
	}

	var req CloseCashSessionRequest
	if err := c.Bind(&req); err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_request", "Invalid request body", nil)
	}

	session, err := h.service.CloseCashSession(c.Request().Context(), id, req)
	if err != nil {
		return response.JSONError(c, http.StatusBadRequest, "close_failed", err.Error(), nil)
	}

	return response.JSONSuccess(c, http.StatusOK, session)
}

func (h *Handler) ListCashSessions(c echo.Context) error {
	limit := int32(20)
	if l := c.QueryParam("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = int32(parsed)
		}
	}
	offset := int32(0)
	if o := c.QueryParam("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = int32(parsed)
		}
	}

	sessions, err := h.service.ListCashSessions(c.Request().Context(), limit, offset)
	if err != nil {
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", err.Error(), nil)
	}

	return response.JSONSuccess(c, http.StatusOK, sessions)
}
