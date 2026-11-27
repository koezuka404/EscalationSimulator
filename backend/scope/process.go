package scope

import "context"

type processKey struct{}

//プロセスの生存期間を、この操作の続きから読めるように残す
func WithProcess(ctx context.Context, life context.Context) context.Context {
	return context.WithValue(ctx, processKey{}, life)
}

//この操作を止めるときに一緒に止める生存期間を返す
func Process(ctx context.Context) context.Context {
	life, _ := ctx.Value(processKey{}).(context.Context)
	if life == nil {
		return context.Background()
	}
	return life
}
