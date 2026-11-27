package httpx

import "strings"

//AuthorizationからBearerのトークンを取る
func BearerToken(authorization string) string {
	const prefix = "Bearer "
	value := strings.TrimSpace(authorization)
	if len(value) < len(prefix) || !strings.EqualFold(value[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(value[len(prefix):])
}
