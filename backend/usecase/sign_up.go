package usecase

import (
	"context"

	"escalator/entity"
	"escalator/port"
	"escalator/rule"
)

type SignUp struct {
	users    port.UserRepository
	password port.PasswordHasher
}

func NewSignUp(users port.UserRepository, password port.PasswordHasher) *SignUp {
	return &SignUp{users: users, password: password}
}

//申請者を登録する
func (s *SignUp) Execute(ctx context.Context, name, email, password string) (entity.User, error) {
	user, err := rule.NewApplicant(name, email, password)
	if err != nil {
		return entity.User{}, err
	}
	hash, err := s.password.Hash(password)
	if err != nil {
		return entity.User{}, err
	}
	user.PasswordHash = hash
	return s.users.Save(ctx, user)
}
