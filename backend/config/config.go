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
		errs = append(errs, fmt.Errorf("HTTP_PORT must be between 1 and 65535"))
	}
	if c.DatabaseURL == "" {
		errs = append(errs, fmt.Errorf("DATABASE_URL is required"))
	} else if _, err := url.Parse(c.DatabaseURL); err != nil || !strings.Contains(c.DatabaseURL, "://") {
		errs = append(errs, fmt.Errorf("DATABASE_URL is invalid"))
	}
	if c.RedisURL == "" {
		errs = append(errs, fmt.Errorf("REDIS_URL is required"))
	} else if _, err := url.Parse(c.RedisURL); err != nil || !strings.Contains(c.RedisURL, "://") {
		errs = append(errs, fmt.Errorf("REDIS_URL is invalid"))
	}
	if len([]byte(c.JWTSecret)) < 32 {
		errs = append(errs, fmt.Errorf("JWT_SECRET must be at least 32 bytes"))
	}
	if strings.TrimSpace(c.JWTIssuer) == "" {
		errs = append(errs, fmt.Errorf("JWT_ISSUER is required"))
	}
	if strings.TrimSpace(c.JWTAudience) == "" {
		errs = append(errs, fmt.Errorf("JWT_AUDIENCE is required"))
	}
	if c.AccessTokenTTL <= 0 || c.AccessTokenTTL > 24*time.Hour {
		errs = append(errs, fmt.Errorf("ACCESS_TOKEN_TTL_MINUTES must be greater than 0 and no more than 1440"))
	}
	if c.RefreshTokenTTL <= c.AccessTokenTTL {
		errs = append(errs, fmt.Errorf("refresh token lifetime must be greater than the access token lifetime"))
	}
	if c.BcryptCost < 10 || c.BcryptCost > 16 {
		errs = append(errs, fmt.Errorf("BCRYPT_COST must be between 10 and 16"))
	}
	if c.Environment == EnvironmentProduction && !c.CookieSecure {
		errs = append(errs, fmt.Errorf("COOKIE_SECURE must be true in production"))
	}
	if strings.TrimSpace(c.RefreshTokenCookieName) == "" {
		errs = append(errs, fmt.Errorf("REFRESH_TOKEN_COOKIE_NAME is required"))
	}
	if strings.TrimSpace(c.CSRFTokenCookieName) == "" {
		errs = append(errs, fmt.Errorf("CSRF_TOKEN_COOKIE_NAME is required"))
	}
	if c.LoginMaxFailures < 1 {
		errs = append(errs, fmt.Errorf("LOGIN_MAX_FAILURES must be greater than 0"))
	}
	if c.LoginLock <= 0 {
		errs = append(errs, fmt.Errorf("LOGIN_LOCK_SECONDS must be greater than 0"))
	}
	if c.PriorityRecalc <= 0 {
		errs = append(errs, fmt.Errorf("PRIORITY_RECALC_SECONDS must be greater than 0"))
	}
	if c.AgentDisconnectGrace <= 0 {
		errs = append(errs, fmt.Errorf("AGENT_DISCONNECT_GRACE_SECONDS must be greater than 0"))
	}
	if c.CORSAllowedOrigin == "" {
		errs = append(errs, fmt.Errorf("CORS_ALLOWED_ORIGIN is required"))
	} else if err := validateOrigin(c.Environment, c.CORSAllowedOrigin); err != nil {
		errs = append(errs, err)
	}
	if strings.TrimSpace(c.RedisPubSubChannelPrefix) == "" {
		errs = append(errs, fmt.Errorf("REDIS_PUBSUB_CHANNEL_PREFIX is required"))
	}

	return errors.Join(errs...)
}

func validateOrigin(environment Environment, origin string) error {
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("CORS_ALLOWED_ORIGIN must include a scheme and host")
	}
	if environment == EnvironmentProduction && parsed.Scheme != "https" {
		return fmt.Errorf("CORS_ALLOWED_ORIGIN must be HTTPS in production")
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
		return "", fmt.Errorf("APP_ENV must be development or production")
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
