package usecase

import (
	"context"

	"golang.org/x/crypto/bcrypt"

	"escalator/entity"
	"escalator/repository"
)

type CreateAgent struct {
	users   repository.UserRepository
	current *CurrentUser
	cost    int
}

func NewCreateAgent(users repository.UserRepository, current *CurrentUser, cost int) *CreateAgent {
	return &CreateAgent{users: users, current: current, cost: cost}
}

//管理者が担当者を登録する
func (c *CreateAgent) Execute(ctx context.Context, authorization, name, email, password string) (entity.User, error) {
	actor, err := c.current.Execute(ctx, authorization)
	if err != nil {
		return entity.User{}, err
	}
	if actor.Role != entity.RoleAdmin {
		return entity.User{}, entity.ErrForbidden
	}
	user, err := entity.NewAgent(name, email, password)
	if err != nil {
		return entity.User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), c.cost)
	if err != nil {
		return entity.User{}, err
	}
	user.PasswordHash = string(hash)
	return c.users.SaveAgent(ctx, user)
}
