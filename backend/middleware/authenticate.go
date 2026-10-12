package middleware

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"

	"escalator/entity"
)

//ログイン用トークンから利用者を確かめ、通らなければ先へ進まない
func RequireLogin(lookup func(context.Context, string) (entity.User, error)) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if _, err := lookup(c.Request().Context(), c.Request().Header.Get("Authorization")); err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"message": entity.ErrUnauthenticated.Error(),
				})
			}
			return next(c)
		}
	}
}
