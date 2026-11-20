package usecase

import (
	"context"
	"log"
	"time"

	"escalator/entity"
	"escalator/port"
	"escalator/rule"
)

type ChangeSeverity struct {
	tickets   port.TicketRepository
	customers port.CustomerRepository
	current   *CurrentUser
	queue     port.TicketQueue
	notices   Notifier
	audit     *AuditLog
}

func NewChangeSeverity(tickets port.TicketRepository, customers port.CustomerRepository, current *CurrentUser, queue port.TicketQueue, notices Notifier, audit *AuditLog) *ChangeSeverity {
	return &ChangeSeverity{tickets: tickets, customers: customers, current: current, queue: queue, notices: notices, audit: audit}
}

// 緊急度を変え、対応待ちなら待ち順の点数も直す
func (c *ChangeSeverity) Execute(ctx context.Context, authorization, ticketID, reason string, severity int) (entity.Ticket, error) {
	reason, err := rule.SeverityReason(reason)
	if err != nil {
		return entity.Ticket{}, err
	}
	if severity < 1 || severity > 4 {
		return entity.Ticket{}, rule.ErrInvalidSeverity
	}
	actor, err := c.current.Execute(ctx, authorization)
	if err != nil {
		return entity.Ticket{}, err
	}
	if actor.Role != entity.RoleAgent && actor.Role != entity.RoleAdmin {
		return entity.Ticket{}, rule.ErrSeverityRole
	}
	ticket, err := c.tickets.FindByID(ctx, ticketID)
	if err != nil {
		return entity.Ticket{}, err
	}
	if err := rule.CanChangeSeverity(actor.Role, actor.ID, ticket, severity); err != nil {
		return entity.Ticket{}, err
	}
	customer, err := c.customers.FindByID(ctx, ticket.CustomerID)
	if err != nil {
		return entity.Ticket{}, err
	}
	planScore, err := rule.PlanScore(customer)
	if err != nil {
		return entity.Ticket{}, err
	}
	now := time.Now()
	assigned := ticket.Status == entity.TicketInProgress
	score, err := rule.PriorityScore(severity, planScore, ticket.CreatedAt, now, customer.SLAMinutes, assigned)
	if err != nil {
		return entity.Ticket{}, err
	}
	updated, err := c.tickets.UpdateSeverity(ctx, ticket.ID, severity, score, reason, actor.ID, now)
	if err != nil {
		return entity.Ticket{}, err
	}
	c.audit.Record(ctx, actor.ID, entity.AuditSeverityChanged, "ticket", updated.ID, "緊急度を変更しました")
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
