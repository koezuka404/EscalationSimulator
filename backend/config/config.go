package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Environment string

const (
	EnvironmentDevelopment Environment = "development"
	EnvironmentProduction  Environment = "production"
)

type Config struct {
	Environment Environment
	HTTPPort    int

	DatabaseURL string
	RedisURL    string

	JWTSecret       string
	JWTIssuer       string
	JWTAudience     string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	BcryptCost             int
	CookieSecure           bool
	RefreshTokenCookieName string
	CSRFTokenCookieName    string

	LoginMaxFailures         int
	LoginLock                time.Duration
	PriorityRecalc           time.Duration
	AgentDisconnectGrace     time.Duration
	CORSAllowedOrigin        string
	RedisPubSubChannelPrefix string
}

func Load() (*Config, error) {
	return LoadFromEnv(os.Getenv)
}

func LoadFromEnv(getenv func(string) string) (*Config, error) {
	environment, err := parseEnvironment(valueOrDefault(getenv("APP_ENV"), "development"))
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Environment: environment,
		HTTPPort:    intValue(getenv("HTTP_PORT"), 8080),

		DatabaseURL: strings.TrimSpace(getenv("DATABASE_URL")),
		RedisURL:    strings.TrimSpace(getenv("REDIS_URL")),

		JWTSecret:       getenv("JWT_SECRET"),
		JWTIssuer:       valueOrDefault(getenv("JWT_ISSUER"), "escalator-api"),
		JWTAudience:     valueOrDefault(getenv("JWT_AUDIENCE"), "escalator-client"),
		AccessTokenTTL:  time.Duration(intValue(getenv("ACCESS_TOKEN_TTL_MINUTES"), 15)) * time.Minute,
		RefreshTokenTTL: resolveRefreshTokenTTL(environment, getenv),

		BcryptCost:             intValue(getenv("BCRYPT_COST"), 12),
		CookieSecure:           boolValue(getenv("COOKIE_SECURE"), environment == EnvironmentProduction),
		RefreshTokenCookieName: valueOrDefault(getenv("REFRESH_TOKEN_COOKIE_NAME"), "escalator_refresh_token"),
		CSRFTokenCookieName:    valueOrDefault(getenv("CSRF_TOKEN_COOKIE_NAME"), "escalator_csrf_token"),

		LoginMaxFailures:         intValue(getenv("LOGIN_MAX_FAILURES"), 5),
		LoginLock:                time.Duration(resolveLoginLockSeconds(environment, getenv)) * time.Second,
		PriorityRecalc:           time.Duration(intValue(getenv("PRIORITY_RECALC_SECONDS"), 30)) * time.Second,
		AgentDisconnectGrace:     time.Duration(intValue(getenv("AGENT_DISCONNECT_GRACE_SECONDS"), 300)) * time.Second,
		CORSAllowedOrigin:        strings.TrimSpace(getenv("CORS_ALLOWED_ORIGIN")),
		RedisPubSubChannelPrefix: valueOrDefault(getenv("REDIS_PUBSUB_CHANNEL_PREFIX"), "escalator:notify:"),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	var errs []error

	if c.HTTPPort < 1 || c.HTTPPort > 65535 {
		errs = append(errs, fmt.Errorf("HTTP_PORT は1から65535の番号にしてください。画面から呼び出す入口のポートです"))
	}
	if c.DatabaseURL == "" {
		errs = append(errs, fmt.Errorf("DATABASE_URL を設定してください。データベースの接続先です"))
	} else if _, err := url.Parse(c.DatabaseURL); err != nil || !strings.Contains(c.DatabaseURL, "://") {
		errs = append(errs, fmt.Errorf("DATABASE_URL の形が違います。データベースの接続先を確認してください"))
	}
	if c.RedisURL == "" {
		errs = append(errs, fmt.Errorf("REDIS_URL を設定してください。待ち順の保存先です"))
	} else if _, err := url.Parse(c.RedisURL); err != nil || !strings.Contains(c.RedisURL, "://") {
		errs = append(errs, fmt.Errorf("REDIS_URL の形が違います。待ち順の保存先を確認してください"))
	}
	if len([]byte(c.JWTSecret)) < 32 {
		errs = append(errs, fmt.Errorf("JWT_SECRET は32文字以上にしてください。ログイン用の秘密の文字列です"))
	}
	if strings.TrimSpace(c.JWTIssuer) == "" {
		errs = append(errs, fmt.Errorf("JWT_ISSUER を設定してください。ログイン情報の発行元の名前です"))
	}
	if strings.TrimSpace(c.JWTAudience) == "" {
		errs = append(errs, fmt.Errorf("JWT_AUDIENCE を設定してください。ログイン情報の宛先の名前です"))
	}
	if c.AccessTokenTTL <= 0 || c.AccessTokenTTL > 24*time.Hour {
		errs = append(errs, fmt.Errorf("ACCESS_TOKEN_TTL_MINUTES は1分以上、1440分（24時間）以内にしてください。ログインの有効時間です"))
	}
	if c.RefreshTokenTTL <= c.AccessTokenTTL {
		errs = append(errs, fmt.Errorf("再ログインできる時間は、ログインの有効時間より長くしてください"))
	}
	if c.BcryptCost < 10 || c.BcryptCost > 16 {
		errs = append(errs, fmt.Errorf("BCRYPT_COST は10から16の間にしてください。パスワードを保存するときの強さです"))
	}
	if c.Environment == EnvironmentProduction && !c.CookieSecure {
		errs = append(errs, fmt.Errorf("本番では COOKIE_SECURE を true にしてください。再ログイン用の印を保護します"))
	}
	if strings.TrimSpace(c.RefreshTokenCookieName) == "" {
		errs = append(errs, fmt.Errorf("REFRESH_TOKEN_COOKIE_NAME を設定してください。再ログイン用の印の名前です"))
	}
	if strings.TrimSpace(c.CSRFTokenCookieName) == "" {
		errs = append(errs, fmt.Errorf("CSRF_TOKEN_COOKIE_NAME を設定してください。操作確認用の印の名前です"))
	}
	if c.LoginMaxFailures < 1 {
		errs = append(errs, fmt.Errorf("LOGIN_MAX_FAILURES は1以上にしてください。ログイン失敗を何回まで許すかの回数です"))
	}
	if c.LoginLock <= 0 {
		errs = append(errs, fmt.Errorf("LOGIN_LOCK_SECONDS は1秒以上にしてください。ログインを止めておく時間です"))
	}
	if c.PriorityRecalc <= 0 {
		errs = append(errs, fmt.Errorf("PRIORITY_RECALC_SECONDS は1秒以上にしてください。点数をやり直す間隔です"))
	}
	if c.AgentDisconnectGrace <= 0 {
		errs = append(errs, fmt.Errorf("AGENT_DISCONNECT_GRACE_SECONDS は1秒以上にしてください。担当者の接続が切れてから外すまでの時間です"))
	}
	if c.CORSAllowedOrigin == "" {
		errs = append(errs, fmt.Errorf("CORS_ALLOWED_ORIGIN を設定してください。画面のアドレスです"))
	} else if err := validateOrigin(c.Environment, c.CORSAllowedOrigin); err != nil {
		errs = append(errs, err)
	}
	if strings.TrimSpace(c.RedisPubSubChannelPrefix) == "" {
		errs = append(errs, fmt.Errorf("REDIS_PUBSUB_CHANNEL_PREFIX を設定してください。画面へ知らせを渡す名前の先頭です"))
	}

	return errors.Join(errs...)
}

