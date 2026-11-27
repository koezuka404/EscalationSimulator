package job

import "context"

type demoRecovery interface {
	Recover(ctx context.Context) error
}

//前回の起動で残ったデモを止める
func RecoverDemo(ctx context.Context, demo demoRecovery) error {
	return demo.Recover(ctx)
}
