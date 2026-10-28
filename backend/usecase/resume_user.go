package usecase

import (
	"context"

	"escalator/entity"
	"escalator/repository"
)

type ResumeUser struct {
	users   repository.UserRepository
	current *CurrentUser
}

func NewResumeUser(users repository.UserRepository, current *CurrentUser) *ResumeUser {
	return &ResumeUser{users: users, current: current}
}

//管理者だけが、停止中の利用者を利用中に戻す
func (r *ResumeUser) Execute(ctx context.Context, authorization, userID string) (entity.User, error) {
	actor, err := r.current.Execute(ctx, authorization)
	if err != nil {
		return entity.User{}, err
	}
	if actor.Role != entity.RoleAdmin {
		return entity.User{}, entity.ErrForbidden
	}
	user, err := r.users.FindByID(ctx, userID)
	if err != nil {
		return entity.User{}, err
	}
	resumed, err := user.Resume()
	if err != nil {
		return entity.User{}, err
	}
	if err := r.users.UpdateStatus(ctx, resumed); err != nil {
		return entity.User{}, err
	}
	return resumed, nil
}
