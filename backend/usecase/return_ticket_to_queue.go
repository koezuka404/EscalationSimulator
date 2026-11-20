package usecase

import (
	"context"
	"log"
	"time"

	"escalator/entity"
	"escalator/port"
	"escalator/rule"
)

type ReturnTicketToQueue struct {
	tickets   port.TicketRepository
	customers port.CustomerRepository
	queue     port.TicketQueue
	current   *CurrentUser
	notices   Notifier
	audit     *AuditLog
}

func NewReturnTicketToQueue(tickets port.TicketRepository, customers port.CustomerRepository, queue port.TicketQueue, current *CurrentUser, notices Notifier, audit *AuditLog) *ReturnTicketToQueue {
	return &ReturnTicketToQueue{tickets: tickets, customers: customers, queue: queue, current: current, notices: notices, audit: audit}
}

// 管理者が担当を外し、対応中のチケットを待ちに戻す
func (r *ReturnTicketToQueue) Execute(ctx context.Context, authorization, ticketID string) (entity.Ticket, error) {
	actor, err := r.current.Execute(ctx, authorization)
	if err != nil {
		return entity.Ticket{}, err
	}
	if actor.Role != entity.RoleAdmin {
		return entity.Ticket{}, rule.ErrReleaseForbidden
	}
	ticket, err := r.tickets.FindByID(ctx, ticketID)
	if err != nil {
		return entity.Ticket{}, err
	}
	if ticket.Status != entity.TicketInProgress {
		return entity.Ticket{}, rule.ErrNotInProgress
	}
	customer, err := r.customers.FindByID(ctx, ticket.CustomerID)
	if err != nil {
		return entity.Ticket{}, err
	}
	planScore, err := rule.PlanScore(customer)
	if err != nil {
		return entity.Ticket{}, err
	}
	now := time.Now()
	score, err := rule.PriorityScore(ticket.Severity, planScore, ticket.CreatedAt, now, customer.SLAMinutes, false)
	if err != nil {
		return entity.Ticket{}, err
	}
	released, err := r.tickets.Release(ctx, ticket.ID, score, now)
	if err != nil {
		return entity.Ticket{}, err
	}
	enqueueCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := r.queue.Enqueue(enqueueCtx, released.ID, released.PriorityScore); err != nil {
		log.Printf("待ち順に載せられませんでした。チケットは待ちに戻してあります。id=%s: %v", released.ID, err)
	}
	publish(r.notices, true, released.CreatedBy, map[string]any{
		"type":   "ticket_returned",
		"ticket": ticketView(released, customer.Name, string(customer.Plan)),
	})
	r.audit.Record(ctx, actor.ID, entity.AuditTicketReleased, "ticket", released.ID, "担当を外して待ちに戻しました")
	return released, nil
}
