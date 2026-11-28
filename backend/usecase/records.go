package usecase

import (
	"context"
	"log"
	"time"

	"escalator/entity"
	"escalator/port"
	"escalator/scope"
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

type RequestLog struct {
	repo port.RequestLogRepository
}

func NewRequestLog(repo port.RequestLogRepository) *RequestLog {
	return &RequestLog{repo: repo}
}

//受け付けた操作を1件残す失敗しても、応答は取り消さない
func (r *RequestLog) Save(ctx context.Context, requestID, userID, method, path string, status, latencyMS int, errorCode string) {
	if r == nil || r.repo == nil {
		return
	}
	err := r.repo.Save(ctx, entity.RequestLogEntry{
		RequestID: requestID,
		UserID:    userID,
		Method:    method,
		Path:      path,
		Status:    status,
		LatencyMS: latencyMS,
		ErrorCode: errorCode,
		CreatedAt: time.Now(),
	})
	if err != nil {
		log.Printf("受け付けた操作の記録を残せませんでした: %v", err)
	}
}
