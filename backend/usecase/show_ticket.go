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

type WorkNote struct {
	ID         string
	Body       string
	AuthorName string
	CreatedAt  time.Time
}

type TicketDetail struct {
	Ticket          entity.Ticket
	CustomerName    string
	Plan            entity.Plan
	AssigneeName    string
	WorkNotes       []WorkNote
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

//見られる人に、チケットの詳細、対応メモ、緊急度の変更履歴を返す
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
	notes, err := s.tickets.ListWorkNotes(ctx, ticket.ID)
	if err != nil {
		return TicketDetail{}, err
	}
	names, err := s.users.ListNames(ctx, detailNameIDs(ticket.AssigneeID, changes, notes))
	if err != nil {
		return TicketDetail{}, err
	}
	shownNotes := make([]WorkNote, 0, len(notes))
	for _, note := range notes {
		shownNotes = append(shownNotes, WorkNote{
			ID:         note.ID,
			Body:       note.Body,
			AuthorName: names[note.UserID],
			CreatedAt:  note.CreatedAt,
		})
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
		WorkNotes:       shownNotes,
		SeverityChanges: shown,
	}, nil
}

func detailNameIDs(assigneeID string, changes []entity.SeverityChange, notes []entity.WorkNote) []string {
	seen := make(map[string]struct{}, len(changes)+len(notes)+1)
	ids := make([]string, 0, len(changes)+len(notes)+1)
	if assigneeID != "" {
		seen[assigneeID] = struct{}{}
		ids = append(ids, assigneeID)
	}
	for _, note := range notes {
		if note.UserID == "" {
			continue
		}
		if _, ok := seen[note.UserID]; ok {
			continue
		}
		seen[note.UserID] = struct{}{}
		ids = append(ids, note.UserID)
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
