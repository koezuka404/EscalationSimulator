package usecase

import (
	"context"
	"log"
	"time"

	"escalator/entity"
	"escalator/port"
	"escalator/rule"
)

type CreateTicket struct {
	tickets   port.TicketRepository
	customers port.CustomerRepository
	current   *CurrentUser
	queue     port.TicketQueue
	notices   Notifier
	audit     *AuditLog
}

func NewCreateTicket(tickets port.TicketRepository, customers port.CustomerRepository, current *CurrentUser, queue port.TicketQueue, notices Notifier, audit *AuditLog) *CreateTicket {
	return &CreateTicket{tickets: tickets, customers: customers, current: current, queue: queue, notices: notices, audit: audit}
}

// 申請者がチケットを起票し、保存したあと待ち順へ載せる
func (c *CreateTicket) Execute(ctx context.Context, authorization, title, description, category string, severity int) (entity.Ticket, error) {
	actor, err := c.current.Execute(ctx, authorization)
	if err != nil {
		return entity.Ticket{}, err
	}
	if actor.Role != entity.RoleApplicant {
		return entity.Ticket{}, rule.ErrTicketApplicant
	}
	if actor.CustomerID == "" {
		return entity.Ticket{}, rule.ErrCustomerRequired
	}
	customer, err := c.customers.FindByID(ctx, actor.CustomerID)
	if err != nil {
		return entity.Ticket{}, err
	}
	return c.CreateFor(ctx, actor.ID, customer, title, description, severity, entity.Category(category))
}

// 所属顧客の点数でチケットを保存し、待ち順へ載せて知らせる
func (c *CreateTicket) CreateFor(ctx context.Context, createdBy string, customer entity.Customer, title, description string, severity int, category entity.Category) (entity.Ticket, error) {
	planScore, err := rule.PlanScore(customer)
	if err != nil {
		return entity.Ticket{}, err
	}
	now := time.Now()
	ticket, err := rule.NewTicket(customer.ID, createdBy, title, description, severity, category, now)
	if err != nil {
		return entity.Ticket{}, err
	}
	score, err := rule.PriorityScore(ticket.Severity, planScore, now, now, customer.SLAMinutes, false)
	if err != nil {
		return entity.Ticket{}, err
	}
	ticket.PriorityScore = score
	saved, err := c.tickets.Save(ctx, ticket)
	if err != nil {
		return entity.Ticket{}, err
	}
	enqueueCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := c.queue.Enqueue(enqueueCtx, saved.ID, saved.PriorityScore); err != nil {
		log.Printf("待ち順に載せられませんでした。チケットは保存してあります。id=%s: %v", saved.ID, err)
	}
	publish(c.notices, true, saved.CreatedBy, map[string]any{
		"type":   "ticket_created",
		"ticket": ticketView(saved, customer.Name, string(customer.Plan)),
	})
	c.audit.Record(ctx, saved.CreatedBy, entity.AuditTicketCreated, "ticket", saved.ID, "チケットを起票しました")
	return saved, nil
}
