package usecase

import (
	"context"

	"escalator/entity"
	"escalator/port"
	"escalator/rule"
)

type Customers struct {
	repo    port.CustomerRepository
	current *CurrentUser
	audit   *AuditLog
}

func NewCustomers(repo port.CustomerRepository, current *CurrentUser, audit *AuditLog) *Customers {
	return &Customers{repo: repo, current: current, audit: audit}
}

func (c *Customers) allow(ctx context.Context, authorization string) (entity.User, error) {
	actor, err := c.current.Execute(ctx, authorization)
	if err != nil {
		return entity.User{}, err
	}
	if actor.Role != entity.RoleAdmin {
		return entity.User{}, rule.ErrForbidden
	}
	return actor, nil
}

//管理者だけが顧客の一覧を返す
func (c *Customers) List(ctx context.Context, authorization string) ([]entity.Customer, error) {
	if _, err := c.allow(ctx, authorization); err != nil {
		return nil, err
	}
	return c.repo.List(ctx)
}

//管理者だけが顧客を登録する
func (c *Customers) Create(ctx context.Context, authorization, name string, plan entity.Plan, slaMinutes int) (entity.Customer, error) {
	if _, err := c.allow(ctx, authorization); err != nil {
		return entity.Customer{}, err
	}
	customer, err := rule.NewCustomer(name, plan, slaMinutes)
	if err != nil {
		return entity.Customer{}, err
	}
	return c.repo.Save(ctx, customer)
}

//管理者だけが顧客の名前、プラン、目標時間を更新する
func (c *Customers) Update(ctx context.Context, authorization, id, name string, plan entity.Plan, slaMinutes int) (entity.Customer, error) {
	actor, err := c.allow(ctx, authorization)
	if err != nil {
		return entity.Customer{}, err
	}
	if _, err = c.repo.FindByID(ctx, id); err != nil {
		return entity.Customer{}, err
	}
	customer, err := rule.NewCustomer(name, plan, slaMinutes)
	if err != nil {
		return entity.Customer{}, err
	}
	customer.ID = id
	if err := c.repo.Update(ctx, customer); err != nil {
		return entity.Customer{}, err
	}
	c.audit.Record(ctx, actor.ID, entity.AuditCustomerChanged, "customer", customer.ID, "顧客の名前、プラン、目標時間を更新しました")
	return customer, nil
}
