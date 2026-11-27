package controller

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"escalator/rule"
)

type MeAPI struct {
	current currentUserUsecase
}

func NewMeAPI(current currentUserUsecase) *MeAPI {
	return &MeAPI{current: current}
}

//今ログインしている人を返す
func (a *MeAPI) Show(c echo.Context) error {
	user, err := a.current.Execute(c.Request().Context(), accessToken(c))
	if err != nil {
		if errors.Is(err, rule.ErrUnauthenticated) {
			return c.JSON(http.StatusUnauthorized, messageJSON{Message: err.Error()})
		}
		return c.JSON(http.StatusInternalServerError, messageJSON{Message: "ログイン中の利用者を確認できませんでした。もう一度ログインしてください"})
	}
	return c.JSON(http.StatusOK, toUserJSON(user))
}
