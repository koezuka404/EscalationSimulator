package usecase

import (
	"context"

	"escalator/entity"
	"escalator/repository"
)

type ListUsers struct {
	users   repository.UserRepository
	current *CurrentUser
}

func NewListUsers(users repository.UserRepository, current *CurrentUser) *ListUsers {
	return &ListUsers{users: users, current: current}
}

//管理者が利用者の一覧を取る
func (l *ListUsers) Execute(ctx context.Context, authorization string) ([]entity.User, error) {
	actor, err := l.current.Execute(ctx, authorization)
	if err != nil {
		return nil, err
	}
	if actor.Role != entity.RoleAdmin {
		return nil, entity.ErrForbidden
	}
	return l.users.List(ctx)
}
