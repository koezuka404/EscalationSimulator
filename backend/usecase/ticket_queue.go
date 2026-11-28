package usecase

import (
	"context"
	"errors"
	"log"
	"sort"
	"time"

	"escalator/entity"
	"escalator/port"
	"escalator/rule"
)

type ClaimNextTicket struct {
	claims  port.ClaimStore
	queue   port.TicketQueue
	current *CurrentUser
	notices Notifier
	scores  *RecalcOpenScores
	locks   PriorityLock
	every   time.Duration
	audit   *AuditLog
}

func NewClaimNextTicket(claims port.ClaimStore, queue port.TicketQueue, current *CurrentUser, notices Notifier, scores *RecalcOpenScores, locks PriorityLock, every time.Duration, audit *AuditLog) *ClaimNextTicket {
	return &ClaimNextTicket{claims: claims, queue: queue, current: current, notices: notices, scores: scores, locks: locks, every: every, audit: audit}
}

//担当者がいちばん先の対応待ちを引き取る
func (c *ClaimNextTicket) Execute(ctx context.Context, authorization string) (entity.Ticket, error) {
	actor, err := c.current.Execute(ctx, authorization)
	if err != nil {
		return entity.Ticket{}, err
	}
	if actor.Role != entity.RoleAgent {
		return entity.Ticket{}, rule.ErrClaimAgent
	}
	ticket, err := c.claims.ClaimNext(ctx, actor.ID, c.queue, func(createdAt time.Time, slaMinutes int, at time.Time) bool {
		_, _, _, late := rule.SLAProgress(createdAt, at, slaMinutes, false)
		return late
	})
	if errors.Is(err, rule.ErrQueueMismatch) {
		c.rebuildAfterMismatch(ctx)
		return entity.Ticket{}, rule.ErrQueueUnavailable
	}
	if err != nil {
		return entity.Ticket{}, err
	}
	publish(c.notices, true, ticket.CreatedBy, map[string]any{
		"type":          "ticket_claimed",
		"ticket_id":     ticket.ID,
		"title":         ticket.Title,
		"assignee_id":   actor.ID,
		"assignee_name": actor.Name,
	})
	c.audit.Record(ctx, actor.ID, entity.AuditTicketClaimed, "ticket", ticket.ID, "チケットを引き取りました")
	return ticket, nil
}

//3回担当を付けられなかったとき、対応待ちから待ち順を作り直す
func (c *ClaimNextTicket) rebuildAfterMismatch(ctx context.Context) {
	WithPriorityLock(ctx, c.locks, c.every, "待ち順の作り直しは、別のやり直しの途中だったため飛ばしました", func() {
		if err := c.scores.Rebuild(ctx); err != nil {
			log.Printf("待ち順を作り直せませんでした: %v", err)
		}
	})
}

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
	tickets   port.TicketRepository
	customers port.CustomerRepository
	current   *CurrentUser
}

func NewListWaitingTickets(tickets port.TicketRepository, customers port.CustomerRepository, current *CurrentUser) *ListWaitingTickets {
	return &ListWaitingTickets{tickets: tickets, customers: customers, current: current}
}

//担当者と管理者に、対応待ちを点数の高い順で返す
func (l *ListWaitingTickets) Execute(ctx context.Context, authorization string) ([]WaitingTicket, error) {
	actor, err := l.current.Execute(ctx, authorization)
	if err != nil {
		return nil, err
	}
	if actor.Role != entity.RoleAgent && actor.Role != entity.RoleAdmin {
		return nil, rule.ErrQueueForbidden
	}
	open, err := l.tickets.ListOpen(ctx)
	if err != nil {
		return nil, err
	}
	customers, err := l.customers.List(ctx)
	if err != nil {
		return nil, err
	}
	return arrangeWaiting(open, customers, time.Now()), nil
}

func arrangeWaiting(open []entity.Ticket, customers []entity.Customer, now time.Time) []WaitingTicket {
	byID := make(map[string]entity.Customer, len(customers))
	for _, customer := range customers {
		byID[customer.ID] = customer
	}
	sort.Slice(open, func(i, j int) bool {
		return rule.WaitingFirst(open[i].PriorityScore, open[i].CreatedAt, open[i].ID, open[j].PriorityScore, open[j].CreatedAt, open[j].ID)
	})
	waiting := make([]WaitingTicket, 0, len(open))
	for i, ticket := range open {
		customer := byID[ticket.CustomerID]
		waited, remaining, overdueMinutes, overdue := rule.SLAProgress(ticket.CreatedAt, now, customer.SLAMinutes, ticket.AssigneeID != "")
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
	return waiting
}

type SnapshotAgent struct {
	UserID string
	Name   string
	Status string
}

type QueuePicture struct {
	Tickets []WaitingTicket
	Agents  []SnapshotAgent
}

type QueueSnapshot struct {
	tickets   port.TicketRepository
	customers port.CustomerRepository
	users     port.UserRepository
	statuses  port.AgentStatusRepository
}

func NewQueueSnapshot(tickets port.TicketRepository, customers port.CustomerRepository, users port.UserRepository, statuses port.AgentStatusRepository) *QueueSnapshot {
	return &QueueSnapshot{tickets: tickets, customers: customers, users: users, statuses: statuses}
}

//つながった担当者と管理者に、今の待ち順と稼働を返す
func (s *QueueSnapshot) Execute(ctx context.Context) (QueuePicture, error) {
	open, err := s.tickets.ListOpen(ctx)
	if err != nil {
		return QueuePicture{}, err
	}
	customers, err := s.customers.List(ctx)
	if err != nil {
		return QueuePicture{}, err
	}
	statuses, err := s.statuses.List(ctx)
	if err != nil {
		return QueuePicture{}, err
	}
	ids := make([]string, 0, len(statuses))
	for _, status := range statuses {
		ids = append(ids, status.UserID)
	}
	names, err := s.users.ListNames(ctx, ids)
	if err != nil {
		return QueuePicture{}, err
	}
	agents := make([]SnapshotAgent, 0, len(statuses))
	for _, status := range statuses {
		agents = append(agents, SnapshotAgent{
			UserID: status.UserID,
			Name:   names[status.UserID],
			Status: status.Status,
		})
	}
	return QueuePicture{
		Tickets: arrangeWaiting(open, customers, time.Now()),
		Agents:  agents,
	}, nil
}
