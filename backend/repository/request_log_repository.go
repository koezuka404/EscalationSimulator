package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"escalator/entity"
	"escalator/port"
)

type requestLogRow struct {
	ID        string `gorm:"primaryKey"`
	RequestID string `gorm:"index"`
	UserID    string `gorm:"index"`
	Method    string
	Path      string
	Status    int
	LatencyMS int
	ErrorCode string
	CreatedAt time.Time
}

func (requestLogRow) TableName() string { return "api_request_logs" }

type requestLogRepository struct {
	db *gorm.DB
}

func NewRequestLogRepository(db *gorm.DB) port.RequestLogRepository {
	return &requestLogRepository{db: db}
}

func MigrateRequestLogs(db *gorm.DB) error {
	return db.AutoMigrate(&requestLogRow{})
}

//受け付けた操作の記録を1件残す
func (r *requestLogRepository) Save(ctx context.Context, entry entity.RequestLogEntry) error {
	if entry.ID == "" {
		entry.ID = uuid.NewString()
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}
	row := requestLogRow{
		ID:        entry.ID,
		RequestID: entry.RequestID,
		UserID:    entry.UserID,
		Method:    entry.Method,
		Path:      entry.Path,
		Status:    entry.Status,
		LatencyMS: entry.LatencyMS,
		ErrorCode: entry.ErrorCode,
		CreatedAt: entry.CreatedAt,
	}
	return r.db.WithContext(ctx).Create(&row).Error
}
