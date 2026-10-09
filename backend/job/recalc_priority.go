package job

import (
	"context"
	"log"
	"time"

	"escalator/usecase"
)

const recalcLockKey = "lock:recalc_priority"

type RecalcLock interface {
	TryLock(ctx context.Context, key string, ttl time.Duration) (token string, ok bool, err error)
	Unlock(ctx context.Context, key, token string) error
}

type DashboardSummary interface {
	Publish(ctx context.Context)
}

type RecalcPriority struct {
	recalc    *usecase.RecalcOpenScores
	dashboard DashboardSummary
	locks     RecalcLock
	every     time.Duration
}

func NewRecalcPriority(recalc *usecase.RecalcOpenScores, dashboard DashboardSummary, locks RecalcLock, every time.Duration) *RecalcPriority {
	return &RecalcPriority{recalc: recalc, dashboard: dashboard, locks: locks, every: every}
}

//起動時と一定間隔で、対応待ちの点数をやり直す
func (j *RecalcPriority) Run(ctx context.Context) {
	j.runOnce(ctx)
	ticker := time.NewTicker(j.every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			j.runOnce(ctx)
		}
	}
}

//他で実行中なら飛ばし、対応待ちの点数をやり直し、現場の数字をまとめる
func (j *RecalcPriority) runOnce(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	token, ok, err := j.locks.TryLock(ctx, recalcLockKey, j.every)
	if err != nil {
		log.Printf("点数のやり直しを始められませんでした: %v", err)
		return
	}
	if !ok {
		return
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := j.locks.Unlock(unlockCtx, recalcLockKey, token); err != nil {
			log.Printf("点数のやり直しを終えても、次をすぐ始められる状態に戻せませんでした。次のやり直しは、しばらくしてから始まります: %v", err)
		}
	}()
	if err := j.recalc.Execute(ctx); err != nil {
		log.Printf("対応待ちの点数をやり直せませんでした: %v", err)
	}
	if j.dashboard != nil {
		j.dashboard.Publish(ctx)
	}
}
