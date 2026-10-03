package usecase

import (
	"context"

	"escalator/domain"
	"escalator/infra/token"
)

// LogOut は再ログイン用の印を無効にする。
type LogOut struct {
	sessions domain.SessionRepository
}

func NewLogOut(sessions domain.SessionRepository) *LogOut {
	return &LogOut{sessions: sessions}
}

func (l *LogOut) Execute(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	return l.sessions.RevokeByHash(ctx, token.Hash(refreshToken))
}
