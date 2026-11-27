package usecase

import (
	"context"
	"errors"

	"escalator/entity"
	"escalator/port"
	"escalator/rule"
)

type LogOut struct {
	secrets  port.SecretTokens
	sessions port.SessionRepository
	audit    *AuditLog
}

func NewLogOut(sessions port.SessionRepository, secrets port.SecretTokens, audit *AuditLog) *LogOut {
	return &LogOut{secrets: secrets, sessions: sessions, audit: audit}
}

//再ログイン用の印を無効にする
func (l *LogOut) Execute(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	hash := l.secrets.Hash(refreshToken)
	userID, err := l.sessions.FindUserIDByHash(ctx, hash)
	if err != nil && !errors.Is(err, rule.ErrInvalidRefresh) {
		return err
	}
	if err := l.sessions.RevokeByHash(ctx, hash); err != nil {
		return err
	}
	if userID != "" {
		l.audit.Record(ctx, userID, entity.AuditLogout, "user", userID, "ログアウトしました")
	}
	return nil
}
