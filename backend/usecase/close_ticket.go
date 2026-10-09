package usecase

import (
	"context"
	"log"
	"time"

	"escalator/entity"
	"escalator/repository"
)

type CloseTicket struct {
	tickets repository.TicketRepository
	queue   repository.TicketQueue
	current *CurrentUser
	notices Notifier
}

func NewCloseTicket(tickets repository.TicketRepository, queue repository.TicketQueue, current *CurrentUser, notices Notifier) *CloseTicket {
	return &CloseTicket{tickets: tickets, queue: queue, current: current, notices: notices}
}

//対応中のチケットを完了にする
func (c *CloseTicket) Execute(ctx context.Context, authorization, ticketID, comment string) (entity.Ticket, error) {
	comment, err := entity.CloseCommentText(comment)
	if err != nil {
		return entity.Ticket{}, err
	}
	actor, err := c.current.Execute(ctx, authorization)
	if err != nil {
		return entity.Ticket{}, err
	}
	if actor.Role != entity.RoleAgent && actor.Role != entity.RoleAdmin {
		return entity.Ticket{}, entity.ErrCloseForbidden
	}
	ticket, err := c.tickets.FindByID(ctx, ticketID)
	if err != nil {
		return entity.Ticket{}, err
	}
	if actor.Role == entity.RoleAgent && actor.ID != ticket.AssigneeID {
		return entity.Ticket{}, entity.ErrCloseForbidden
	}
	if ticket.Status != entity.TicketInProgress {
		return entity.Ticket{}, entity.ErrNotInProgress
	}
	closed, err := c.tickets.Close(ctx, ticket.ID, comment, time.Now())
	if err != nil {
		return entity.Ticket{}, err
	}
	removeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := c.queue.Remove(removeCtx, closed.ID); err != nil {
		log.Printf("待ち順から外せませんでした。チケットは完了しています。id=%s: %v", closed.ID, err)
	}
	publish(c.notices, true, closed.CreatedBy, map[string]any{
		"type":      "ticket_closed",
		"ticket_id": closed.ID,
		"title":     closed.Title,
	})
	return closed, nil
}
