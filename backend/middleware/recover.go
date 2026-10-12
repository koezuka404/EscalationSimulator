package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/labstack/echo/v4"
)

//操作の途中で止まったとき、画面には短い説明だけを返す
func Recover() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			defer func() {
				recovered := recover()
				if recovered == nil {
					return
				}
				log.Printf("操作の途中で止まったため、結果を返せませんでした 番号=%s: %v\n%s", RequestIDFrom(c), recovered, debug.Stack())
				if c.Response().Committed {
					return
				}
				err = c.JSON(http.StatusInternalServerError, map[string]string{
					"message": "操作を完了できませんでした。しばらくしてから、もう一度試してください",
				})
			}()
			return next(c)
		}
	}
}
