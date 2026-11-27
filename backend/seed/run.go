package seed

import (
	"context"
	"fmt"

	"escalator/port"
)

//サンプルの顧客と利用者を入れる
func Run(ctx context.Context, customers port.CustomerRepository, users port.UserRepository, password string, passwords port.PasswordHasher) error {
	if err := SeedCustomersIfEmpty(ctx, customers); err != nil {
		return fmt.Errorf("サンプルの顧客を登録できませんでした: %w", err)
	}
	if err := SeedUsersIfEmpty(ctx, users, customers, password, passwords); err != nil {
		return fmt.Errorf("サンプルの利用者を登録できませんでした: %w", err)
	}
	return nil
}
