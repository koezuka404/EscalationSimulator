package redis

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

type Bus struct {
	client *goredis.Client
	origin string
}

func NewBus(order *TicketOrder) *Bus {
	return &Bus{client: order.client, origin: uuid.NewString()}
}

func (b *Bus) Origin() string {
	return b.origin
}

//他のサーバーへ知らせの中身を渡す
func (b *Bus) Publish(ctx context.Context, channel string, payload []byte) error {
	pubCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return b.client.Publish(pubCtx, channel, payload).Err()
}

//他のサーバーからの知らせを受け取る
func (b *Bus) Listen(ctx context.Context, channels []string, on func(channel string, payload []byte)) {
	for {
		if ctx.Err() != nil {
			return
		}
		sub := b.client.Subscribe(ctx, channels...)
		if _, err := sub.Receive(ctx); err != nil {
			log.Printf("他のサーバーからの知らせを受け始められませんでした: %v", err)
			_ = sub.Close()
			if !wait(ctx, 3*time.Second) {
				return
			}
			continue
		}
		messages := sub.Channel()
		stopped := false
		for !stopped {
			select {
			case <-ctx.Done():
				_ = sub.Close()
				return
			case message, ok := <-messages:
				if !ok {
					_ = sub.Close()
					stopped = true
					break
				}
				on(message.Channel, []byte(message.Payload))
			}
		}
	}
}

func wait(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
