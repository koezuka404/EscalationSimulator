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
	RevokeByHash(ctx context.Context, tokenHash string) error
	// Rotate は提示された印を無効にし、同じ系統の新しい印を保存する。
	// 無効済みの印が再使用されたときは、その系統をすべて無効にして reused を true にする。
	Rotate(ctx context.Context, oldHash string, next Session, now time.Time) (userID string, reused bool, err error)
}
