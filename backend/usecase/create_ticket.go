package usecase

import (
	"context"
	"log"
	"time"

	"escalator/domain"
)

type CreateTicket struct {
	tickets   domain.TicketRepository
	customers domain.CustomerRepository
	current   *CurrentUser
	queue     domain.TicketQueue
}

func NewCreateTicket(tickets domain.TicketRepository, customers domain.CustomerRepository, current *CurrentUser, queue domain.TicketQueue) *CreateTicket {
	return &CreateTicket{tickets: tickets, customers: customers, current: current, queue: queue}
}

//申請者がチケットを起票し、保存したあと待ち順へ載せる
func (c *CreateTicket) Execute(ctx context.Context, authorization, title, description, category string, severity int) (domain.Ticket, error) {
	actor, err := c.current.Execute(ctx, authorization)
	if err != nil {
		return domain.Ticket{}, err
	}
	if actor.Role != domain.RoleApplicant {
		return domain.Ticket{}, domain.ErrTicketApplicant
	}
	if actor.CustomerID == "" {
		return domain.Ticket{}, domain.ErrCustomerRequired
	}
	customer, err := c.customers.FindByID(ctx, actor.CustomerID)
	if err != nil {
		return domain.Ticket{}, err
	}
	planScore, err := customer.PlanScore()
	if err != nil {
		return domain.Ticket{}, err
	}
	now := time.Now()
	ticket, err := domain.NewTicket(actor.CustomerID, actor.ID, title, description, severity, domain.Category(category), now)
	if err != nil {
		return domain.Ticket{}, err
	}
	score, err := domain.PriorityScore(ticket.Severity, planScore, now, now, customer.SLAMinutes, false)
	if err != nil {
		return domain.Ticket{}, err
	}
	ticket.PriorityScore = score
	saved, err := c.tickets.Save(ctx, ticket)
	if err != nil {
		return domain.Ticket{}, err
	}
	enqueueCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := c.queue.Enqueue(enqueueCtx, saved.ID, saved.PriorityScore); err != nil {
		log.Printf("待ち順に載せられませんでした。チケットは保存してあります。id=%s: %v", saved.ID, err)
	}
	return saved, nil
}
