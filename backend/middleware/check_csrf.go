package middleware

import (
	"crypto/subtle"
	"errors"
	"github.com/labstack/echo/v4"
	"net/http"
)

const CSRFHeaderName = "X-CSRF-Token"

var ErrCSRFRejected = errors.New("この操作を確認できませんでした。ページを開き直してから、もう一度試してください")

//再発行とログアウトで、操作の出どころと確認用の印を確かめる
func Allow(r *http.Request, cookieName string) error {
	switch r.Header.Get("Sec-Fetch-Site") {
	case "same-origin", "same-site":
	default:
		return ErrCSRFRejected
	}

	cookie, err := r.Cookie(cookieName)
	if err != nil || cookie.Value == "" {
		return ErrCSRFRejected
	}
	header := r.Header.Get(CSRFHeaderName)
	if header == "" {
		return ErrCSRFRejected
	}

	cookieBytes := []byte(cookie.Value)
	headerBytes := []byte(header)
	if len(cookieBytes) != len(headerBytes) || subtle.ConstantTimeCompare(cookieBytes, headerBytes) != 1 {
		return ErrCSRFRejected
	}
	return nil
}

//再発行とログアウトの前に、送信元を確かめる
func CheckCSRF(cookieName string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if err := Allow(c.Request(), cookieName); err != nil {
				return c.JSON(http.StatusForbidden, map[string]string{"message": err.Error()})
			}
			return next(c)
		}
	}
}