func validateOrigin(environment Environment, origin string) error {
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("CORS_ALLOWED_ORIGIN は http:// または https:// から始まるアドレスにしてください")
	}
	if environment == EnvironmentProduction && parsed.Scheme != "https" {
		return fmt.Errorf("本番の画面アドレスは https:// から始めてください")
	}
	return nil
}

func resolveRefreshTokenTTL(environment Environment, getenv func(string) string) time.Duration {
	if raw := strings.TrimSpace(getenv("REFRESH_TOKEN_TTL_MINUTES")); raw != "" {
		return time.Duration(intValue(raw, -1)) * time.Minute
	}
	if raw := strings.TrimSpace(getenv("REFRESH_TOKEN_TTL_DAYS")); raw != "" {
		return time.Duration(intValue(raw, -1)) * 24 * time.Hour
	}
	if environment == EnvironmentProduction {
		return 14 * 24 * time.Hour
	}
	return 30 * time.Minute
}

func resolveLoginLockSeconds(environment Environment, getenv func(string) string) int {
	if raw := strings.TrimSpace(getenv("LOGIN_LOCK_SECONDS")); raw != "" {
		return intValue(raw, -1)
	}
	if environment == EnvironmentProduction {
		return 3600
	}
	return 900
}

func parseEnvironment(raw string) (Environment, error) {
	switch Environment(strings.ToLower(strings.TrimSpace(raw))) {
	case EnvironmentDevelopment:
		return EnvironmentDevelopment, nil
	case EnvironmentProduction:
		return EnvironmentProduction, nil
	default:
		return "", fmt.Errorf("APP_ENV は development か production にしてください。開発中か本番かを表します")
	}
}

func valueOrDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func intValue(raw string, fallback int) int {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return -1
	}
	return value
}

func boolValue(raw string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "":
		return fallback
	case "1", "true", "yes":
		return true
	case "0", "false", "no":
		return false
	default:
		return fallback
	}
}
