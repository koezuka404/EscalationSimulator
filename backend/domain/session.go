package domain

import (
	"context"
	"time"
)

type Session struct {
	ID        string
	UserID    string
	TokenHash string
	FamilyID  string
	ExpiresAt time.Time
}

type SessionRepository interface {
	Save(ctx context.Context, session Session) error
}
