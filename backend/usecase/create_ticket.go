package usecase

import (
	"context"
	"log"
	"time"

	"escalator/entity"
	"escalator/repository"
)

type CreateTicket struct {
	tickets   repository.TicketRepository
	customers repository.CustomerRepository
	current   *CurrentUser
	queue     repository.TicketQueue
	notices   Notifier
}

func NewCreateTicket(tickets repository.TicketRepository, customers repository.CustomerRepository, current *CurrentUser, queue repository.TicketQueue, notices Notifier) *CreateTicket {
	return &CreateTicket{tickets: tickets, customers: customers, current: current, queue: queue, notices: notices}
}

//申請者がチケットを起票し、保存したあと待ち順へ載せる
func (c *CreateTicket) Execute(ctx context.Context, authorization, title, description, category string, severity int) (entity.Ticket, error) {
	actor, err := c.current.Execute(ctx, authorization)
	if err != nil {
		return entity.Ticket{}, err
	}
	if actor.Role != entity.RoleApplicant {
		return entity.Ticket{}, entity.ErrTicketApplicant
	}
	if actor.CustomerID == "" {
		return entity.Ticket{}, entity.ErrCustomerRequired
	}
	customer, err := c.customers.FindByID(ctx, actor.CustomerID)
	if err != nil {
		return entity.Ticket{}, err
	}
	planScore, err := customer.PlanScore()
	if err != nil {
		return entity.Ticket{}, err
	}
	now := time.Now()
	ticket, err := entity.NewTicket(actor.CustomerID, actor.ID, title, description, severity, entity.Category(category), now)
	if err != nil {
		return entity.Ticket{}, err
	}
	score, err := entity.PriorityScore(ticket.Severity, planScore, now, now, customer.SLAMinutes, false)
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
	return saved, nil
}
