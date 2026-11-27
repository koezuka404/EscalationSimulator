package db

import (
	"context"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Open(databaseURL string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
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
		return err
	}
	return sqlDB.PingContext(ctx)
}
