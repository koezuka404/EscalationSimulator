package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"escalator/domain"
)

type sessionRow struct {
	ID        string `gorm:"primaryKey"`
	UserID    string `gorm:"index"`
	TokenHash string `gorm:"uniqueIndex"`
	FamilyID  string `gorm:"index"`
	ExpiresAt time.Time
	Revoked   bool
	CreatedAt time.Time
}

func (sessionRow) TableName() string { return "sessions" }

type SessionRepository struct {
	db *gorm.DB
}

func NewSessionRepository(db *gorm.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

func MigrateSessions(db *gorm.DB) error {
	return db.AutoMigrate(&sessionRow{})
}

func (r *SessionRepository) Save(ctx context.Context, session domain.Session) error {
	if session.ID == "" {
		session.ID = uuid.NewString()
	}
	row := sessionRow{
		ID:        session.ID,
		UserID:    session.UserID,
		TokenHash: session.TokenHash,
		FamilyID:  session.FamilyID,
		ExpiresAt: session.ExpiresAt,
	}
	return r.db.WithContext(ctx).Create(&row).Error
}
