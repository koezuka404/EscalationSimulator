package controller

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"escalator/usecase"
)

type HealthAPI struct {
	check *usecase.CheckHealth
}

func NewHealthAPI(check *usecase.CheckHealth) *HealthAPI {
	return &HealthAPI{check: check}
}

//データベースと待ち順の保存先が応答するかを返す
func (a *HealthAPI) Show(c echo.Context) error {
	if err := a.check.Execute(c.Request().Context()); err != nil {
		return c.JSON(http.StatusServiceUnavailable, messageJSON{Message: err.Error()})
	}
	return c.JSON(http.StatusOK, messageJSON{Message: "動いています"})
}
