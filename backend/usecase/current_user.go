package usecase

import (
	"context"
	"strings"

	"escalator/entity"
	"escalator/port"
	"escalator/rule"
	"escalator/usecase/crypto"
)

type CurrentUser struct {
	users    port.UserRepository
	secret   []byte
	issuer   string
	audience string
}

func NewCurrentUser(users port.UserRepository, secret []byte, issuer, audience string) *CurrentUser {
	return &CurrentUser{users: users, secret: secret, issuer: issuer, audience: audience}
}

// ログイン用トークンから、今の利用者を取る
func (c *CurrentUser) Execute(ctx context.Context, authorization string) (entity.User, error) {
	raw := bearerToken(authorization)
	if raw == "" {
		return entity.User{}, rule.ErrUnauthenticated
	}
	userID, authVersion, err := crypto.ParseAccess(c.secret, c.issuer, c.audience, raw)
	if err != nil || userID == "" {
		return entity.User{}, rule.ErrUnauthenticated
	}
	user, err := c.users.FindByID(ctx, userID)
	if err != nil || user.Status != entity.StatusActive || user.AuthVersion != authVersion {
		return entity.User{}, rule.ErrUnauthenticated
	}
	return user, nil
}

func bearerToken(authorization string) string {
	const prefix = "Bearer "
	value := strings.TrimSpace(authorization)
	if len(value) < len(prefix) || !strings.EqualFold(value[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(value[len(prefix):])
}
