package usecase

import (
	"context"
	"log"
	"time"

	"escalator/entity"
	"escalator/port"
)

type ConnectionLog struct {
	repo port.ConnectionLogRepository
}

func NewConnectionLog(repo port.ConnectionLogRepository) *ConnectionLog {
	return &ConnectionLog{repo: repo}
}

//常時接続の記録を1件残す失敗しても、接続はそのままにする
func (c *ConnectionLog) Save(ctx context.Context, userID, connectionID, eventType, reason, ip string) {
	if c == nil || c.repo == nil {
		return
	}
	err := c.repo.Save(ctx, entity.ConnectionLogEntry{
		UserID:       userID,
		ConnectionID: connectionID,
		EventType:    eventType,
		Reason:       reason,
		IPAddress:    ip,
		CreatedAt:    time.Now(),
	})
	if err != nil {
		log.Printf("常時接続の記録を残せませんでした: %v", err)
	}
}
