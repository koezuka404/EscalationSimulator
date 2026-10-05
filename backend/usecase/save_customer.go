package usecase

import (
	"context"

	"escalator/domain"
)

type Customers struct {
	repo domain.CustomerRepository
}

func NewCustomers(repo domain.CustomerRepository) *Customers {
	return &Customers{repo: repo}
}

//顧客の一覧を返す
func (c *Customers) List(ctx context.Context) ([]domain.Customer, error) {
	return c.repo.List(ctx)
}

//顧客の名前、プラン、目標時間を更新する
func (c *Customers) Update(ctx context.Context, id, name string, plan domain.Plan, slaMinutes int) (domain.Customer, error) {
	if _, err := c.repo.FindByID(ctx, id); err != nil {
		return domain.Customer{}, err
	}
	customer, err := domain.NewCustomer(name, plan, slaMinutes)
	if err != nil {
		return domain.Customer{}, err
	}
	customer.ID = id
	if err := c.repo.Update(ctx, customer); err != nil {
		return domain.Customer{}, err
	}
	return customer, nil
}
