package domain

import "context"

type UserRepository interface {
	Save(ctx context.Context, user User) (User, error)
	FindByEmail(ctx context.Context, email string) (User, error)
	UpdateLoginState(ctx context.Context, user User) error
}
