package usecase

import (
	"context"

	"escalator/entity"
	"escalator/repository"
)

type Customers struct {
	repo repository.CustomerRepository
}

func NewCustomers(repo repository.CustomerRepository) *Customers {
	return &Customers{repo: repo}
}

//顧客の一覧を返す
func (c *Customers) List(ctx context.Context) ([]entity.Customer, error) {
	return c.repo.List(ctx)
}

//顧客の名前、プラン、目標時間を更新する
func (c *Customers) Update(ctx context.Context, id, name string, plan entity.Plan, slaMinutes int) (entity.Customer, error) {
	if _, err := c.repo.FindByID(ctx, id); err != nil {
		return entity.Customer{}, err
	}
	customer, err := entity.NewCustomer(name, plan, slaMinutes)
	if err != nil {
		return entity.Customer{}, err
	}
	customer.ID = id
	if err := c.repo.Update(ctx, customer); err != nil {
		return entity.Customer{}, err
	}
	return customer, nil
}
