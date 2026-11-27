package router

import (
	"context"

	"github.com/labstack/echo/v4"

	"escalator/controller"
	"escalator/middleware"
	"escalator/port"
	"escalator/websocket"
)

type Server struct {
	Origin       string
	Process      context.Context
	Tokens       port.TokenVerifier
	Records      middleware.RequestRecorder
	Limiter      *middleware.Limiter
	RequireLogin echo.MiddlewareFunc
	CSRFCookie   string
}

type APIs struct {
	Health       *controller.HealthAPI
	Auth         *controller.AuthAPI
	Customers    *controller.CustomerAPI
	Me           *controller.MeAPI
	Users        *controller.UserAPI
	Tickets      *controller.TicketAPI
	Queue        *controller.QueueAPI
	Agent        *controller.AgentAPI
	Dashboard    *controller.DashboardAPI
	AdminTickets *controller.AdminTicketAPI
	Demo         *controller.DemoAPI
	Notices      *websocket.Hub
}

//Echoを作り、共通処理と経路を載せる
func New(server Server, api APIs) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(middleware.RequestID())
	e.Use(middleware.RememberClientIP())
	e.Use(middleware.RememberProcess(server.Process))
	e.Use(middleware.AccessLog(server.Tokens, server.Records))
	e.Use(middleware.Recover())
	e.Use(middleware.AllowOrigin(server.Origin))
	Health(e, api.Health)
	Customers(e, api.Customers, server.RequireLogin)
	Me(e, api.Me, server.RequireLogin)
	Users(e, api.Users, server.RequireLogin)
	Tickets(e, api.Tickets, server.Limiter, server.RequireLogin)
	Agent(e, api.Agent, server.RequireLogin)
	Demo(e, api.Demo, server.Limiter, server.RequireLogin)
	AdminTickets(e, api.AdminTickets, server.RequireLogin)
	Dashboard(e, api.Dashboard, server.RequireLogin)
	Live(e, api.Notices)
	Queue(e, api.Queue, server.Limiter, server.RequireLogin)
	Auth(e, api.Auth, server.Limiter, middleware.CheckCSRF(server.CSRFCookie))
	return e
}
