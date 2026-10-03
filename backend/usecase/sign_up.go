package usecase

import (
	"context"

	"golang.org/x/crypto/bcrypt"

	"escalator/domain"
)

// SignUp は申請者を登録する。
type SignUp struct {
	users domain.UserRepository
	cost  int
}

func NewSignUp(users domain.UserRepository, cost int) *SignUp {
	return &SignUp{users: users, cost: cost}
}

func (s *SignUp) Execute(ctx context.Context, name, email, password string) (domain.User, error) {
	user, err := domain.NewApplicant(name, email, password)
	if err != nil {
		return domain.User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.cost)
	if err != nil {
		return domain.User{}, err
	}
	user.PasswordHash = string(hash)
	return s.users.Save(ctx, user)
}
