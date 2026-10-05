package usecase

import (
	"context"
	"time"

	"escalator/domain"
)

// CreateTicket は申請者がチケットを起票する。顧客とプランは所属から取る。
type CreateTicket struct {
	tickets   domain.TicketRepository
	customers domain.CustomerRepository
	current   *CurrentUser
}

func NewCreateTicket(tickets domain.TicketRepository, customers domain.CustomerRepository, current *CurrentUser) *CreateTicket {
	return &CreateTicket{tickets: tickets, customers: customers, current: current}
}

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
	return c.tickets.Save(ctx, ticket)
}
