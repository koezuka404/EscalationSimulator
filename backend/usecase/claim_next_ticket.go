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
	notices Notifier
}

func NewClaimNextTicket(claims repository.ClaimStore, queue repository.TicketQueue, current *CurrentUser, notices Notifier) *ClaimNextTicket {
	return &ClaimNextTicket{claims: claims, queue: queue, current: current, notices: notices}
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
	ticket, err := c.claims.ClaimNext(ctx, actor.ID, c.queue)
	if err != nil {
		return entity.Ticket{}, err
	}
	publish(c.notices, true, ticket.CreatedBy, map[string]any{
		"type":          "ticket_claimed",
		"ticket_id":     ticket.ID,
		"title":         ticket.Title,
		"assignee_id":   actor.ID,
		"assignee_name": actor.Name,
	})
	return ticket, nil
}
