package usecase

import (
	"context"
	"log"

	"escalator/entity"
	"escalator/repository"
)

type SuspendUser struct {
	users   repository.UserRepository
	current *CurrentUser
	screens ScreenConnection
}

func NewSuspendUser(users repository.UserRepository, current *CurrentUser, screens ScreenConnection) *SuspendUser {
	return &SuspendUser{users: users, current: current, screens: screens}
}

//管理者だけが利用者を停止し、その人の画面の接続を切る
func (s *SuspendUser) Execute(ctx context.Context, authorization, userID string) (entity.User, error) {
	actor, err := s.current.Execute(ctx, authorization)
	if err != nil {
		return entity.User{}, err
	}
	if actor.Role != entity.RoleAdmin {
		return entity.User{}, entity.ErrForbidden
	}
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return entity.User{}, err
	}
	if actor.ID == user.ID {
		return entity.User{}, entity.ErrSuspendSelf
	}
	suspended, err := user.Suspend()
	if err != nil {
		return entity.User{}, err
	}
	if err := s.users.UpdateStatus(ctx, suspended); err != nil {
		return entity.User{}, err
	}
	if s.screens != nil {
		s.screens.CloseUser(suspended.ID)
	}
	log.Printf("利用者を停止したため、その人の画面の接続を切りました。user_id=%s", suspended.ID)
	return suspended, nil
}
