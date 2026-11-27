package middleware

import (
	"context"

	"github.com/labstack/echo/v4"

	"escalator/scope"
)

//プロセスの生存期間を、このリクエストの続きから読めるように残す
func RememberProcess(life context.Context) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := scope.WithProcess(c.Request().Context(), life)
			c.SetRequest(c.Request().WithContext(ctx))
			return next(c)
		}
	}
}
