package usecase

import (
	"context"
	"errors"
	"time"

	"escalator/entity"
	"escalator/repository"
)

type SeverityChange struct {
	FromSeverity  int
	ToSeverity    int
	Reason        string
	ChangedByName string
	CreatedAt     time.Time
}

type TicketDetail struct {
	Ticket          entity.Ticket
	CustomerName    string
	Plan            entity.Plan
	AssigneeName    string
	SeverityChanges []SeverityChange
}

type ShowTicket struct {
	tickets   repository.TicketRepository
	customers repository.CustomerRepository
	users     repository.UserRepository
	current   *CurrentUser
}

func NewShowTicket(tickets repository.TicketRepository, customers repository.CustomerRepository, users repository.UserRepository, current *CurrentUser) *ShowTicket {
	return &ShowTicket{tickets: tickets, customers: customers, users: users, current: current}
}

//見られる人に、チケットの詳細と緊急度の変更履歴を返す
func (s *ShowTicket) Execute(ctx context.Context, authorization, ticketID string) (TicketDetail, error) {
	actor, err := s.current.Execute(ctx, authorization)
	if err != nil {
		return TicketDetail{}, err
	}
	ticket, err := s.tickets.FindByID(ctx, ticketID)
	if err != nil {
		return TicketDetail{}, err
	}
	if err := entity.CanViewTicket(actor.Role, actor.ID, ticket); err != nil {
		return TicketDetail{}, err
	}
	customer, err := s.customers.FindByID(ctx, ticket.CustomerID)
	if err != nil && !errors.Is(err, entity.ErrCustomerNotFound) {
		return TicketDetail{}, err
	}
	changes, err := s.tickets.ListSeverityChanges(ctx, ticket.ID)
	if err != nil {
		return TicketDetail{}, err
	}
	names, err := s.users.ListNames(ctx, detailNameIDs(ticket.AssigneeID, changes))
	if err != nil {
		return TicketDetail{}, err
	}
	shown := make([]SeverityChange, 0, len(changes))
	for _, change := range changes {
		shown = append(shown, SeverityChange{
			FromSeverity:  change.FromSeverity,
			ToSeverity:    change.ToSeverity,
			Reason:        change.Reason,
			ChangedByName: names[change.ChangedBy],
			CreatedAt:     change.CreatedAt,
		})
	}
	return TicketDetail{
		Ticket:          ticket,
		CustomerName:    customer.Name,
		Plan:            customer.Plan,
		AssigneeName:    names[ticket.AssigneeID],
		SeverityChanges: shown,
	}, nil
}

func detailNameIDs(assigneeID string, changes []entity.SeverityChange) []string {
	seen := make(map[string]struct{}, len(changes)+1)
	ids := make([]string, 0, len(changes)+1)
	if assigneeID != "" {
		seen[assigneeID] = struct{}{}
		ids = append(ids, assigneeID)
	}
	for _, change := range changes {
		if change.ChangedBy == "" {
			continue
		}
		if _, ok := seen[change.ChangedBy]; ok {
			continue
		}
		seen[change.ChangedBy] = struct{}{}
		ids = append(ids, change.ChangedBy)
	}
	return ids
}
