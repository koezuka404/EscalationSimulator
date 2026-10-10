package controller

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"escalator/entity"
	"escalator/usecase"
)

type UserAPI struct {
	link   *usecase.LinkApplicant
	create *usecase.CreateAgent
	list   *usecase.ListUsers
}

func NewUserAPI(link *usecase.LinkApplicant, create *usecase.CreateAgent, list *usecase.ListUsers) *UserAPI {
	return &UserAPI{link: link, create: create, list: list}
}

//利用者の一覧を返す
func (a *UserAPI) List(c echo.Context) error {
	users, err := a.list.Execute(c.Request().Context(), c.Request().Header.Get("Authorization"))
	if err != nil {
		return writeListUsersError(c, err)
	}
	body := make([]userJSON, 0, len(users))
	for _, user := range users {
		body = append(body, toUserJSON(user))
	}
	return c.JSON(http.StatusOK, body)
}

//担当者の登録を受ける
func (a *UserAPI) CreateAgent(c echo.Context) error {
	var body registerJSON
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, messageJSON{Message: "送られた内容を読み取れませんでした。名前、メールアドレス、パスワードを入力してください"})
	}
	user, err := a.create.Execute(c.Request().Context(), c.Request().Header.Get("Authorization"), body.Name, body.Email, body.Password)
	if err != nil {
		return writeCreateAgentError(c, err)
	}
	return c.JSON(http.StatusCreated, toUserJSON(user))
}

//申請者を顧客に結びつける
func (a *UserAPI) LinkCustomer(c echo.Context) error {
	var body linkCustomerJSON
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, messageJSON{Message: "送られた内容を読み取れませんでした。結びつける顧客を選んでください"})
	}
	user, err := a.link.Execute(c.Request().Context(), c.Request().Header.Get("Authorization"), c.Param("id"), body.CustomerID)
	if err != nil {
		return writeLinkError(c, err)
	}
	return c.JSON(http.StatusOK, toUserJSON(user))
}

type linkCustomerJSON struct {
	CustomerID string `json:"customer_id"`
}

func writeCreateAgentError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, entity.ErrUnauthenticated):
		return c.JSON(http.StatusUnauthorized, messageJSON{Message: err.Error()})
	case errors.Is(err, entity.ErrForbidden):
		return c.JSON(http.StatusForbidden, messageJSON{Message: err.Error()})
	case errors.Is(err, entity.ErrInvalidUserName),
		errors.Is(err, entity.ErrInvalidEmail),
		errors.Is(err, entity.ErrInvalidPassword):
		return c.JSON(http.StatusBadRequest, messageJSON{Message: err.Error()})
	case errors.Is(err, entity.ErrEmailTaken):
		return c.JSON(http.StatusConflict, messageJSON{Message: err.Error()})
	default:
		return c.JSON(http.StatusInternalServerError, messageJSON{Message: "担当者を登録できませんでした。しばらくしてから、もう一度試してください"})
	}
}

func writeListUsersError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, entity.ErrUnauthenticated):
		return c.JSON(http.StatusUnauthorized, messageJSON{Message: err.Error()})
	case errors.Is(err, entity.ErrForbidden):
		return c.JSON(http.StatusForbidden, messageJSON{Message: err.Error()})
	default:
		return c.JSON(http.StatusInternalServerError, messageJSON{Message: "利用者の一覧をまとめられませんでした。しばらくしてから、もう一度試してください"})
	}
}

func writeLinkError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, entity.ErrUnauthenticated):
		return c.JSON(http.StatusUnauthorized, messageJSON{Message: err.Error()})
	case errors.Is(err, entity.ErrForbidden), errors.Is(err, entity.ErrNotApplicant):
		return c.JSON(http.StatusForbidden, messageJSON{Message: err.Error()})
	case errors.Is(err, entity.ErrUserNotFound), errors.Is(err, entity.ErrCustomerNotFound):
		return c.JSON(http.StatusNotFound, messageJSON{Message: err.Error()})
	case errors.Is(err, entity.ErrCustomerLink):
		return c.JSON(http.StatusBadRequest, messageJSON{Message: err.Error()})
	default:
		return c.JSON(http.StatusInternalServerError, messageJSON{Message: "申請者を顧客に結びつけられませんでした。しばらくしてから、もう一度試してください"})
	}
}
