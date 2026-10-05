package middleware

import (
	"crypto/subtle"
	"errors"
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
