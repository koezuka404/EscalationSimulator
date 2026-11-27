package controller

import (
	"github.com/labstack/echo/v4"

	"escalator/middleware"
)

func accessToken(c echo.Context) string {
	return middleware.BearerToken(c.Request().Header.Get("Authorization"))
}
