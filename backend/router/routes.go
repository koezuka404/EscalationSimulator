package router

import (
	"github.com/labstack/echo/v4"

	"escalator/controller"
)

func Customers(e *echo.Echo, api *controller.CustomerAPI) {
	e.GET("/admin/customers", api.List)
	e.PUT("/admin/customers/:id", api.Update)
}
