package usecase

import (
	"context"

	"escalator/domain"
)

type LinkApplicant struct {
	users     domain.UserRepository
	customers domain.CustomerRepository
	current   *CurrentUser
}

func NewLinkApplicant(users domain.UserRepository, customers domain.CustomerRepository, current *CurrentUser) *LinkApplicant {
	return &LinkApplicant{users: users, customers: customers, current: current}
}

// 管理者だけが、申請者を顧客に結びつける。
func (l *LinkApplicant) Execute(ctx context.Context, authorization, userID, customerID string) (domain.User, error) {
	actor, err := l.current.Execute(ctx, authorization)
	if err != nil {
		return domain.User{}, err
	}
	if actor.Role != domain.RoleAdmin {
		return domain.User{}, domain.ErrForbidden
	}
	user, err := l.users.FindByID(ctx, userID)
	if err != nil {
		return domain.User{}, err
	}
	linked, err := user.LinkCustomer(customerID)
	if err != nil {
		return domain.User{}, err
	}
	if _, err := l.customers.FindByID(ctx, customerID); err != nil {
		return domain.User{}, err
	}
	if linked.AuthVersion == user.AuthVersion {
		return user, nil
	}
	if err := l.users.UpdateCustomer(ctx, linked); err != nil {
		return domain.User{}, err
	}
	return linked, nil
}
