package job

import (
	"context"
	"time"

	"escalator/usecase"
)

type ReleaseDisconnectedAgent struct {
	release *usecase.ReleaseDisconnectedAgents
	every   time.Duration
}

func NewReleaseDisconnectedAgent(release *usecase.ReleaseDisconnectedAgents, every time.Duration) *ReleaseDisconnectedAgent {
	return &ReleaseDisconnectedAgent{release: release, every: every}
}

//一定間隔で、接続が切れたままの担当者を離席にする
func (j *ReleaseDisconnectedAgent) Run(ctx context.Context) {
	ticker := time.NewTicker(j.every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if ctx.Err() != nil {
				return
			}
			_ = j.release.Execute(ctx)
		}
	}
}
