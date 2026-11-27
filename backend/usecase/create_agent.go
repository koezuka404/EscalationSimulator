package usecase

import (
	"context"

	"escalator/entity"
	"escalator/port"
	"escalator/rule"
)

type CreateAgent struct {
	users    port.UserRepository
	current  *CurrentUser
	password port.PasswordHasher
}

func NewCreateAgent(users port.UserRepository, current *CurrentUser, password port.PasswordHasher) *CreateAgent {
	return &CreateAgent{users: users, current: current, password: password}
}

//管理者が担当者を登録する
func (c *CreateAgent) Execute(ctx context.Context, authorization, name, email, password string) (entity.User, error) {
	actor, err := c.current.Execute(ctx, authorization)
	if err != nil {
		return entity.User{}, err
	}
	if actor.Role != entity.RoleAdmin {
		return entity.User{}, rule.ErrForbidden
	}
	user, err := rule.NewAgent(name, email, password)
	if err != nil {
		return entity.User{}, err
	}
	hash, err := c.password.Hash(password)
	if err != nil {
		return entity.User{}, err
	}
	user.PasswordHash = hash
	return c.users.SaveAgent(ctx, user)
}
