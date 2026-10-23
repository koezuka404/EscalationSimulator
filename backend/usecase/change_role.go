package usecase

import (
	"context"

	"escalator/entity"
	"escalator/repository"
)

type ChangeRole struct {
	users     repository.UserRepository
	customers repository.CustomerRepository
	current   *CurrentUser
}

func NewChangeRole(users repository.UserRepository, customers repository.CustomerRepository, current *CurrentUser) *ChangeRole {
	return &ChangeRole{users: users, customers: customers, current: current}
}

//管理者だけが利用者の役割を変える
func (c *ChangeRole) Execute(ctx context.Context, authorization, userID, role, customerID string) (entity.User, error) {
	actor, err := c.current.Execute(ctx, authorization)
	if err != nil {
		return entity.User{}, err
	}
	if actor.Role != entity.RoleAdmin {
		return entity.User{}, entity.ErrForbidden
	}
	user, err := c.users.FindByID(ctx, userID)
	if err != nil {
		return entity.User{}, err
	}
	if actor.ID == user.ID {
		return entity.User{}, entity.ErrChangeOwnRole
	}
	changed, err := user.ChangeRole(entity.Role(role), customerID)
	if err != nil {
		return entity.User{}, err
	}
	if changed.Role == entity.RoleApplicant {
		if _, err := c.customers.FindByID(ctx, changed.CustomerID); err != nil {
			return entity.User{}, err
		}
	}
	if err := c.users.UpdateRole(ctx, changed); err != nil {
		return entity.User{}, err
	}
	return changed, nil
}
