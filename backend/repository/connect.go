package repository

import (
	"fmt"

	"gorm.io/gorm"

	"escalator/db"
)

//データベースを開き、テーブルを作る
func Connect(databaseURL string) (*gorm.DB, error) {
	conn, err := db.Open(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("データベースに接続できませんでした: %w", err)
	}
	sqlDB, err := conn.DB()
	if err != nil {
		return nil, fmt.Errorf("データベースに接続できませんでした: %w", err)
	}
	if err := Migrate(conn); err != nil {
		sqlDB.Close()
		return nil, err
	}
	return conn, nil
}
