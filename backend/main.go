package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/labstack/echo/v4"

	"escalator/config"
	"escalator/controller"
	"escalator/db"
	"escalator/job"
	"escalator/redis"
	"escalator/repository"
	"escalator/router"
	"escalator/seed"
	"escalator/usecase"
	"escalator/websocket"
)

//設定を読み、データベースと待ち順につないでサーバーを起動する
func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "設定を読み込めませんでした: %v\n", err)
		os.Exit(1)
	}

	conn, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "データベースに接続できませんでした: %v\n", err)
		os.Exit(1)
	}
	sqlDB, err := conn.DB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "データベースに接続できませんでした: %v\n", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	if err := repository.Migrate(conn); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	if err := seed.SeedCustomersIfEmpty(context.Background(), repository.NewCustomerRepository(conn)); err != nil {
		fmt.Fprintf(os.Stderr, "サンプルの顧客を登録できませんでした: %v\n", err)
		os.Exit(1)
	}

	order, err := redis.Open(cfg.RedisURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "待ち順の保存先に接続できませんでした: %v\n", err)
		os.Exit(1)
	}
	defer order.Close()

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	users := repository.NewUserRepository(conn)
	sessions := repository.NewSessionRepository(conn)
	customers := repository.NewCustomerRepository(conn)
	router.Customers(e, controller.NewCustomerAPI(usecase.NewCustomers(customers)))
	current := usecase.NewCurrentUser(users, []byte(cfg.JWTSecret), cfg.JWTIssuer, cfg.JWTAudience)
	router.Me(e, controller.NewMeAPI(current))
	router.Users(e, controller.NewUserAPI(
		usecase.NewLinkApplicant(users, customers, current),
		usecase.NewCreateAgent(users, current, cfg.BcryptCost),
	))
	tickets := repository.NewTicketRepository(conn)
	statuses := repository.NewAgentStatusRepository(conn)
	dashboard := usecase.NewShowDashboard(tickets, customers, statuses, current)
	notices := websocket.NewHub(current, usecase.NewQueueSnapshot(tickets, customers, users, statuses), dashboard, cfg.CORSAllowedOrigin)
	dashboard.SetNotifier(notices)
	go notices.Run()
	router.Live(e, notices)
	router.Dashboard(e, controller.NewDashboardAPI(dashboard))
	router.AdminTickets(e, controller.NewAdminTicketAPI(usecase.NewSearchTickets(tickets, customers, users, current)))
	router.Tickets(e, controller.NewTicketAPI(
		usecase.NewCreateTicket(tickets, customers, current, order, notices),
		usecase.NewCloseTicket(tickets, order, current, notices),
		usecase.NewChangeSeverity(tickets, customers, current, order, notices),
		usecase.NewReturnTicketToQueue(tickets, customers, order, current, notices),
		usecase.NewListMyTickets(tickets, customers, users, current),
		usecase.NewShowTicket(tickets, customers, users, current),
		usecase.NewAddWorkNote(tickets, current),
	))
	router.Agent(e, controller.NewAgentAPI(usecase.NewChangeAgentStatus(users, customers, statuses, order, current, notices)))
	router.Queue(e, controller.NewQueueAPI(
		usecase.NewListWaitingTickets(tickets, customers, current),
		usecase.NewClaimNextTicket(repository.NewClaimRepository(conn), order, current, notices),
	))
	jobCtx, stopJob := context.WithCancel(context.Background())
	defer stopJob()
	go job.NewRecalcPriority(
		usecase.NewRecalcOpenScores(tickets, customers, order, notices),
		dashboard,
		order,
		cfg.PriorityRecalc,
	).Run(jobCtx)
	go job.NewReleaseDisconnectedAgent(
		usecase.NewReleaseDisconnectedAgents(users, customers, statuses, order, notices, notices, cfg.AgentDisconnectGrace),
		cfg.PriorityRecalc,
	).Run(jobCtx)

	router.Auth(e, controller.NewAuthAPI(
		usecase.NewSignUp(users, cfg.BcryptCost),
		usecase.NewLogIn(users, sessions, []byte(cfg.JWTSecret), cfg.JWTIssuer, cfg.JWTAudience, cfg.AccessTokenTTL, cfg.RefreshTokenTTL, cfg.LoginMaxFailures, cfg.LoginLock),
		usecase.NewLogOut(sessions),
		usecase.NewRefresh(users, sessions, []byte(cfg.JWTSecret), cfg.JWTIssuer, cfg.JWTAudience, cfg.AccessTokenTTL, cfg.RefreshTokenTTL),
		cfg.RefreshTokenCookieName,
		cfg.CSRFTokenCookieName,
		cfg.CookieSecure,
		int(cfg.RefreshTokenTTL.Seconds()),
	))

	go func() {
		if err := e.Start(fmt.Sprintf(":%d", cfg.HTTPPort)); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(os.Stderr, "サーバーを起動できませんでした: %v\n", err)
			os.Exit(1)
		}
	}()

	fmt.Printf("環境を起動しました env=%s port=%d\n", cfg.Environment, cfg.HTTPPort)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
	stopJob()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		fmt.Fprintf(os.Stderr, "サーバーを停止できませんでした: %v\n", err)
		os.Exit(1)
	}
}
