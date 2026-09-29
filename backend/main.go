package main

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"os/signal"
	"time"

	"escalator/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "設定を読めません: %v\n", err)
		os.Exit(1)
	}

	if err := dial(cfg.DatabaseURL); err != nil {
		fmt.Fprintf(os.Stderr, "PostgreSQL に接続できません: %v\n", err)
		os.Exit(1)
	}
	if err := dial(cfg.RedisURL); err != nil {
		fmt.Fprintf(os.Stderr, "Redis に接続できません: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("環境を起動しました env=%s port=%d\n", cfg.Environment, cfg.HTTPPort)
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop
}

func dial(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	host := parsed.Host
	if host == "" {
		return fmt.Errorf("host is empty")
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
