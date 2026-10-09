package router

import (
	"github.com/labstack/echo/v4"

	"escalator/controller"
	"escalator/websocket"
)

func Customers(e *echo.Echo, api *controller.CustomerAPI) {
	e.GET("/admin/customers", api.List)
	e.PUT("/admin/customers/:id", api.Update)
}

func Users(e *echo.Echo, api *controller.UserAPI) {
	e.POST("/api/users", api.CreateAgent)
	e.PATCH("/api/users/:id", api.LinkCustomer)
}

func Tickets(e *echo.Echo, api *controller.TicketAPI) {
	e.GET("/api/tickets", api.ListMine)
	e.GET("/api/tickets/:id", api.Show)
	e.POST("/api/tickets", api.Create)
	e.POST("/api/tickets/:id/close", api.Close)
	e.POST("/api/tickets/:id/escalate", api.Escalate)
	e.POST("/api/tickets/:id/release", api.Release)
	e.POST("/api/tickets/:id/comments", api.AddNote)
}

func Agent(e *echo.Echo, api *controller.AgentAPI) {
	e.PATCH("/api/agent/status", api.Update)
}

func Dashboard(e *echo.Echo, api *controller.DashboardAPI) {
	e.GET("/api/admin/dashboard", api.Show)
}

func Live(e *echo.Echo, hub *websocket.Hub) {
	e.GET("/ws", hub.Serve)
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
