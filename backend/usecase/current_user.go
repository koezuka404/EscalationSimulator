package usecase

import (
	"context"

	"escalator/entity"
	"escalator/port"
	"escalator/rule"
)

type CurrentUser struct {
	users  port.UserRepository
	tokens port.TokenVerifier
}

func NewCurrentUser(users port.UserRepository, tokens port.TokenVerifier) *CurrentUser {
	return &CurrentUser{users: users, tokens: tokens}
}

//ログイン用トークンから、今の利用者を取る
func (c *CurrentUser) Execute(ctx context.Context, token string) (entity.User, error) {
	if token == "" {
		return entity.User{}, rule.ErrUnauthenticated
	}
	userID, authVersion, err := c.tokens.Parse(token)
	if err != nil || userID == "" {
		return entity.User{}, rule.ErrUnauthenticated
	}
	user, err := c.users.FindByID(ctx, userID)
	if err != nil || user.Status != entity.StatusActive || user.AuthVersion != authVersion {
		return entity.User{}, rule.ErrUnauthenticated
	}
	return user, nil
}
