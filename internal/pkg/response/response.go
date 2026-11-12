package response

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type ErrorDetail struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type SuccessResponse[T any] struct {
	Data T `json:"data"`
}

type Meta struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

type ListResponse[T any] struct {
	Data T    `json:"data"`
	Meta Meta `json:"meta"`
}

func JSONSuccess[T any](c echo.Context, status int, data T) error {
	return c.JSON(status, SuccessResponse[T]{Data: data})
}

func JSONList[T any](c echo.Context, status int, data T, meta Meta) error {
	return c.JSON(status, ListResponse[T]{
		Data: data,
		Meta: meta,
	})
}

func JSONError(c echo.Context, status int, code, message string, fields map[string]string) error {
	return c.JSON(status, ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
			Fields:  fields,
		},
	})
}

func CentralErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	he, ok := err.(*echo.HTTPError)
	if ok {
		msg, _ := he.Message.(string)
		if msg == "" {
			msg = http.StatusText(he.Code)
		}
		_ = JSONError(c, he.Code, "http_error", msg, nil)
		return
	}

	c.Logger().Error(err)
	_ = JSONError(c, http.StatusInternalServerError, "internal_error", "An internal server error occurred", nil)
}
