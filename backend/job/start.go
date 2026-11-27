package job

import (
	"context"
	"fmt"
	"sync"
)

//前回のデモを止め、定期処理を始める
func Start(ctx context.Context, demo demoRecovery, recalc *RecalcPriority, release *ReleaseDisconnectedAgent) (func(), error) {
	if err := RecoverDemo(ctx, demo); err != nil {
		return nil, fmt.Errorf("前回のデモを止めた状態にできませんでした: %w", err)
	}
	var group sync.WaitGroup
	group.Add(2)
	go func() {
		defer group.Done()
		recalc.Run(ctx)
	}()
	go func() {
		defer group.Done()
		release.Run(ctx)
	}()
	return group.Wait, nil
}
