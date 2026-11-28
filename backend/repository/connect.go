package repository

import (
	"gorm.io/gorm"

	"escalator/db"
)

//データベースを開き、テーブルを作る
func Connect(databaseURL string) (*gorm.DB, error) {
	conn, err := db.Open(databaseURL)
	if err != nil {
		return nil, err
	}
	if err := Migrate(conn); err != nil {
		db.Close(conn)
		return nil, err
	}
	return conn, nil
}
