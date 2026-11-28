package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
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
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "アプリケーションを終了します: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("設定を読み込めませんでした: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	passwords := crypto.NewBcrypt(cfg.BcryptCost)
	conn, err := repository.Connect(cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close(conn)

	order, err := redis.Open(cfg.RedisURL)
	if err != nil {
		return fmt.Errorf("待ち順の保存先に接続できませんでした: %w", err)
	}
	defer order.Close()

	users := repository.NewUserRepository(conn)
	sessions := repository.NewSessionRepository(conn)
	customers := repository.NewCustomerRepository(conn)
	if cfg.SeedSampleData {
		if err := seed.Run(ctx, customers, users, cfg.SampleUserPassword, passwords); err != nil {
			return err
		}
	}
	tickets := repository.NewTicketRepository(conn)
	statuses := repository.NewAgentStatusRepository(conn)
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

	e := router.New(router.Server{
		Origin:       cfg.CORSAllowedOrigin,
		Process:      ctx,
		Tokens:       access,
		Records:      usecase.NewRequestLog(repository.NewRequestLogRepository(conn)),
		Limiter:      limits,
		RequireLogin: requireLogin,
		CSRFCookie:   cfg.CSRFTokenCookieName,
	}, router.APIs{
		Health:       controller.NewHealthAPI(usecase.NewCheckHealth(db.NewProbe(conn), order)),
		Auth:         controller.NewAuthAPI(usecase.NewSignUp(users, passwords), usecase.NewLogIn(users, repository.NewLoginAttemptRepository(conn), sessions, passwords, access, secrets, policy, audit), usecase.NewLogOut(sessions, secrets, audit), usecase.NewRefresh(users, sessions, access, secrets, policy, notices, audit), cfg.RefreshTokenCookieName, cfg.CSRFTokenCookieName, cfg.CookieSecure, cfg.RefreshTokenMaxAge()),
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

	waitJobs, err := job.Start(ctx, demo, job.NewRecalcPriority(scores, dashboard, shared, order, cfg.PriorityRecalc), job.NewReleaseDisconnectedAgent(usecase.NewReleaseDisconnectedAgents(users, customers, statuses, order, notices, shared, cfg.AgentDisconnectGrace, audit), cfg.PriorityRecalc))
	if err != nil {
		return err
	}
	var sockets sync.WaitGroup
	sockets.Add(1)
	go func() {
		defer sockets.Done()
		websocket.Serve(ctx, notices, shared)
	}()
	waitBackground := func() {
		waitJobs()
		sockets.Wait()
	}

	serverErr := make(chan error, 1)
	go func() {
		err := e.Start(fmt.Sprintf(":%d", cfg.HTTPPort))
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- fmt.Errorf("サーバーを起動できませんでした: %w", err)
		}
	}()

	fmt.Printf("環境を起動しました env=%s port=%d\n", cfg.Environment, cfg.HTTPPort)
	select {
	case err := <-serverErr:
		stop()
		waitBackground()
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = e.Shutdown(shutdownCtx)
	waitBackground()
	if err != nil {
		return fmt.Errorf("サーバーを停止できませんでした: %w", err)
	}
	return nil
}
