package usecase

import (
	"context"

	"escalator/entity"
	"escalator/repository"
)

type ClaimNextTicket struct {
	claims  repository.ClaimStore
	queue   repository.TicketQueue
	current *CurrentUser
}

func NewClaimNextTicket(claims repository.ClaimStore, queue repository.TicketQueue, current *CurrentUser) *ClaimNextTicket {
	return &ClaimNextTicket{claims: claims, queue: queue, current: current}
}

//担当者がいちばん先の対応待ちを引き取る
func (c *ClaimNextTicket) Execute(ctx context.Context, authorization string) (entity.Ticket, error) {
	actor, err := c.current.Execute(ctx, authorization)
	if err != nil {
		return entity.Ticket{}, err
	}
	if actor.Role != entity.RoleAgent {
		return entity.Ticket{}, entity.ErrClaimAgent
	}
	return c.claims.ClaimNext(ctx, actor.ID, c.queue)
}
