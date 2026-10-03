package controller

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"escalator/domain"
	"escalator/usecase"
)

// AuthAPI は会員登録を HTTP で受ける入口。
type AuthAPI struct {
	signUp *usecase.SignUp
}

func NewAuthAPI(signUp *usecase.SignUp) *AuthAPI {
	return &AuthAPI{signUp: signUp}
}

func (a *AuthAPI) Register(c echo.Context) error {
	var body registerJSON
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, messageJSON{Message: "送られた内容を読み取れませんでした。名前、メールアドレス、パスワードを入力してください"})
	}
	user, err := a.signUp.Execute(c.Request().Context(), body.Name, body.Email, body.Password)
	if err != nil {
		return writeSignUpError(c, err)
	}
	return c.JSON(http.StatusCreated, toUserJSON(user))
}

type registerJSON struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type userJSON struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	Role       string `json:"role"`
	CustomerID string `json:"customer_id"`
	Status     string `json:"status"`
}

func toUserJSON(user domain.User) userJSON {
	return userJSON{
		ID:         user.ID,
		Name:       user.Name,
		Email:      user.Email,
		Role:       string(user.Role),
		CustomerID: user.CustomerID,
		Status:     string(user.Status),
	}
}

func writeSignUpError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidUserName),
		errors.Is(err, domain.ErrInvalidEmail),
		errors.Is(err, domain.ErrInvalidPassword):
		return c.JSON(http.StatusBadRequest, messageJSON{Message: err.Error()})
	case errors.Is(err, domain.ErrEmailTaken):
		return c.JSON(http.StatusConflict, messageJSON{Message: err.Error()})
	default:
		return c.JSON(http.StatusInternalServerError, messageJSON{Message: "登録できませんでした。しばらくしてから、もう一度試してください"})
	}
}
