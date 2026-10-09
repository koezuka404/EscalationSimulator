package usecase

import (
	"context"
	"log"
	"time"

	"escalator/entity"
	"escalator/repository"
)

type ChangeSeverity struct {
	tickets   repository.TicketRepository
	customers repository.CustomerRepository
	current   *CurrentUser
	queue     repository.TicketQueue
	notices   Notifier
}

func NewChangeSeverity(tickets repository.TicketRepository, customers repository.CustomerRepository, current *CurrentUser, queue repository.TicketQueue, notices Notifier) *ChangeSeverity {
	return &ChangeSeverity{tickets: tickets, customers: customers, current: current, queue: queue, notices: notices}
}

//緊急度を変え、対応待ちなら待ち順の点数も直す
func (c *ChangeSeverity) Execute(ctx context.Context, authorization, ticketID, reason string, severity int) (entity.Ticket, error) {
	reason, err := entity.SeverityReason(reason)
	if err != nil {
		return entity.Ticket{}, err
	}
	if severity < 1 || severity > 4 {
		return entity.Ticket{}, entity.ErrInvalidSeverity
	}
	actor, err := c.current.Execute(ctx, authorization)
	if err != nil {
		return entity.Ticket{}, err
	}
	if actor.Role != entity.RoleAgent && actor.Role != entity.RoleAdmin {
		return entity.Ticket{}, entity.ErrSeverityRole
	}
	ticket, err := c.tickets.FindByID(ctx, ticketID)
	if err != nil {
		return entity.Ticket{}, err
	}
	if err := entity.CanChangeSeverity(actor.Role, actor.ID, ticket, severity); err != nil {
		return entity.Ticket{}, err
	}
	customer, err := c.customers.FindByID(ctx, ticket.CustomerID)
	if err != nil {
		return entity.Ticket{}, err
	}
	planScore, err := customer.PlanScore()
	if err != nil {
		return entity.Ticket{}, err
	}
	now := time.Now()
	updated, err := c.tickets.UpdateSeverity(ctx, ticket.ID, severity, planScore, customer.SLAMinutes, reason, actor.ID, now)
	if err != nil {
		return entity.Ticket{}, err
	}
	publish(c.notices, true, updated.CreatedBy, map[string]any{
		"type":           "severity_changed",
		"ticket_id":      updated.ID,
		"severity":       updated.Severity,
		"priority_score": updated.PriorityScore,
	})
	if updated.Status != entity.TicketOpen {
		return updated, nil
	}
	enqueueCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := c.queue.Enqueue(enqueueCtx, updated.ID, updated.PriorityScore); err != nil {
		log.Printf("待ち順の点数を直せませんでした。緊急度は保存してあります。id=%s: %v", updated.ID, err)
	}
	return updated, nil
}
