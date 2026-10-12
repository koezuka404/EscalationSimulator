package middleware

import (
	"log"
	"time"

	"github.com/labstack/echo/v4"
)

//受け付けた操作の番号、経路、状態、かかった時間を残す
func AccessLog() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			started := time.Now()
			err := next(c)
			status := c.Response().Status
			if status == 0 {
				status = 200
			}
			log.Printf("受け付けました 番号=%s %s %s 状態=%d 経過=%s",
				RequestIDFrom(c),
				c.Request().Method,
				c.Request().URL.Path,
				status,
				time.Since(started).Truncate(time.Millisecond),
			)
			return err
		}
	}
}
