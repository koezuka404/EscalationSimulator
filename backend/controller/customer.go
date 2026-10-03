package controller

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"escalator/domain"
	"escalator/usecase"
)

// CustomerAPI は顧客の一覧と更新を HTTP で受ける入口。
type CustomerAPI struct {
	customers *usecase.Customers
}

func NewCustomerAPI(customers *usecase.Customers) *CustomerAPI {
	return &CustomerAPI{customers: customers}
}

func (a *CustomerAPI) List(c echo.Context) error {
	customers, err := a.customers.List(c.Request().Context())
	if err != nil {
		return writeCustomerError(c, err)
	}
	body := make([]customerJSON, 0, len(customers))
	for _, customer := range customers {
		body = append(body, toCustomerJSON(customer))
	}
	return c.JSON(http.StatusOK, body)
}

func (a *CustomerAPI) Update(c echo.Context) error {
	var body updateCustomerJSON
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, messageJSON{Message: "送られた内容を読み取れませんでした。顧客名、プラン、目標時間を入力してください"})
	}
	customer, err := a.customers.Update(
		c.Request().Context(),
		c.Param("id"),
		body.Name,
		domain.Plan(body.Plan),
		body.SLAMinutes,
	)
	if err != nil {
		return writeCustomerError(c, err)
	}
	return c.JSON(http.StatusOK, toCustomerJSON(customer))
}

type customerJSON struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Plan       string `json:"plan"`
	SLAMinutes int    `json:"sla_minutes"`
}

type updateCustomerJSON struct {
	Name       string `json:"name"`
	Plan       string `json:"plan"`
	SLAMinutes int    `json:"sla_minutes"`
}

type messageJSON struct {
	Message string `json:"message"`
}

func toCustomerJSON(customer domain.Customer) customerJSON {
	return customerJSON{
		ID:         customer.ID,
		Name:       customer.Name,
		Plan:       string(customer.Plan),
		SLAMinutes: customer.SLAMinutes,
	}
}

func writeCustomerError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, domain.ErrCustomerNotFound):
		return c.JSON(http.StatusNotFound, messageJSON{Message: err.Error()})
	case errors.Is(err, domain.ErrInvalidCustomerName),
		errors.Is(err, domain.ErrInvalidPlan),
		errors.Is(err, domain.ErrInvalidSLAMinutes):
		return c.JSON(http.StatusBadRequest, messageJSON{Message: err.Error()})
	default:
		return c.JSON(http.StatusInternalServerError, messageJSON{Message: "顧客を保存できませんでした。しばらくしてから、もう一度試してください"})
	}
}
