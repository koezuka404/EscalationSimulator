package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"escalator/entity"
	"escalator/port"
)

type connectionLogRow struct {
	ID           string `gorm:"primaryKey"`
	UserID       string `gorm:"index"`
	ConnectionID string `gorm:"index"`
	EventType    string
	Reason       string
	IPAddress    string
	CreatedAt    time.Time
}

func (connectionLogRow) TableName() string { return "websocket_logs" }

type connectionLogRepository struct {
	db *gorm.DB
}

func NewConnectionLogRepository(db *gorm.DB) port.ConnectionLogRepository {
	return &connectionLogRepository{db: db}
}

func MigrateConnectionLogs(db *gorm.DB) error {
	return db.AutoMigrate(&connectionLogRow{})
}

//常時接続の記録を1件残す
func (r *connectionLogRepository) Save(ctx context.Context, entry entity.ConnectionLogEntry) error {
	if entry.ID == "" {
		entry.ID = uuid.NewString()
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}
	row := connectionLogRow{
		ID:           entry.ID,
		UserID:       entry.UserID,
		ConnectionID: entry.ConnectionID,
		EventType:    entry.EventType,
		Reason:       entry.Reason,
		IPAddress:    entry.IPAddress,
		CreatedAt:    entry.CreatedAt,
	}
	return r.db.WithContext(ctx).Create(&row).Error
}
