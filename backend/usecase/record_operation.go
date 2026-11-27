package usecase

import (
	"context"
	"log"
	"time"

	"escalator/entity"
	"escalator/port"
	"escalator/scope"
)

type AuditLog struct {
	repo port.AuditRepository
}

func NewAuditLog(repo port.AuditRepository) *AuditLog {
	return &AuditLog{repo: repo}
}

//操作の記録を残す失敗しても、元の操作は取り消さない
func (a *AuditLog) Record(ctx context.Context, userID, action, targetType, targetID, details string) {
	if a == nil || a.repo == nil {
		return
	}
	err := a.repo.Save(ctx, entity.AuditEntry{
		UserID:     userID,
		Action:     action,
		TargetType: targetType,
		TargetID:   targetID,
		IPAddress:  scope.ClientIP(ctx),
		Details:    details,
		CreatedAt:  time.Now(),
	})
	if err != nil {
		log.Printf("操作の記録を残せませんでした: %v", err)
	}
}
