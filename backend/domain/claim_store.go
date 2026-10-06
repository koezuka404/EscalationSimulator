package domain

import "context"

type ClaimStore interface {
	ClaimNext(ctx context.Context, agentID string, queue TicketQueue) (Ticket, error)
}
