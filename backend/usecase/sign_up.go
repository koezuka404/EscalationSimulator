package usecase

import (
	"context"

	"golang.org/x/crypto/bcrypt"

	"escalator/entity"
	"escalator/repository"
)

type SignUp struct {
	users repository.UserRepository
	cost  int
}

func NewSignUp(users repository.UserRepository, cost int) *SignUp {
	return &SignUp{users: users, cost: cost}
}

//申請者を登録する
func (s *SignUp) Execute(ctx context.Context, name, email, password string) (entity.User, error) {
	user, err := entity.NewApplicant(name, email, password)
	if err != nil {
		return entity.User{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.cost)
	if err != nil {
		return entity.User{}, err
	}
	user.PasswordHash = string(hash)
	return s.users.Save(ctx, user)
}
