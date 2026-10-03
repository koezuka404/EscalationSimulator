package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"

	"escalator/domain"
)

type userRow struct {
	ID           string `gorm:"primaryKey"`
	Email        string `gorm:"uniqueIndex"`
	PasswordHash string
	Name         string
	Role         string
	CustomerID   string
	Status       string
	AuthVersion  int
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (userRow) TableName() string { return "users" }

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func MigrateUsers(db *gorm.DB) error {
	return db.AutoMigrate(&userRow{})
}

func (r *UserRepository) Save(ctx context.Context, user domain.User) (domain.User, error) {
	if user.ID == "" {
		user.ID = uuid.NewString()
	}
	row := userToRow(user)
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.User{}, domain.ErrEmailTaken
		}
		return domain.User{}, err
	}
	return userFromRow(row), nil
}

func userToRow(user domain.User) userRow {
	return userRow{
		ID:           user.ID,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		Name:         user.Name,
		Role:         string(user.Role),
		CustomerID:   user.CustomerID,
		Status:       string(user.Status),
		AuthVersion:  user.AuthVersion,
	}
}

func userFromRow(row userRow) domain.User {
	return domain.User{
		ID:           row.ID,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
		Name:         row.Name,
		Role:         domain.Role(row.Role),
		CustomerID:   row.CustomerID,
		Status:       domain.AccountStatus(row.Status),
		AuthVersion:  row.AuthVersion,
	}
}
