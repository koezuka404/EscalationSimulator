package middleware

import (
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

const RequestIDHeader = "X-Request-ID"

const requestIDKey = "request_id"

//操作ごとに番号を付け、応答にも載せる
func RequestID() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			id := strings.TrimSpace(c.Request().Header.Get(RequestIDHeader))
			if !acceptableRequestID(id) {
				id = uuid.NewString()
			}
			c.Set(requestIDKey, id)
			c.Response().Header().Set(RequestIDHeader, id)
			return next(c)
		}
	}
}

func RequestIDFrom(c echo.Context) string {
	id, _ := c.Get(requestIDKey).(string)
	return id
}

func acceptableRequestID(id string) bool {
	if id == "" || len(id) > 80 {
		return false
	}
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_' || r == '.':
		default:
			return false
		}
	}
	return true
}
