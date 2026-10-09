package usecase

import (
	"context"
	"time"

	"escalator/entity"
	"escalator/repository"
)

type FoundTicket struct {
	ID               string
	Title            string
	Severity         int
	Status           entity.TicketStatus
	CustomerID       string
	CustomerName     string
	Plan             entity.Plan
	AssigneeID       string
	AssigneeName     string
	PriorityScore    int
	CreatedAt        time.Time
	RemainingMinutes int
	Overdue          bool
	OverdueMinutes   int
}

type SearchTickets struct {
	tickets   repository.TicketRepository
	customers repository.CustomerRepository
	users     repository.UserRepository
	current   *CurrentUser
}

func NewSearchTickets(tickets repository.TicketRepository, customers repository.CustomerRepository, users repository.UserRepository, current *CurrentUser) *SearchTickets {
	return &SearchTickets{tickets: tickets, customers: customers, users: users, current: current}
}

//管理者に、条件に合うチケットを更新が新しい順で返す
func (s *SearchTickets) Execute(ctx context.Context, authorization, status string, severity int, customerID, assigneeID string) ([]FoundTicket, error) {
	actor, err := s.current.Execute(ctx, authorization)
	if err != nil {
		return nil, err
	}
	if actor.Role != entity.RoleAdmin {
		return nil, entity.ErrForbidden
	}
	filter := repository.TicketFilter{CustomerID: customerID, AssigneeID: assigneeID}
	if status != "" {
		parsed, err := entity.ParseTicketStatus(status)
		if err != nil {
			return nil, err
		}
		filter.Status = string(parsed)
	}
	if severity != 0 {
		if severity < 1 || severity > 4 {
			return nil, entity.ErrInvalidSeverity
		}
		filter.Severity = severity
	}
	tickets, err := s.tickets.Search(ctx, filter)
	if err != nil {
		return nil, err
	}
	customers, err := s.customers.List(ctx)
	if err != nil {
		return nil, err
	}
	byCustomer := make(map[string]entity.Customer, len(customers))
	for _, customer := range customers {
		byCustomer[customer.ID] = customer
	}
	names, err := s.users.ListNames(ctx, assigneeIDs(tickets))
	if err != nil {
		return nil, err
	}
	now := time.Now()
	found := make([]FoundTicket, 0, len(tickets))
	for _, ticket := range tickets {
		customer := byCustomer[ticket.CustomerID]
		_, remaining, overdueMinutes, overdue := entity.SLAProgress(ticket.CreatedAt, now, customer.SLAMinutes, ticket.AssigneeID != "")
		if remaining < 0 {
			remaining = 0
		}
		found = append(found, FoundTicket{
			ID:               ticket.ID,
			Title:            ticket.Title,
			Severity:         ticket.Severity,
			Status:           ticket.Status,
			CustomerID:       ticket.CustomerID,
			CustomerName:     customer.Name,
			Plan:             customer.Plan,
			AssigneeID:       ticket.AssigneeID,
			AssigneeName:     names[ticket.AssigneeID],
			PriorityScore:    ticket.PriorityScore,
			CreatedAt:        ticket.CreatedAt,
			RemainingMinutes: remaining,
			Overdue:          overdue,
			OverdueMinutes:   overdueMinutes,
		})
	}
	return found, nil
}
