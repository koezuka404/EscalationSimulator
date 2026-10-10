package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	"escalator/usecase/crypto"
)

const (
	tooManyMessage = "短い時間に同じ操作が続いたため、今は受け付けられません。時間を空けてから、もう一度試してください"
	checkMessage   = "回数を確認できませんでした。しばらくしてから、もう一度試してください"
)

type Taker interface {
	Take(ctx context.Context, key string, capacity int, perSecond float64) (bool, error)
}

type Limiter struct {
	taker       Taker
	secret      []byte
	issuer      string
	audience    string
	refreshName string
	userByHash  func(context.Context, string) (string, error)
}

type messageBody struct {
	Message string `json:"message"`
}

//回数を数える準備をする
func NewLimiter(taker Taker, secret []byte, issuer, audience, refreshCookie string, userByHash func(context.Context, string) (string, error)) *Limiter {
	return &Limiter{
		taker:       taker,
		secret:      secret,
		issuer:      issuer,
		audience:    audience,
		refreshName: refreshCookie,
		userByHash:  userByHash,
	}
}

//会員登録を、接続元ごとに数える
func (l *Limiter) Register(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		return l.decide(c, next, "register:"+clientIP(c.Request()), 3, 3.0/3600)
	}
}

//ログインを、接続元とアカウントごとに数える
func (l *Limiter) Login(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		return l.decide(c, next, "login:"+clientIP(c.Request())+":"+loginEmail(c), 5, 1.0/60)
	}
}

//ログイン用トークンの再発行を、利用者ごとに数える
func (l *Limiter) Refresh(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		raw := cookieValue(c, l.refreshName)
		if raw == "" {
			return next(c)
		}
		userID, err := l.userByHash(c.Request().Context(), crypto.Hash(raw))
		if err != nil {
			return c.JSON(http.StatusInternalServerError, messageBody{Message: checkMessage})
		}
		if userID == "" {
			return next(c)
		}
		return l.decide(c, next, "refresh:"+userID, 10, 5.0/60)
	}
}

//起票を、利用者ごとに数える
func (l *Limiter) CreateTicket(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		return l.forUser(c, next, "ticket", 10, 5.0/60)
	}
}

//次のチケットの引き取りを、利用者ごとに数える
func (l *Limiter) Claim(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		return l.forUser(c, next, "claim", 10, 5.0/60)
	}
}

//緊急度の変更を、利用者ごとに数える
func (l *Limiter) ChangeSeverity(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		return l.forUser(c, next, "severity", 10, 5.0/60)
	}
}

//デモの開始を、利用者ごとに数える
func (l *Limiter) StartDemo(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		return l.forUser(c, next, "demo", 3, 3.0/600)
	}
}

func (l *Limiter) forUser(c echo.Context, next echo.HandlerFunc, name string, capacity int, perSecond float64) error {
	header := c.Request().Header.Get("Authorization")
	raw := bearerToken(header)
	if raw == "" {
		return next(c)
	}
	userID, _, err := crypto.ParseAccess(l.secret, l.issuer, l.audience, raw)
	if err != nil || userID == "" {
		return next(c)
	}
	return l.decide(c, next, name+":"+userID, capacity, perSecond)
}

func (l *Limiter) decide(c echo.Context, next echo.HandlerFunc, key string, capacity int, perSecond float64) error {
	allowed, err := l.taker.Take(c.Request().Context(), key, capacity, perSecond)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, messageBody{Message: checkMessage})
	}
	if !allowed {
		return c.JSON(http.StatusTooManyRequests, messageBody{Message: tooManyMessage})
	}
	return next(c)
}

func loginEmail(c echo.Context) string {
	body, err := io.ReadAll(c.Request().Body)
	c.Request().Body = io.NopCloser(bytes.NewReader(body))
	if err != nil {
		return ""
	}
	var payload struct {
		Email string `json:"email"`
	}
	if json.Unmarshal(body, &payload) != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(payload.Email))
}

func cookieValue(c echo.Context, name string) string {
	cookie, err := c.Cookie(name)
	if err != nil {
		return ""
	}
	return cookie.Value
}

func bearerToken(authorization string) string {
	const prefix = "Bearer "
	value := strings.TrimSpace(authorization)
	if len(value) < len(prefix) || !strings.EqualFold(value[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(value[len(prefix):])
}

//プロキシが足した接続元を返す
func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		last := strings.TrimSpace(parts[len(parts)-1])
		if ip := net.ParseIP(last); ip != nil {
			return ip.String()
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
