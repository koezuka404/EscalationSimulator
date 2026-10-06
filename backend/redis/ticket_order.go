package redis

import (
	"context"
	"math"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const ticketQueueKey = "queue:tickets"

type TicketOrder struct {
	client *goredis.Client
}

//待ち順の保存先へつなぐ
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

//チケットのIDと点数を待ち順へ載せる
func (o *TicketOrder) Enqueue(ctx context.Context, ticketID string, score int) error {
	return o.client.ZAdd(ctx, ticketQueueKey, goredis.Z{
		Score:  float64(score),
		Member: ticketID,
	}).Err()
}

//点数がいちばん高い1件を取り待ち順から消す
func (o *TicketOrder) PopMax(ctx context.Context) (string, int, bool, error) {
	values, err := o.client.ZPopMax(ctx, ticketQueueKey, 1).Result()
	if err == goredis.Nil {
		return "", 0, false, nil
	}
	if err != nil {
		return "", 0, false, err
	}
	if len(values) == 0 {
		return "", 0, false, nil
	}
	id, ok := values[0].Member.(string)
	if !ok || id == "" {
		return "", 0, false, goredis.Nil
	}
	return id, int(math.Round(values[0].Score)), true, nil
}

//待ち順からチケットを外す
func (o *TicketOrder) Remove(ctx context.Context, ticketID string) error {
	return o.client.ZRem(ctx, ticketQueueKey, ticketID).Err()
}
