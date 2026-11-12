package server

import (
	"fmt"
	"net/http"
	"strings"

	"garage-pos-be/internal/auth"
	"garage-pos-be/internal/pkg/response"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

func AuthMiddleware(jwtSecret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Missing authorization header", nil)
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				return response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Invalid authorization format", nil)
			}

			tokenString := parts[1]
			token, err := jwt.ParseWithClaims(tokenString, &auth.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				return response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Invalid or expired token", nil)
			}

			claims, ok := token.Claims.(*auth.JWTClaims)
			if !ok {
				return response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Invalid token claims", nil)
			}

			c.Set("user", claims)
			return next(c)
		}
	}
}

func RequireRoles(allowedRoles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims, ok := c.Get("user").(*auth.JWTClaims)
			if !ok || claims == nil {
				return response.JSONError(c, http.StatusUnauthorized, "unauthorized", "Unauthorized", nil)
			}

			for _, role := range allowedRoles {
				if claims.Role == role {
					return next(c)
				}
			}

			return response.JSONError(c, http.StatusForbidden, "forbidden", "Insufficient permissions for this action", nil)
		}
	}
}
