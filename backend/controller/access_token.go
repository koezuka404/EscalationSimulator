package controller

import (
	"github.com/labstack/echo/v4"

	"escalator/httpx"
)

func accessToken(c echo.Context) string {
	return httpx.BearerToken(c.Request().Header.Get("Authorization"))
}
