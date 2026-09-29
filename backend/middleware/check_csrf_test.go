package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAllowAcceptsSameOriginAndMatchingToken(t *testing.T) {
	req := request(t, "same-origin", "escalator_csrf_token", "token-value", "token-value")
	if err := Allow(req, "escalator_csrf_token"); err != nil {
		t.Fatal(err)
	}
}

func TestAllowAcceptsSameSite(t *testing.T) {
	req := request(t, "same-site", "escalator_csrf_token", "token-value", "token-value")
	if err := Allow(req, "escalator_csrf_token"); err != nil {
		t.Fatal(err)
	}
}

func TestAllowRejectsCrossSiteEvenWhenTokensMatch(t *testing.T) {
	req := request(t, "cross-site", "escalator_csrf_token", "token-value", "token-value")
	if err := Allow(req, "escalator_csrf_token"); err == nil {
		t.Fatal("cross-site must be rejected")
	}
}

func TestAllowRejectsMissingFetchSite(t *testing.T) {
	req := request(t, "", "escalator_csrf_token", "token-value", "token-value")
	if err := Allow(req, "escalator_csrf_token"); err == nil {
		t.Fatal("missing Sec-Fetch-Site must be rejected")
	}
}

func TestAllowRejectsMismatchedDoubleSubmit(t *testing.T) {
	req := request(t, "same-origin", "escalator_csrf_token", "cookie-token", "header-token")
	if err := Allow(req, "escalator_csrf_token"); err == nil {
		t.Fatal("mismatched tokens must be rejected")
	}
}

func request(t *testing.T, site, cookieName, cookieValue, headerValue string) *http.Request {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	if site != "" {
		req.Header.Set("Sec-Fetch-Site", site)
	}
	if cookieValue != "" {
		req.AddCookie(&http.Cookie{Name: cookieName, Value: cookieValue})
	}
	if headerValue != "" {
		req.Header.Set(CSRFHeaderName, headerValue)
	}
	return req
}
