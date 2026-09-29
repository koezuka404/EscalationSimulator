package domain

import (
	"context"
	"errors"
)

var ErrCustomerNotFound = errors.New("顧客が見つかりません")

// CustomerRepository は顧客の保存窓口。実装は外側が持つ。
type CustomerRepository interface {
	Save(ctx context.Context, customer Customer) (Customer, error)
	FindByID(ctx context.Context, id string) (Customer, error)
	List(ctx context.Context) ([]Customer, error)
	Update(ctx context.Context, customer Customer) error
}
