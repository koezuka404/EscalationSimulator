package job

import (
	"context"
	"log"
	"time"

	"escalator/usecase"
)

type DashboardSummary interface {
	Publish(ctx context.Context)
}

type RecalcPriority struct {
	recalc    *usecase.RecalcOpenScores
	dashboard DashboardSummary
	locks     usecase.PriorityLock
	every     time.Duration
}

func NewRecalcPriority(recalc *usecase.RecalcOpenScores, dashboard DashboardSummary, locks usecase.PriorityLock, every time.Duration) *RecalcPriority {
	return &RecalcPriority{recalc: recalc, dashboard: dashboard, locks: locks, every: every}
}

// 起動時に待ち順を作り直し、その後は一定間隔で点数をやり直す
func (j *RecalcPriority) Run(ctx context.Context) {
	j.rebuild(ctx)
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

// 他で実行中なら飛ばし、対応待ちだけを待ち順へ載せ直す
func (j *RecalcPriority) rebuild(ctx context.Context) {
	j.withLock(ctx, "待ち順の作り直しは、別のやり直しの途中だったため飛ばしました", func() {
		if err := j.recalc.Rebuild(ctx); err != nil {
			log.Printf("待ち順を作り直せませんでした: %v", err)
		}
		if j.dashboard != nil {
			j.dashboard.Publish(ctx)
		}
	})
}

// 他で実行中なら飛ばし、対応待ちの点数をやり直し、現場の数字をまとめる
func (j *RecalcPriority) runOnce(ctx context.Context) {
	j.withLock(ctx, "", func() {
		if err := j.recalc.Execute(ctx); err != nil {
			log.Printf("対応待ちの点数をやり直せませんでした: %v", err)
		}
		if j.dashboard != nil {
			j.dashboard.Publish(ctx)
		}
	})
}

func (j *RecalcPriority) withLock(ctx context.Context, missed string, run func()) {
	usecase.WithPriorityLock(ctx, j.locks, j.every, missed, run)
}
