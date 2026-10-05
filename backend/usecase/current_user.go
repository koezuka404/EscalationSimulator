package usecase

import (
	"context"
	"strings"

	"escalator/domain"
	"escalator/infra/token"
)

type CurrentUser struct {
	users    domain.UserRepository
	secret   []byte
	issuer   string
	audience string
}

func NewCurrentUser(users domain.UserRepository, secret []byte, issuer, audience string) *CurrentUser {
	return &CurrentUser{users: users, secret: secret, issuer: issuer, audience: audience}
}

//ログイン用トークンから、今の利用者を取る
func (c *CurrentUser) Execute(ctx context.Context, authorization string) (domain.User, error) {
	raw := bearerToken(authorization)
	if raw == "" {
		return domain.User{}, domain.ErrUnauthenticated
	}
	userID, authVersion, err := token.ParseAccess(c.secret, c.issuer, c.audience, raw)
	if err != nil || userID == "" {
		return domain.User{}, domain.ErrUnauthenticated
	}
	user, err := c.users.FindByID(ctx, userID)
	if err != nil || user.Status != domain.StatusActive || user.AuthVersion != authVersion {
		return domain.User{}, domain.ErrUnauthenticated
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
