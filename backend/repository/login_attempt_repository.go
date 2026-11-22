package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"escalator/entity"
	"escalator/port"
)

type loginAttemptRow struct {
	ID           string `gorm:"primaryKey"`
	Email        string `gorm:"uniqueIndex:idx_login_attempts_email_ip"`
	IPAddress    string `gorm:"uniqueIndex:idx_login_attempts_email_ip"`
	FailedCount  int
	LockedUntil  *time.Time
	LastFailedAt *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (loginAttemptRow) TableName() string { return "login_attempts" }

type loginAttemptRepository struct {
	db *gorm.DB
}

func NewLoginAttemptRepository(db *gorm.DB) port.LoginAttemptRepository {
	return &loginAttemptRepository{db: db}
}

func MigrateLoginAttempts(db *gorm.DB) error {
	return db.AutoMigrate(&loginAttemptRow{})
}

//メールと接続元のログイン失敗を1件探す
func (r *loginAttemptRepository) Find(ctx context.Context, email, ip string) (entity.LoginAttempt, error) {
	var row loginAttemptRow
	err := r.db.WithContext(ctx).Where("email = ? AND ip_address = ?", email, ip).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return entity.LoginAttempt{Email: email, IPAddress: ip}, nil
	}
	if err != nil {
		return entity.LoginAttempt{}, err
	}
	return attemptFromRow(row), nil
}

//ログイン失敗の回数と止めている時刻を保存する
func (r *loginAttemptRepository) Save(ctx context.Context, attempt entity.LoginAttempt) error {
	now := time.Now()
	if attempt.UpdatedAt.IsZero() {
		attempt.UpdatedAt = now
	}
	if attempt.ID == "" {
		attempt.ID = uuid.NewString()
		if attempt.CreatedAt.IsZero() {
			attempt.CreatedAt = now
		}
		row := attemptToRow(attempt)
		return r.db.WithContext(ctx).Create(&row).Error
	}
	return r.db.WithContext(ctx).Model(&loginAttemptRow{}).Where("id = ?", attempt.ID).Updates(map[string]any{
		"failed_count":   attempt.FailedCount,
		"locked_until":   timePtr(attempt.LockedUntil),
		"last_failed_at": timePtr(attempt.LastFailedAt),
		"updated_at":     attempt.UpdatedAt,
	}).Error
}

func attemptToRow(attempt entity.LoginAttempt) loginAttemptRow {
	return loginAttemptRow{
		ID:           attempt.ID,
		Email:        attempt.Email,
		IPAddress:    attempt.IPAddress,
		FailedCount:  attempt.FailedCount,
		LockedUntil:  timePtr(attempt.LockedUntil),
		LastFailedAt: timePtr(attempt.LastFailedAt),
		CreatedAt:    attempt.CreatedAt,
		UpdatedAt:    attempt.UpdatedAt,
	}
}

func attemptFromRow(row loginAttemptRow) entity.LoginAttempt {
	return entity.LoginAttempt{
		ID:           row.ID,
		Email:        row.Email,
		IPAddress:    row.IPAddress,
		FailedCount:  row.FailedCount,
		LockedUntil:  timeValue(row.LockedUntil),
		LastFailedAt: timeValue(row.LastFailedAt),
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
	}
}
