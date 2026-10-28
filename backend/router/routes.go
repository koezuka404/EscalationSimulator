package router

import (
	"github.com/labstack/echo/v4"

	"escalator/controller"
	"escalator/middleware"
	"escalator/websocket"
)

func Health(e *echo.Echo, api *controller.HealthAPI) {
	e.GET("/api/health", api.Show)
}

func Customers(e *echo.Echo, api *controller.CustomerAPI) {
	e.GET("/admin/customers", api.List)
	e.PUT("/admin/customers/:id", api.Update)
}

func Users(e *echo.Echo, api *controller.UserAPI, require echo.MiddlewareFunc) {
	e.GET("/api/users", api.List, require)
	e.POST("/api/users", api.CreateAgent, require)
	e.PATCH("/api/users/:id", api.LinkCustomer, require)
	e.POST("/api/users/:id/suspend", api.Suspend, require)
	e.POST("/api/users/:id/resume", api.Resume, require)
	e.PATCH("/api/users/:id/role", api.ChangeRole, require)
}

func Tickets(e *echo.Echo, api *controller.TicketAPI, limits *middleware.Limiter, require echo.MiddlewareFunc) {
	e.GET("/api/tickets", api.ListMine, require)
	e.GET("/api/tickets/:id", api.Show, require)
	e.POST("/api/tickets", api.Create, require, limits.CreateTicket)
	e.POST("/api/tickets/:id/close", api.Close, require)
	e.POST("/api/tickets/:id/escalate", api.Escalate, require, limits.ChangeSeverity)
	e.POST("/api/tickets/:id/release", api.Release, require)
	e.POST("/api/tickets/:id/comments", api.AddNote, require)
}

func Agent(e *echo.Echo, api *controller.AgentAPI, require echo.MiddlewareFunc) {
	e.PATCH("/api/agent/status", api.Update, require)
}

func Demo(e *echo.Echo, api *controller.DemoAPI, limits *middleware.Limiter, require echo.MiddlewareFunc) {
	e.POST("/api/admin/demo/start", api.Start, require, limits.StartDemo)
	e.POST("/api/admin/demo/stop", api.Stop, require)
}

func AdminTickets(e *echo.Echo, api *controller.AdminTicketAPI, require echo.MiddlewareFunc) {
	e.GET("/api/admin/tickets", api.Search, require)
}

func Dashboard(e *echo.Echo, api *controller.DashboardAPI, require echo.MiddlewareFunc) {
	e.GET("/api/admin/dashboard", api.Show, require)
}

func Live(e *echo.Echo, hub *websocket.Hub) {
	e.GET("/ws", hub.Serve)
}

func Queue(e *echo.Echo, api *controller.QueueAPI, limits *middleware.Limiter, require echo.MiddlewareFunc) {
	e.GET("/api/queue", api.List, require)
	e.POST("/api/queue/claim", api.Claim, require, limits.Claim)
}

func Me(e *echo.Echo, api *controller.MeAPI, require echo.MiddlewareFunc) {
	e.GET("/api/me", api.Show, require)
}

func Auth(e *echo.Echo, api *controller.AuthAPI, limits *middleware.Limiter) {
	e.POST("/api/auth/register", api.Register, limits.Register)
	e.POST("/api/auth/login", api.Login, limits.Login)
	e.POST("/api/auth/logout", api.Logout)
	e.POST("/api/auth/refresh", api.Refresh, limits.Refresh)
}
