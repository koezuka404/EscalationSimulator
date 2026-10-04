package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/labstack/echo/v4"

	"escalator/config"
	"escalator/controller"
	"escalator/infra/postgres"
	"escalator/router"
	"escalator/usecase"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "設定を読み込めませんでした: %v\n", err)
		os.Exit(1)
	}

	db, err := postgres.Open(cfg.DatabaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "データベースに接続できませんでした: %v\n", err)
		os.Exit(1)
	}
	sqlDB, err := db.DB()
	if err != nil {
		fmt.Fprintf(os.Stderr, "データベースに接続できませんでした: %v\n", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	if err := postgres.MigrateCustomers(db); err != nil {
		fmt.Fprintf(os.Stderr, "顧客用のテーブルを作成できませんでした: %v\n", err)
		os.Exit(1)
	}
	if err := postgres.MigrateUsers(db); err != nil {
		fmt.Fprintf(os.Stderr, "利用者用のテーブルを作成できませんでした: %v\n", err)
		os.Exit(1)
	}
	if err := postgres.MigrateSessions(db); err != nil {
		fmt.Fprintf(os.Stderr, "ログイン用のテーブルを作成できませんでした: %v\n", err)
		os.Exit(1)
	}
	if err := postgres.SeedCustomersIfEmpty(context.Background(), postgres.NewCustomerRepository(db)); err != nil {
		fmt.Fprintf(os.Stderr, "サンプルの顧客を登録できませんでした: %v\n", err)
		os.Exit(1)
	}

	if err := dial(cfg.RedisURL); err != nil {
		fmt.Fprintf(os.Stderr, "待ち順の保存先に接続できませんでした: %v\n", err)
		os.Exit(1)
	}

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	users := postgres.NewUserRepository(db)
	sessions := postgres.NewSessionRepository(db)
	router.Customers(e, controller.NewCustomerAPI(usecase.NewCustomers(postgres.NewCustomerRepository(db))))
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

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := e.Shutdown(shutdownCtx); err != nil {
		fmt.Fprintf(os.Stderr, "サーバーを停止できませんでした: %v\n", err)
		os.Exit(1)
	}
}

func dial(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	host := parsed.Host
	if host == "" {
		return fmt.Errorf("接続先のアドレスが空です")
	}
	if _, _, err := net.SplitHostPort(host); err != nil {
		switch parsed.Scheme {
		case "redis", "rediss":
			host = net.JoinHostPort(host, "6379")
		default:
			host = net.JoinHostPort(host, "5432")
		}
	}
	conn, err := net.DialTimeout("tcp", host, 3*time.Second)
	if err != nil {
		return err
	}
	return conn.Close()
}
