package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"escalator/config"
	"escalator/controller"
	"escalator/crypto"
	"escalator/db"
	"escalator/job"
	"escalator/middleware"
	"escalator/redis"
	"escalator/repository"
	"escalator/router"
	"escalator/seed"
	"escalator/usecase"
	"escalator/websocket"
)

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

	order, err := redis.Open(cfg.RedisURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "待ち順の保存先に接続できませんでした: %v\n", err)
		os.Exit(1)
	}
	defer order.Close()

	users := repository.NewUserRepository(conn)
	sessions := repository.NewSessionRepository(conn)
	customers := repository.NewCustomerRepository(conn)
	tickets := repository.NewTicketRepository(conn)
	statuses := repository.NewAgentStatusRepository(conn)
	passwords := crypto.NewBcrypt(cfg.BcryptCost)
	if err := seed.SeedCustomersIfEmpty(context.Background(), customers); err != nil {
		fmt.Fprintf(os.Stderr, "サンプルの顧客を登録できませんでした: %v\n", err)
		os.Exit(1)
	}
	if err := seed.SeedUsersIfEmpty(context.Background(), users, customers, cfg.SampleUserPassword, passwords); err != nil {
		fmt.Fprintf(os.Stderr, "サンプルの利用者を登録できませんでした: %v\n", err)
		os.Exit(1)
	}

	jobCtx, stopJob := context.WithCancel(context.Background())
	defer stopJob()

	access := crypto.NewAccessTokens([]byte(cfg.JWTSecret), cfg.JWTIssuer, cfg.JWTAudience, cfg.AccessTokenTTL)
	secrets := crypto.Secrets{}
	policy := usecase.SessionPolicy{RefreshTTL: cfg.RefreshTokenTTL, MaxFailures: cfg.LoginMaxFailures, Lock: cfg.LoginLock}
	audit := usecase.NewAuditLog(repository.NewAuditRepository(conn))
	current := usecase.NewCurrentUser(users, access)
	limits := middleware.NewLimiter(redis.NewGate(order), access, secrets, cfg.RefreshTokenCookieName, middleware.UserIDFromRefresh(sessions), audit)
	requireLogin := middleware.RequireLogin(current.Execute)
	dashboard := usecase.NewShowDashboard(tickets, customers, statuses, current)
	notices := websocket.NewHub(current, usecase.NewQueueSnapshot(tickets, customers, users, statuses), dashboard, usecase.NewConnectionLog(repository.NewConnectionLogRepository(conn)), cfg.CORSAllowedOrigin)
	shared := websocket.NewRemoteNotices(notices, redis.NewBus(order), cfg.RedisPubSubChannelPrefix)
	createTicket := usecase.NewCreateTicket(tickets, customers, current, order, shared, audit)
	demo := usecase.NewRunDemo(users, customers, repository.NewDemoRepository(conn), createTicket, shared, audit)
	scores := usecase.NewRecalcOpenScores(tickets, customers, order, shared)

	if err := job.RecoverDemo(context.Background(), demo); err != nil {
		fmt.Fprintf(os.Stderr, "前回のデモを止めた状態にできませんでした: %v\n", err)
		os.Exit(1)
	}

	e := router.New(router.Server{
		Origin:       cfg.CORSAllowedOrigin,
		Process:      jobCtx,
		Tokens:       access,
		Records:      usecase.NewRequestLog(repository.NewRequestLogRepository(conn)),
		Limiter:      limits,
		RequireLogin: requireLogin,
		CSRFCookie:   cfg.CSRFTokenCookieName,
	}, router.APIs{
		Health:       controller.NewHealthAPI(usecase.NewCheckHealth(db.NewProbe(conn), order)),
		Auth:         controller.NewAuthAPI(usecase.NewSignUp(users, passwords), usecase.NewLogIn(users, repository.NewLoginAttemptRepository(conn), sessions, passwords, access, secrets, policy, audit), usecase.NewLogOut(sessions, secrets, audit), usecase.NewRefresh(users, sessions, access, secrets, policy, notices, audit), cfg.RefreshTokenCookieName, cfg.CSRFTokenCookieName, cfg.CookieSecure, int(cfg.RefreshTokenTTL.Seconds())),
		Customers:    controller.NewCustomerAPI(usecase.NewCustomers(customers, current, audit)),
		Me:           controller.NewMeAPI(current),
		Users:        controller.NewUserAPI(usecase.NewLinkApplicant(users, customers, current, audit), usecase.NewCreateAgent(users, current, passwords), usecase.NewListUsers(users, current), usecase.NewSuspendUser(users, current, notices, audit), usecase.NewResumeUser(users, current), usecase.NewDeleteUser(users, customers, statuses, order, current, shared, audit), usecase.NewChangeRole(users, customers, statuses, order, current, shared, audit)),
		Tickets:      controller.NewTicketAPI(createTicket, usecase.NewCloseTicket(tickets, order, current, shared, audit), usecase.NewChangeSeverity(tickets, customers, current, order, shared, audit), usecase.NewReturnTicketToQueue(tickets, customers, order, current, shared, audit), usecase.NewListMyTickets(tickets, customers, users, current), usecase.NewShowTicket(tickets, customers, users, current), usecase.NewAddWorkNote(tickets, current)),
		Queue:        controller.NewQueueAPI(usecase.NewListWaitingTickets(tickets, customers, current), usecase.NewClaimNextTicket(repository.NewClaimRepository(conn), order, current, shared, scores, order, cfg.PriorityRecalc, audit)),
		Agent:        controller.NewAgentAPI(usecase.NewChangeAgentStatus(users, customers, statuses, order, current, shared, audit)),
		Dashboard:    controller.NewDashboardAPI(dashboard),
		AdminTickets: controller.NewAdminTicketAPI(usecase.NewSearchTickets(tickets, customers, users, current)),
		Demo:         controller.NewDemoAPI(demo),
		Notices:      notices,
	})

	go websocket.Serve(jobCtx, notices, shared)
	go job.NewRecalcPriority(scores, dashboard, shared, order, cfg.PriorityRecalc).Run(jobCtx)
	go job.NewReleaseDisconnectedAgent(usecase.NewReleaseDisconnectedAgents(users, customers, statuses, order, notices, shared, cfg.AgentDisconnectGrace, audit), cfg.PriorityRecalc).Run(jobCtx)

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
