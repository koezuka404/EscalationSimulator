package usecase

import (
	"context"
	"log"
	"time"
)

const PriorityLockKey = "lock:recalc_priority"

type PriorityLock interface {
	TryLock(ctx context.Context, key string, ttl time.Duration) (token string, ok bool, err error)
	Unlock(ctx context.Context, key, token string) error
}

//点数のやり直しが重ならないように、先に取れたときだけ実行する
func WithPriorityLock(ctx context.Context, locks PriorityLock, every time.Duration, missed string, run func()) {
	if ctx.Err() != nil {
		return
	}
	token, ok, err := locks.TryLock(ctx, PriorityLockKey, every)
	if err != nil {
		log.Printf("点数のやり直しを始められませんでした: %v", err)
		return
	}
	if !ok {
		if missed != "" {
			log.Printf("%s", missed)
		}
		return
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := locks.Unlock(unlockCtx, PriorityLockKey, token); err != nil {
			log.Printf("点数のやり直しを終えても、次をすぐ始められる状態に戻せませんでした。次のやり直しは、しばらくしてから始まります: %v", err)
		}
	}()
	run()
}
