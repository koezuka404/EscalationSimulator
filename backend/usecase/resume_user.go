package usecase

import (
	"context"

	"escalator/entity"
	"escalator/port"
	"escalator/rule"
)

type ResumeUser struct {
	users   port.UserRepository
	current *CurrentUser
}

func NewResumeUser(users port.UserRepository, current *CurrentUser) *ResumeUser {
	return &ResumeUser{users: users, current: current}
}

// 管理者だけが、停止中の利用者を利用中に戻す
func (r *ResumeUser) Execute(ctx context.Context, authorization, userID string) (entity.User, error) {
	actor, err := r.current.Execute(ctx, authorization)
	if err != nil {
		return entity.User{}, err
	}
	if actor.Role != entity.RoleAdmin {
		return entity.User{}, rule.ErrForbidden
	}
	user, err := r.users.FindByID(ctx, userID)
	if err != nil {
		return entity.User{}, err
	}
	resumed, err := rule.Resume(user)
	if err != nil {
		return entity.User{}, err
	}
	if err := r.users.UpdateStatus(ctx, resumed); err != nil {
		return entity.User{}, err
	}
	return resumed, nil
}
