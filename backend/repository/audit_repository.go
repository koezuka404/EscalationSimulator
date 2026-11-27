package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"escalator/entity"
	"escalator/port"
)

type auditRow struct {
	ID         string `gorm:"primaryKey"`
	UserID     string `gorm:"index"`
	Action     string `gorm:"index"`
	TargetType string
	TargetID   string
	IPAddress  string
	Details    string `gorm:"type:text"`
	CreatedAt  time.Time
}

func (auditRow) TableName() string { return "audit_logs" }

type auditRepository struct {
	db *gorm.DB
}

func NewAuditRepository(db *gorm.DB) port.AuditRepository {
	return &auditRepository{db: db}
}

func MigrateAuditLogs(db *gorm.DB) error {
	return db.AutoMigrate(&auditRow{})
}

//操作の記録を1件残す
func (r *auditRepository) Save(ctx context.Context, entry entity.AuditEntry) error {
	if entry.ID == "" {
		entry.ID = uuid.NewString()
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}
	row := auditRow{
		ID:         entry.ID,
		UserID:     entry.UserID,
		Action:     entry.Action,
		TargetType: entry.TargetType,
		TargetID:   entry.TargetID,
		IPAddress:  entry.IPAddress,
		Details:    entry.Details,
		CreatedAt:  entry.CreatedAt,
	}
	return r.db.WithContext(ctx).Create(&row).Error
}
