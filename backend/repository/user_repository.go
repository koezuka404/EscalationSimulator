package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"escalator/entity"
)

type userRow struct {
	ID               string `gorm:"primaryKey"`
	Email            string `gorm:"uniqueIndex"`
	PasswordHash     string
	Name             string
	Role             string
	CustomerID       string
	Status           string
	AuthVersion      int
	FailedLoginCount int
	LockedUntil      *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (userRow) TableName() string { return "users" }

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func MigrateUsers(db *gorm.DB) error {
	return db.AutoMigrate(&userRow{})
}

func (r *userRepository) Save(ctx context.Context, user entity.User) (entity.User, error) {
	if user.ID == "" {
		user.ID = uuid.NewString()
	}
	row := userToRow(user)
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return entity.User{}, entity.ErrEmailTaken
		}
		return entity.User{}, err
	}
	return userFromRow(row), nil
}

func (r *userRepository) FindByID(ctx context.Context, id string) (entity.User, error) {
	var row userRow
	err := r.db.WithContext(ctx).First(&row, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return entity.User{}, entity.ErrUserNotFound
	}
	if err != nil {
		return entity.User{}, err
	}
	return userFromRow(row), nil
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (entity.User, error) {
	var row userRow
	err := r.db.WithContext(ctx).First(&row, "email = ?", email).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return entity.User{}, entity.ErrUserNotFound
	}
	if err != nil {
		return entity.User{}, err
	}
	return userFromRow(row), nil
}

func (r *userRepository) UpdateLoginState(ctx context.Context, user entity.User) error {
	var lockedUntil any
	if user.LockedUntil.IsZero() {
		lockedUntil = nil
	} else {
		lockedUntil = user.LockedUntil
	}
	return r.db.WithContext(ctx).Model(&userRow{}).Where("id = ?", user.ID).Updates(map[string]any{
		"failed_login_count": user.FailedLoginCount,
		"locked_until":       lockedUntil,
	}).Error
}

func (r *userRepository) UpdateCustomer(ctx context.Context, user entity.User) error {
	result := r.db.WithContext(ctx).Model(&userRow{}).Where("id = ?", user.ID).Updates(map[string]any{
		"customer_id":  user.CustomerID,
		"auth_version": user.AuthVersion,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return entity.ErrUserNotFound
	}
	return nil
}

func (r *userRepository) BumpAuthVersion(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&userRow{}).Where("id = ?", id).Update("auth_version", gorm.Expr("auth_version + 1")).Error
}

func userToRow(user entity.User) userRow {
	return userRow{
		ID:               user.ID,
		Email:            user.Email,
		PasswordHash:     user.PasswordHash,
		Name:             user.Name,
		Role:             string(user.Role),
		CustomerID:       user.CustomerID,
		Status:           string(user.Status),
		AuthVersion:      user.AuthVersion,
		FailedLoginCount: user.FailedLoginCount,
		LockedUntil:      timePtr(user.LockedUntil),
	}
}

func userFromRow(row userRow) entity.User {
	return entity.User{
		ID:               row.ID,
		Email:            row.Email,
		PasswordHash:     row.PasswordHash,
		Name:             row.Name,
		Role:             entity.Role(row.Role),
		CustomerID:       row.CustomerID,
		Status:           entity.AccountStatus(row.Status),
		AuthVersion:      row.AuthVersion,
		FailedLoginCount: row.FailedLoginCount,
		LockedUntil:      timeValue(row.LockedUntil),
	}
}

func timePtr(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	return &value
}

func timeValue(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return *value
}
