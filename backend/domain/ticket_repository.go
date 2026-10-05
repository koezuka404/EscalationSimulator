package domain

import "context"

type TicketRepository interface {
	Save(ctx context.Context, ticket Ticket) (Ticket, error)
	ListOpen(ctx context.Context) ([]Ticket, error)
}
