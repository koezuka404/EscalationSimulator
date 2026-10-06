package usecase

import (
	"context"
	"sort"
	"time"

	"escalator/entity"
	"escalator/repository"
)

type WaitingTicket struct {
	Rank             int
	ID               string
	Title            string
	Severity         int
	Plan             entity.Plan
	WaitMinutes      int
	PriorityScore    int
	RemainingMinutes int
	Overdue          bool
	OverdueMinutes   int
	CustomerName     string
}

type ListWaitingTickets struct {
	tickets   repository.TicketRepository
	customers repository.CustomerRepository
	current   *CurrentUser
}

func NewListWaitingTickets(tickets repository.TicketRepository, customers repository.CustomerRepository, current *CurrentUser) *ListWaitingTickets {
	return &ListWaitingTickets{tickets: tickets, customers: customers, current: current}
}

//担当者と管理者に、対応待ちを点数の高い順で返す
func (l *ListWaitingTickets) Execute(ctx context.Context, authorization string) ([]WaitingTicket, error) {
	actor, err := l.current.Execute(ctx, authorization)
	if err != nil {
		return nil, err
	}
	if actor.Role != entity.RoleAgent && actor.Role != entity.RoleAdmin {
		return nil, entity.ErrQueueForbidden
	}
	open, err := l.tickets.ListOpen(ctx)
	if err != nil {
		return nil, err
	}
	customers, err := l.customers.List(ctx)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]entity.Customer, len(customers))
	for _, customer := range customers {
		byID[customer.ID] = customer
	}
	sort.Slice(open, func(i, j int) bool {
		return entity.WaitingFirst(open[i].PriorityScore, open[i].CreatedAt, open[i].ID, open[j].PriorityScore, open[j].CreatedAt, open[j].ID)
	})
	now := time.Now()
	waiting := make([]WaitingTicket, 0, len(open))
	for i, ticket := range open {
		customer := byID[ticket.CustomerID]
		waited, remaining, overdueMinutes, overdue := entity.SLAProgress(ticket.CreatedAt, now, customer.SLAMinutes, ticket.AssigneeID != "")
		waiting = append(waiting, WaitingTicket{
			Rank:             i + 1,
			ID:               ticket.ID,
			Title:            ticket.Title,
			Severity:         ticket.Severity,
			Plan:             customer.Plan,
			WaitMinutes:      waited,
			PriorityScore:    ticket.PriorityScore,
			RemainingMinutes: remaining,
			Overdue:          overdue,
			OverdueMinutes:   overdueMinutes,
			CustomerName:     customer.Name,
		})
	}
	return waiting, nil
}
