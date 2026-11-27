package middleware

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"escalator/httpx"
	"escalator/port"
)

type RequestRecorder interface {
	Save(ctx context.Context, requestID, userID, method, path string, status, latencyMS int, errorCode string)
}

//受け付けた操作の番号、経路、状態、かかった時間を残す
func AccessLog(tokens port.TokenVerifier, records RequestRecorder) echo.MiddlewareFunc {
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
				records.Save(saveCtx, RequestIDFrom(c), accessUserID(c, tokens), c.Request().Method, c.Request().URL.Path, status, int(latency.Milliseconds()), errorCode(status))
				cancel()
			}
			return err
		}
	}
}

func accessUserID(c echo.Context, tokens port.TokenVerifier) string {
	raw := httpx.BearerToken(c.Request().Header.Get("Authorization"))
	if raw == "" || tokens == nil {
		return ""
	}
	userID, _, err := tokens.Parse(raw)
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
