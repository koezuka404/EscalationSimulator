package usecase

import (
	"context"

	"escalator/domain"
)

type ClaimNextTicket struct {
	claims  domain.ClaimStore
	queue   domain.TicketQueue
	current *CurrentUser
}

func NewClaimNextTicket(claims domain.ClaimStore, queue domain.TicketQueue, current *CurrentUser) *ClaimNextTicket {
	return &ClaimNextTicket{claims: claims, queue: queue, current: current}
}

//担当者がいちばん先の対応待ちを引き取る
func (c *ClaimNextTicket) Execute(ctx context.Context, authorization string) (domain.Ticket, error) {
	actor, err := c.current.Execute(ctx, authorization)
	if err != nil {
		return domain.Ticket{}, err
	}
	if actor.Role != domain.RoleAgent {
		return domain.Ticket{}, domain.ErrClaimAgent
	}
	return c.claims.ClaimNext(ctx, actor.ID, c.queue)
}
