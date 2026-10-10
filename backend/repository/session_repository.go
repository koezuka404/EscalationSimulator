package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"escalator/entity"
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

type sessionRepository struct {
	db *gorm.DB
}

func NewSessionRepository(db *gorm.DB) SessionRepository {
	return &sessionRepository{db: db}
}

func MigrateSessions(db *gorm.DB) error {
	return db.AutoMigrate(&sessionRow{})
}

func (r *sessionRepository) Save(ctx context.Context, session entity.Session) error {
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

func (r *sessionRepository) RevokeByHash(ctx context.Context, tokenHash string) error {
	return r.db.WithContext(ctx).Model(&sessionRow{}).Where("token_hash = ?", tokenHash).Update("revoked", true).Error
}

//再ログイン用の印から利用者IDを返す
func (r *sessionRepository) FindUserIDByHash(ctx context.Context, tokenHash string) (string, error) {
	var row sessionRow
	err := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", entity.ErrInvalidRefresh
	}
	if err != nil {
		return "", err
	}
	return row.UserID, nil
}

//再ログイン用の印を新しい印に替える古い印の再使用は系統ごと無効にする
func (r *sessionRepository) Rotate(ctx context.Context, oldHash string, next entity.Session, now time.Time) (string, bool, error) {
	var userID string
	var reused bool
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row sessionRow
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("token_hash = ?", oldHash).First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.ErrInvalidRefresh
		}
		if err != nil {
			return err
		}
		userID = row.UserID
		if row.Revoked {
			reused = true
			return tx.Model(&sessionRow{}).Where("family_id = ?", row.FamilyID).Update("revoked", true).Error
		}
		if !row.ExpiresAt.After(now) {
			return entity.ErrInvalidRefresh
		}
		if err := tx.Model(&sessionRow{}).Where("id = ?", row.ID).Update("revoked", true).Error; err != nil {
			return err
		}
		if next.ID == "" {
			next.ID = uuid.NewString()
		}
		return tx.Create(&sessionRow{
			ID:        next.ID,
			UserID:    row.UserID,
			TokenHash: next.TokenHash,
			FamilyID:  row.FamilyID,
			ExpiresAt: next.ExpiresAt,
		}).Error
	})
	return userID, reused, err
}
