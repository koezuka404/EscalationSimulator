package usecase

import (
	"context"
	"errors"
	"time"
)

var (
	ErrHealthDatabase = errors.New("データベースに接続できませんでした。しばらくしてから、もう一度試してください")
	ErrHealthQueue    = errors.New("待ち順の保存先に接続できませんでした。しばらくしてから、もう一度試してください")
	ErrHealthBoth     = errors.New("データベースと待ち順の保存先に接続できませんでした。しばらくしてから、もう一度試してください")
)

type Pinger interface {
	Ping(ctx context.Context) error
}

type CheckHealth struct {
	database Pinger
	queue    Pinger
}

func NewCheckHealth(database, queue Pinger) *CheckHealth {
	return &CheckHealth{database: database, queue: queue}
}

//データベースと待ち順の保存先が応答するか確かめる
func (c *CheckHealth) Execute(ctx context.Context) error {
	dbErr := ping(ctx, c.database)
	queueErr := ping(ctx, c.queue)
	switch {
	case dbErr != nil && queueErr != nil:
		return ErrHealthBoth
	case dbErr != nil:
		return ErrHealthDatabase
	case queueErr != nil:
		return ErrHealthQueue
	default:
		return nil
	}
}

func ping(ctx context.Context, target Pinger) error {
	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return target.Ping(pingCtx)
}
