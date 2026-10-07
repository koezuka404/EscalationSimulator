package router

import (
	"github.com/labstack/echo/v4"

	"escalator/controller"
)

func Customers(e *echo.Echo, api *controller.CustomerAPI) {
	e.GET("/admin/customers", api.List)
	e.PUT("/admin/customers/:id", api.Update)
}

func Users(e *echo.Echo, api *controller.UserAPI) {
	e.PATCH("/api/users/:id", api.LinkCustomer)
}

func Tickets(e *echo.Echo, api *controller.TicketAPI) {
	e.GET("/api/tickets", api.ListMine)
	e.POST("/api/tickets", api.Create)
	e.POST("/api/tickets/:id/close", api.Close)
	e.POST("/api/tickets/:id/escalate", api.Escalate)
	e.POST("/api/tickets/:id/release", api.Release)
}

func Queue(e *echo.Echo, api *controller.QueueAPI) {
	e.GET("/api/queue", api.List)
	e.POST("/api/queue/claim", api.Claim)
}

func Me(e *echo.Echo, api *controller.MeAPI) {
	e.GET("/api/me", api.Show)
}

func Auth(e *echo.Echo, api *controller.AuthAPI) {
	e.POST("/api/auth/register", api.Register)
	e.POST("/api/auth/login", api.Login)
	e.POST("/api/auth/logout", api.Logout)
	e.POST("/api/auth/refresh", api.Refresh)
}
