package domain

import "context"

type TicketQueue interface {
	Enqueue(ctx context.Context, ticketID string, score int) error
	PopMax(ctx context.Context) (ticketID string, score int, ok bool, err error)
}
