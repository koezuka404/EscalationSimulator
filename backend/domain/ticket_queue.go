package domain

import "context"

// TicketQueue は対応待ちの順番を持つ。点数が高いほど先。
type TicketQueue interface {
	Enqueue(ctx context.Context, ticketID string, score int) error
}
