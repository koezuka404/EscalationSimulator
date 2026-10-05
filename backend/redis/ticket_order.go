package redis

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const ticketQueueKey = "queue:tickets"

// TicketOrder は対応待ちのチケット ID を点数つきで持つ。
type TicketOrder struct {
	client *goredis.Client
}

func Open(rawURL string) (*TicketOrder, error) {
	opt, err := goredis.ParseURL(rawURL)
	if err != nil {
		return nil, err
	}
	client := goredis.NewClient(opt)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}
	return &TicketOrder{client: client}, nil
}

func (o *TicketOrder) Close() error {
	return o.client.Close()
}

// Enqueue はチケットを待ち順へ載せる。点数は計算した点数のまま入れる。
func (o *TicketOrder) Enqueue(ctx context.Context, ticketID string, score int) error {
	return o.client.ZAdd(ctx, ticketQueueKey, goredis.Z{
		Score:  float64(score),
		Member: ticketID,
	}).Err()
}
