package usecase

import (
	"context"
	"errors"
	"time"

	"escalator/entity"
	"escalator/port"
	"escalator/rule"
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
	tickets   port.TicketRepository
	customers port.CustomerRepository
	users     port.UserRepository
	current   *CurrentUser
}

func NewListMyTickets(tickets port.TicketRepository, customers port.CustomerRepository, users port.UserRepository, current *CurrentUser) *ListMyTickets {
	return &ListMyTickets{tickets: tickets, customers: customers, users: users, current: current}
}

//申請者に、自分の起票を更新が新しい順で返す
func (l *ListMyTickets) Execute(ctx context.Context, authorization string) ([]MyTicket, error) {
	actor, err := l.current.Execute(ctx, authorization)
	if err != nil {
		return nil, err
	}
	if actor.Role != entity.RoleApplicant {
		return nil, rule.ErrMyTicketsForbidden
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
		_, remaining, overdueMinutes, overdue := rule.SLAProgress(ticket.CreatedAt, now, customer.SLAMinutes, ticket.AssigneeID != "")
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
	tickets   port.TicketRepository
	customers port.CustomerRepository
	users     port.UserRepository
	current   *CurrentUser
}

func NewShowTicket(tickets port.TicketRepository, customers port.CustomerRepository, users port.UserRepository, current *CurrentUser) *ShowTicket {
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
	if err := rule.CanViewTicket(actor.Role, actor.ID, ticket); err != nil {
		return TicketDetail{}, err
	}
	customer, err := s.customers.FindByID(ctx, ticket.CustomerID)
	if err != nil && !errors.Is(err, rule.ErrCustomerNotFound) {
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
	tickets   port.TicketRepository
	customers port.CustomerRepository
	users     port.UserRepository
	current   *CurrentUser
}

func NewSearchTickets(tickets port.TicketRepository, customers port.CustomerRepository, users port.UserRepository, current *CurrentUser) *SearchTickets {
	return &SearchTickets{tickets: tickets, customers: customers, users: users, current: current}
}

//管理者に、条件に合うチケットを更新が新しい順で返す
func (s *SearchTickets) Execute(ctx context.Context, authorization, status string, severity int, customerID, assigneeID string) ([]FoundTicket, error) {
	actor, err := s.current.Execute(ctx, authorization)
	if err != nil {
		return nil, err
	}
	if actor.Role != entity.RoleAdmin {
		return nil, rule.ErrForbidden
	}
	filter := port.TicketFilter{CustomerID: customerID, AssigneeID: assigneeID}
	if status != "" {
		parsed, err := rule.ParseTicketStatus(status)
		if err != nil {
			return nil, err
		}
		filter.Status = string(parsed)
	}
	if severity != 0 {
		if severity < 1 || severity > 4 {
			return nil, rule.ErrInvalidSeverity
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
		_, remaining, overdueMinutes, overdue := rule.SLAProgress(ticket.CreatedAt, now, customer.SLAMinutes, ticket.AssigneeID != "")
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
