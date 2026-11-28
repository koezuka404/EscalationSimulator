package db

import (
	"context"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

//データベースへ接続する
func Open(databaseURL string) (*gorm.DB, error) {
	conn, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("データベースに接続できませんでした: %w", err)
	}
	if _, err := conn.DB(); err != nil {
		return nil, fmt.Errorf("データベースに接続できませんでした: %w", err)
	}
	return conn, nil
}

//データベースとの接続を切る
func Close(conn *gorm.DB) {
	if conn == nil {
		return
	}
	sqlDB, err := conn.DB()
	if err != nil {
		return
	}
	sqlDB.Close()
}

type Probe struct {
	conn *gorm.DB
}

func NewProbe(conn *gorm.DB) Probe {
	return Probe{conn: conn}
}

//データベースが応答するか確かめる
func (p Probe) Ping(ctx context.Context) error {
	sqlDB, err := p.conn.DB()
	if err != nil {
		return fmt.Errorf("データベースに接続できませんでした: %w", err)
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("データベースに接続できませんでした: %w", err)
	}
	return nil
}
