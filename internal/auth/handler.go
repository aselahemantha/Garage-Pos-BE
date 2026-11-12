package auth

import (
	"errors"
	"net/http"
	"time"

	"garage-pos-be/internal/pkg/response"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

const RefreshCookieName = "refresh_token"

type Handler struct {
	service   *Service
	validator *validator.Validate
	isProd    bool
}

func NewHandler(service *Service, v *validator.Validate, isProd bool) *Handler {
	return &Handler{
		service:   service,
		validator: v,
		isProd:    isProd,
	}
}

func (h *Handler) Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return response.JSONError(c, http.StatusBadRequest, "invalid_request", "Invalid request body", nil)
	}

	if err := h.validator.Struct(&req); err != nil {
		return response.JSONError(c, http.StatusBadRequest, "validation_failed", err.Error(), nil)
	}

	res, refreshToken, err := h.service.Login(c.Request().Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) || errors.Is(err, ErrUserInactive) {
			return response.JSONError(c, http.StatusUnauthorized, "unauthorized", err.Error(), nil)
		}
		return response.JSONError(c, http.StatusInternalServerError, "internal_error", "Failed to login", nil)
	}

	h.setRefreshCookie(c, refreshToken, h.service.cfg.RefreshTokenTTL)
	return response.JSONSuccess(c, http.StatusOK, res)
}

func (h *Handler) Refresh(c echo.Context) error {
	cookie, err := c.Cookie(RefreshCookieName)
	if err != nil || cookie.Value == "" {
		return response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Missing refresh token", nil)
	}

	newAccessToken, newRefreshToken, err := h.service.Refresh(c.Request().Context(), cookie.Value)
	if err != nil {
		h.clearRefreshCookie(c)
		return response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Invalid or expired session", nil)
	}

	h.setRefreshCookie(c, newRefreshToken, h.service.cfg.RefreshTokenTTL)
	return response.JSONSuccess(c, http.StatusOK, RefreshResponse{
		AccessToken: newAccessToken,
	})
}

func (h *Handler) Logout(c echo.Context) error {
	cookie, _ := c.Cookie(RefreshCookieName)
	if cookie != nil && cookie.Value != "" {
		_ = h.service.Logout(c.Request().Context(), cookie.Value)
	}
	h.clearRefreshCookie(c)
	return response.JSONSuccess(c, http.StatusOK, map[string]string{"message": "Logged out successfully"})
}

func (h *Handler) Me(c echo.Context) error {
	claims, ok := c.Get("user").(*JWTClaims)
	if !ok || claims == nil {
		return response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Unauthorized", nil)
	}

	user, err := h.service.GetUserByID(c.Request().Context(), claims.UserID)
	if err != nil {
		return response.JSONError(c, http.StatusNotFound, "user_not_found", "User not found", nil)
	}

	return response.JSONSuccess(c, http.StatusOK, user)
}

func (h *Handler) setRefreshCookie(c echo.Context, token string, ttl time.Duration) {
	cookie := &http.Cookie{
		Name:     RefreshCookieName,
		Value:    token,
		Path:     "/api/v1/auth",
		Expires:  time.Now().Add(ttl),
		HttpOnly: true,
		Secure:   h.isProd,
		SameSite: http.SameSiteStrictMode,
	}
	c.SetCookie(cookie)
}

func (h *Handler) clearRefreshCookie(c echo.Context) {
	cookie := &http.Cookie{
		Name:     RefreshCookieName,
		Value:    "",
		Path:     "/api/v1/auth",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   h.isProd,
		SameSite: http.SameSiteStrictMode,
	}
	c.SetCookie(cookie)
}
