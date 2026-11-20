package usecase

import (
	"context"
	"log"
	"time"

	"escalator/entity"
	"escalator/port"
	"escalator/rule"
)

type CloseTicket struct {
	tickets port.TicketRepository
	queue   port.TicketQueue
	current *CurrentUser
	notices Notifier
	audit   *AuditLog
}

func NewCloseTicket(tickets port.TicketRepository, queue port.TicketQueue, current *CurrentUser, notices Notifier, audit *AuditLog) *CloseTicket {
	return &CloseTicket{tickets: tickets, queue: queue, current: current, notices: notices, audit: audit}
}

// 対応中のチケットを完了にする
func (c *CloseTicket) Execute(ctx context.Context, authorization, ticketID, comment string) (entity.Ticket, error) {
	comment, err := rule.CloseCommentText(comment)
	if err != nil {
		return entity.Ticket{}, err
	}
	actor, err := c.current.Execute(ctx, authorization)
	if err != nil {
		return entity.Ticket{}, err
	}
	if actor.Role != entity.RoleAgent && actor.Role != entity.RoleAdmin {
		return entity.Ticket{}, rule.ErrCloseForbidden
	}
	ticket, err := c.tickets.FindByID(ctx, ticketID)
	if err != nil {
		return entity.Ticket{}, err
	}
	if actor.Role == entity.RoleAgent && actor.ID != ticket.AssigneeID {
		return entity.Ticket{}, rule.ErrCloseForbidden
	}
	if ticket.Status != entity.TicketInProgress {
		return entity.Ticket{}, rule.ErrNotInProgress
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
	c.audit.Record(ctx, actor.ID, entity.AuditTicketClosed, "ticket", closed.ID, "チケットを完了しました")
	return closed, nil
}
