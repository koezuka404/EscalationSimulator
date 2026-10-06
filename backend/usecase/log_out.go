package usecase

import (
	"context"

	"escalator/repository"
	"escalator/usecase/crypto"
)

type LogOut struct {
	sessions repository.SessionRepository
}

func NewLogOut(sessions repository.SessionRepository) *LogOut {
	return &LogOut{sessions: sessions}
}

//再ログイン用の印を無効にする
func (l *LogOut) Execute(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	return l.sessions.RevokeByHash(ctx, crypto.Hash(refreshToken))
}
