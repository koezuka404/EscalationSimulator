package repository

import (
	"fmt"

	"gorm.io/gorm"
)

func Migrate(db *gorm.DB) error {
	if err := MigrateCustomers(db); err != nil {
		return fmt.Errorf("顧客用のテーブルを作成できませんでした: %w", err)
	}
	if err := MigrateUsers(db); err != nil {
		return fmt.Errorf("利用者用のテーブルを作成できませんでした: %w", err)
	}
	if err := MigrateSessions(db); err != nil {
		return fmt.Errorf("ログイン用のテーブルを作成できませんでした: %w", err)
	}
	if err := MigrateTickets(db); err != nil {
		return fmt.Errorf("チケット用のテーブルを作成できませんでした: %w", err)
	}
	if err := MigrateAgentStatuses(db); err != nil {
		return fmt.Errorf("担当者の稼働用のテーブルを作成できませんでした: %w", err)
	}
	return nil
}
