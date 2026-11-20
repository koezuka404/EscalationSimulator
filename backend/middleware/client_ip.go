package middleware

import (
	"github.com/labstack/echo/v4"

	"escalator/scope"
)

//接続元を、このリクエストの続きから読めるように残す
func RememberClientIP() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := scope.WithClientIP(c.Request().Context(), clientIP(c.Request()))
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}
