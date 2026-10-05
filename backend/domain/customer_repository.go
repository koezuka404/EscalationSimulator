package domain

import (
	"context"
	"errors"
)

var ErrCustomerNotFound = errors.New("指定した顧客は見つかりませんでした")

type CustomerRepository interface {
	Save(ctx context.Context, customer Customer) (Customer, error)
	FindByID(ctx context.Context, id string) (Customer, error)
	List(ctx context.Context) ([]Customer, error)
	Update(ctx context.Context, customer Customer) error
}
