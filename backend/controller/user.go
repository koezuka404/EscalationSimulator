package controller

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"escalator/entity"
	"escalator/usecase"
)

type UserAPI struct {
	link *usecase.LinkApplicant
}

func NewUserAPI(link *usecase.LinkApplicant) *UserAPI {
	return &UserAPI{link: link}
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
