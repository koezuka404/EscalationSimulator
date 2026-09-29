package config

import "testing"

func validEnv() map[string]string {
	return map[string]string{
		"APP_ENV":             "development",
		"JWT_SECRET":          "change-me-use-at-least-32-bytes!!",
		"CORS_ALLOWED_ORIGIN": "http://localhost:5175",
		"DATABASE_URL":        "postgres://escalator:escalator@localhost:5438/escalator?sslmode=disable",
		"REDIS_URL":           "redis://localhost:6384/0",
	}
}

func load(t *testing.T, env map[string]string) (*Config, error) {
	t.Helper()
	return LoadFromEnv(func(key string) string { return env[key] })
}

func TestDevelopmentDefaults(t *testing.T) {
	cfg, err := load(t, validEnv())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.JWTIssuer != "escalator-api" || cfg.JWTAudience != "escalator-client" {
		t.Fatalf("issuer/audience: %s %s", cfg.JWTIssuer, cfg.JWTAudience)
	}
	if cfg.AccessTokenTTL.Minutes() != 15 {
		t.Fatalf("access ttl: %s", cfg.AccessTokenTTL)
	}
	if cfg.RefreshTokenTTL.Minutes() != 30 {
		t.Fatalf("refresh ttl: %s", cfg.RefreshTokenTTL)
	}
	if cfg.BcryptCost != 12 || cfg.CookieSecure {
		t.Fatalf("bcrypt/cookie: %d %v", cfg.BcryptCost, cfg.CookieSecure)
	}
	if cfg.LoginLock.Seconds() != 900 || cfg.PriorityRecalc.Seconds() != 30 || cfg.AgentDisconnectGrace.Seconds() != 300 {
		t.Fatalf("timers: lock=%s recalc=%s grace=%s", cfg.LoginLock, cfg.PriorityRecalc, cfg.AgentDisconnectGrace)
	}
	if cfg.RedisPubSubChannelPrefix != "escalator:notify:" {
		t.Fatal(cfg.RedisPubSubChannelPrefix)
	}
	if cfg.RefreshTokenCookieName != "escalator_refresh_token" || cfg.CSRFTokenCookieName != "escalator_csrf_token" {
		t.Fatal("cookie names")
	}
}

func TestProductionDefaults(t *testing.T) {
	env := validEnv()
	env["APP_ENV"] = "production"
	env["CORS_ALLOWED_ORIGIN"] = "https://support.example.com"
	cfg, err := load(t, env)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.RefreshTokenTTL.Hours() != 14*24 {
		t.Fatalf("refresh ttl: %s", cfg.RefreshTokenTTL)
	}
	if !cfg.CookieSecure {
		t.Fatal("production cookie must be secure")
	}
	if cfg.LoginLock.Seconds() != 3600 {
		t.Fatalf("lock: %s", cfg.LoginLock)
	}
}

func TestRejectsMissingRequiredValues(t *testing.T) {
	env := validEnv()
	delete(env, "JWT_SECRET")
	if _, err := load(t, env); err == nil {
		t.Fatal("JWT_SECRET is required")
	}

	env = validEnv()
	env["JWT_SECRET"] = "short"
	if _, err := load(t, env); err == nil {
		t.Fatal("JWT_SECRET must be at least 32 bytes")
	}

	env = validEnv()
	delete(env, "CORS_ALLOWED_ORIGIN")
	if _, err := load(t, env); err == nil {
		t.Fatal("CORS_ALLOWED_ORIGIN is required")
	}
}

func TestProductionOriginMustBeHTTPS(t *testing.T) {
	env := validEnv()
	env["APP_ENV"] = "production"
	if _, err := load(t, env); err == nil {
		t.Fatal("production origin must be https")
	}
}
