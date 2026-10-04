package domain

import "context"

type UserRepository interface {
	Save(ctx context.Context, user User) (User, error)
	FindByID(ctx context.Context, id string) (User, error)
	FindByEmail(ctx context.Context, email string) (User, error)
	UpdateLoginState(ctx context.Context, user User) error
	UpdateCustomer(ctx context.Context, user User) error
	BumpAuthVersion(ctx context.Context, id string) error
}
