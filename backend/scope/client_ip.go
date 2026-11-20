package scope

import "context"

type clientIPKey struct{}

//接続元を、この操作の続きから読めるように残す
func WithClientIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, clientIPKey{}, ip)
}

//この操作の接続元を返す
func ClientIP(ctx context.Context) string {
	ip, _ := ctx.Value(clientIPKey{}).(string)
	return ip
}
