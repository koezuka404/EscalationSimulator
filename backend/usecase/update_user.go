package usecase

import (
	"context"

	"escalator/entity"
	"escalator/repository"
)

type LinkApplicant struct {
	users     repository.UserRepository
	customers repository.CustomerRepository
	current   *CurrentUser
}

func NewLinkApplicant(users repository.UserRepository, customers repository.CustomerRepository, current *CurrentUser) *LinkApplicant {
	return &LinkApplicant{users: users, customers: customers, current: current}
}

//管理者だけが、申請者を顧客に結びつける
func (l *LinkApplicant) Execute(ctx context.Context, authorization, userID, customerID string) (entity.User, error) {
	actor, err := l.current.Execute(ctx, authorization)
	if err != nil {
		return entity.User{}, err
	}
	if actor.Role != entity.RoleAdmin {
		return entity.User{}, entity.ErrForbidden
	}
	user, err := l.users.FindByID(ctx, userID)
	if err != nil {
		return entity.User{}, err
	}
	linked, err := user.LinkCustomer(customerID)
	if err != nil {
		return entity.User{}, err
	}
	if _, err := l.customers.FindByID(ctx, customerID); err != nil {
		return entity.User{}, err
	}
	if linked.AuthVersion == user.AuthVersion {
		return user, nil
	}
	if err := l.users.UpdateCustomer(ctx, linked); err != nil {
		return entity.User{}, err
	}
	return linked, nil
}
