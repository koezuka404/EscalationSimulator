package redis

import (
	"context"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const limitKeyPrefix = "limit:"

const takeScript = `
local capacity = tonumber(ARGV[1])
local refill = tonumber(ARGV[2])
local now_ms = tonumber(ARGV[3])
local state = redis.call("HMGET", KEYS[1], "tokens", "ts")
local tokens = tonumber(state[1])
local ts = tonumber(state[2])
if tokens == nil or ts == nil then
	tokens = capacity
	ts = now_ms
end
local elapsed = now_ms - ts
if elapsed < 0 then
	elapsed = 0
end
tokens = math.min(capacity, tokens + (elapsed / 1000) * refill)
local allowed = 0
if tokens >= 1 then
	tokens = tokens - 1
	allowed = 1
end
redis.call("HSET", KEYS[1], "tokens", tostring(tokens), "ts", tostring(now_ms))
local ttl = 120
if refill > 0 then
	ttl = math.ceil(capacity / refill) + 60
end
if ttl < 120 then
	ttl = 120
end
redis.call("EXPIRE", KEYS[1], ttl)
return allowed
`

type Gate struct {
	client *goredis.Client
}

//回数制限の保存先を、待ち順と同じ接続にする
func NewGate(order *TicketOrder) *Gate {
	return &Gate{client: order.client}
}

//残回数を1つ減らし、残っていれば受け付ける
func (g *Gate) Take(ctx context.Context, key string, capacity int, perSecond float64) (bool, error) {
	result, err := g.client.Eval(ctx, takeScript, []string{limitKeyPrefix + key}, capacity, perSecond, time.Now().UnixMilli()).Int()
	if err != nil {
		return false, err
	}
	return result == 1, nil
}
