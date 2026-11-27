package usecase

import (
	"context"
	"log"
	"time"

	"escalator/entity"
	"escalator/port"
)

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
