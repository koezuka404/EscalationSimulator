package usecase

import (
	"context"
	"time"

	"escalator/entity"
	"escalator/repository"
)

type MyTicket struct {
	ID               string
	Title            string
	Severity         int
	Status           entity.TicketStatus
	CreatedAt        time.Time
	RemainingMinutes int
	Overdue          bool
	OverdueMinutes   int
	AssigneeName     string
}

type ListMyTickets struct {
	tickets   repository.TicketRepository
	customers repository.CustomerRepository
	users     repository.UserRepository
	current   *CurrentUser
}

func NewListMyTickets(tickets repository.TicketRepository, customers repository.CustomerRepository, users repository.UserRepository, current *CurrentUser) *ListMyTickets {
	return &ListMyTickets{tickets: tickets, customers: customers, users: users, current: current}
}

//申請者に、自分の起票を更新が新しい順で返す
func (l *ListMyTickets) Execute(ctx context.Context, authorization string) ([]MyTicket, error) {
	actor, err := l.current.Execute(ctx, authorization)
	if err != nil {
		return nil, err
	}
	if actor.Role != entity.RoleApplicant {
		return nil, entity.ErrMyTicketsForbidden
	}
	tickets, err := l.tickets.ListByCreator(ctx, actor.ID)
	if err != nil {
		return nil, err
	}
	customers, err := l.customers.List(ctx)
	if err != nil {
		return nil, err
	}
	byCustomer := make(map[string]entity.Customer, len(customers))
	for _, customer := range customers {
		byCustomer[customer.ID] = customer
	}
	names, err := l.users.ListNames(ctx, assigneeIDs(tickets))
	if err != nil {
		return nil, err
	}
	now := time.Now()
	mine := make([]MyTicket, 0, len(tickets))
	for _, ticket := range tickets {
		customer := byCustomer[ticket.CustomerID]
		_, remaining, overdueMinutes, overdue := entity.SLAProgress(ticket.CreatedAt, now, customer.SLAMinutes, ticket.AssigneeID != "")
		if remaining < 0 {
			remaining = 0
		}
		mine = append(mine, MyTicket{
			ID:               ticket.ID,
			Title:            ticket.Title,
			Severity:         ticket.Severity,
			Status:           ticket.Status,
			CreatedAt:        ticket.CreatedAt,
			RemainingMinutes: remaining,
			Overdue:          overdue,
			OverdueMinutes:   overdueMinutes,
			AssigneeName:     names[ticket.AssigneeID],
		})
	}
	return mine, nil
}

func assigneeIDs(tickets []entity.Ticket) []string {
	seen := make(map[string]struct{}, len(tickets))
	ids := make([]string, 0, len(tickets))
	for _, ticket := range tickets {
		if ticket.AssigneeID == "" {
			continue
		}
		if _, ok := seen[ticket.AssigneeID]; ok {
			continue
		}
		seen[ticket.AssigneeID] = struct{}{}
		ids = append(ids, ticket.AssigneeID)
	}
	return ids
}
