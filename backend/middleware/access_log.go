package middleware

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"escalator/usecase/crypto"
)

type RequestRecorder interface {
	Save(ctx context.Context, requestID, userID, method, path string, status, latencyMS int, errorCode string)
}

// 受け付けた操作の番号、経路、状態、かかった時間を残す
func AccessLog(secret []byte, issuer, audience string, records RequestRecorder) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			started := time.Now()
			err := next(c)
			status := c.Response().Status
			if status == 0 {
				status = 200
			}
			latency := time.Since(started).Truncate(time.Millisecond)
			log.Printf("受け付けました 番号=%s %s %s 状態=%d 経過=%s",
				RequestIDFrom(c),
				c.Request().Method,
				c.Request().URL.Path,
				status,
				latency,
			)
			if records != nil {
				saveCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				records.Save(saveCtx, RequestIDFrom(c), accessUserID(c, secret, issuer, audience), c.Request().Method, c.Request().URL.Path, status, int(latency.Milliseconds()), errorCode(status))
				cancel()
			}
			return err
		}
	}
}

func accessUserID(c echo.Context, secret []byte, issuer, audience string) string {
	raw := bearerToken(c.Request().Header.Get("Authorization"))
	if raw == "" {
		return ""
	}
	userID, _, err := crypto.ParseAccess(secret, issuer, audience, raw)
	if err != nil {
		return ""
	}
	return userID
}

func errorCode(status int) string {
	if status < 400 {
		return ""
	}
	return strconv.Itoa(status)
}
