package usecase

import (
	"context"

	"escalator/entity"
	"escalator/port"
	"escalator/rule"
)

type LinkApplicant struct {
	users     port.UserRepository
	customers port.CustomerRepository
	current   *CurrentUser
	audit     *AuditLog
}

func NewLinkApplicant(users port.UserRepository, customers port.CustomerRepository, current *CurrentUser, audit *AuditLog) *LinkApplicant {
	return &LinkApplicant{users: users, customers: customers, current: current, audit: audit}
}

//管理者だけが、申請者を顧客に結びつける
func (l *LinkApplicant) Execute(ctx context.Context, authorization, userID, customerID string) (entity.User, error) {
	actor, err := l.current.Execute(ctx, authorization)
	if err != nil {
		return entity.User{}, err
	}
	if actor.Role != entity.RoleAdmin {
		return entity.User{}, rule.ErrForbidden
	}
	user, err := l.users.FindByID(ctx, userID)
	if err != nil {
		return entity.User{}, err
	}
	linked, err := rule.LinkCustomer(user, customerID)
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
	l.audit.Record(ctx, actor.ID, entity.AuditCustomerChanged, "user", linked.ID, "申請者の所属顧客を変更しました")
	return linked, nil
}
