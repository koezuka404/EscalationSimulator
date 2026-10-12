package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

//許可した画面からの通信だけ、ブラウザに通す
func AllowOrigin(origin string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			requestOrigin := c.Request().Header.Get("Origin")
			if origin != "" && requestOrigin == origin {
				header := c.Response().Header()
				header.Set("Access-Control-Allow-Origin", origin)
				header.Set("Access-Control-Allow-Credentials", "true")
				header.Add("Vary", "Origin")
				header.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-CSRF-Token, X-Request-ID")
				header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			}
			if c.Request().Method == http.MethodOptions && origin != "" && requestOrigin == origin {
				return c.NoContent(http.StatusNoContent)
			}
			return next(c)
		}
	}
}
