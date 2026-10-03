package domain

import "context"

type UserRepository interface {
	Save(ctx context.Context, user User) (User, error)
}
