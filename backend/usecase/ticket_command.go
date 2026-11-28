package usecase

import (
	"context"
	"log"
	"time"

	"escalator/entity"
	"escalator/port"
	"escalator/rule"
)

type CreateTicket struct {
	tickets   port.TicketRepository
	customers port.CustomerRepository
	current   *CurrentUser
	queue     port.TicketQueue
	notices   Notifier
	audit     *AuditLog
}

func NewCreateTicket(tickets port.TicketRepository, customers port.CustomerRepository, current *CurrentUser, queue port.TicketQueue, notices Notifier, audit *AuditLog) *CreateTicket {
	return &CreateTicket{tickets: tickets, customers: customers, current: current, queue: queue, notices: notices, audit: audit}
}

//申請者がチケットを起票し、保存したあと待ち順へ載せる
func (c *CreateTicket) Execute(ctx context.Context, authorization, title, description, category string, severity int) (entity.Ticket, error) {
	actor, err := c.current.Execute(ctx, authorization)
	if err != nil {
		return entity.Ticket{}, err
	}
	if actor.Role != entity.RoleApplicant {
		return entity.Ticket{}, rule.ErrTicketApplicant
	}
	if actor.CustomerID == "" {
		return entity.Ticket{}, rule.ErrCustomerRequired
	}
	customer, err := c.customers.FindByID(ctx, actor.CustomerID)
	if err != nil {
		return entity.Ticket{}, err
	}
	return c.CreateFor(ctx, actor.ID, customer, title, description, severity, entity.Category(category))
}

//所属顧客の点数でチケットを保存し、待ち順へ載せて知らせる
func (c *CreateTicket) CreateFor(ctx context.Context, createdBy string, customer entity.Customer, title, description string, severity int, category entity.Category) (entity.Ticket, error) {
	planScore, err := rule.PlanScore(customer)
	if err != nil {
		return entity.Ticket{}, err
	}
	now := time.Now()
	ticket, err := rule.NewTicket(customer.ID, createdBy, title, description, severity, category, now)
	if err != nil {
		return entity.Ticket{}, err
	}
	score, err := rule.PriorityScore(ticket.Severity, planScore, now, now, customer.SLAMinutes, false)
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
	c.audit.Record(ctx, saved.CreatedBy, entity.AuditTicketCreated, "ticket", saved.ID, "チケットを起票しました")
	return saved, nil
}

type CloseTicket struct {
	tickets port.TicketRepository
	queue   port.TicketQueue
	current *CurrentUser
	notices Notifier
	audit   *AuditLog
}

func NewCloseTicket(tickets port.TicketRepository, queue port.TicketQueue, current *CurrentUser, notices Notifier, audit *AuditLog) *CloseTicket {
	return &CloseTicket{tickets: tickets, queue: queue, current: current, notices: notices, audit: audit}
}

//対応中のチケットを完了にする
func (c *CloseTicket) Execute(ctx context.Context, authorization, ticketID, comment string) (entity.Ticket, error) {
	comment, err := rule.CloseCommentText(comment)
	if err != nil {
		return entity.Ticket{}, err
	}
	actor, err := c.current.Execute(ctx, authorization)
	if err != nil {
		return entity.Ticket{}, err
	}
	if actor.Role != entity.RoleAgent && actor.Role != entity.RoleAdmin {
		return entity.Ticket{}, rule.ErrCloseForbidden
	}
	ticket, err := c.tickets.FindByID(ctx, ticketID)
	if err != nil {
		return entity.Ticket{}, err
	}
	if actor.Role == entity.RoleAgent && actor.ID != ticket.AssigneeID {
		return entity.Ticket{}, rule.ErrCloseForbidden
	}
	if ticket.Status != entity.TicketInProgress {
		return entity.Ticket{}, rule.ErrNotInProgress
	}
	closed, err := c.tickets.Close(ctx, ticket.ID, comment, time.Now())
	if err != nil {
		return entity.Ticket{}, err
	}
	removeCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := c.queue.Remove(removeCtx, closed.ID); err != nil {
		log.Printf("待ち順から外せませんでした。チケットは完了しています。id=%s: %v", closed.ID, err)
	}
	publish(c.notices, true, closed.CreatedBy, map[string]any{
		"type":      "ticket_closed",
		"ticket_id": closed.ID,
		"title":     closed.Title,
	})
	c.audit.Record(ctx, actor.ID, entity.AuditTicketClosed, "ticket", closed.ID, "チケットを完了しました")
	return closed, nil
}

type ChangeSeverity struct {
	tickets   port.TicketRepository
	customers port.CustomerRepository
	current   *CurrentUser
	queue     port.TicketQueue
	notices   Notifier
	audit     *AuditLog
}

func NewChangeSeverity(tickets port.TicketRepository, customers port.CustomerRepository, current *CurrentUser, queue port.TicketQueue, notices Notifier, audit *AuditLog) *ChangeSeverity {
	return &ChangeSeverity{tickets: tickets, customers: customers, current: current, queue: queue, notices: notices, audit: audit}
}

//緊急度を変え、対応待ちなら待ち順の点数も直す
func (c *ChangeSeverity) Execute(ctx context.Context, authorization, ticketID, reason string, severity int) (entity.Ticket, error) {
	reason, err := rule.SeverityReason(reason)
	if err != nil {
		return entity.Ticket{}, err
	}
	if severity < 1 || severity > 4 {
		return entity.Ticket{}, rule.ErrInvalidSeverity
	}
	actor, err := c.current.Execute(ctx, authorization)
	if err != nil {
		return entity.Ticket{}, err
	}
	if actor.Role != entity.RoleAgent && actor.Role != entity.RoleAdmin {
		return entity.Ticket{}, rule.ErrSeverityRole
	}
	ticket, err := c.tickets.FindByID(ctx, ticketID)
	if err != nil {
		return entity.Ticket{}, err
	}
	if err := rule.CanChangeSeverity(actor.Role, actor.ID, ticket, severity); err != nil {
		return entity.Ticket{}, err
	}
	customer, err := c.customers.FindByID(ctx, ticket.CustomerID)
	if err != nil {
		return entity.Ticket{}, err
	}
	planScore, err := rule.PlanScore(customer)
	if err != nil {
		return entity.Ticket{}, err
	}
	now := time.Now()
	assigned := ticket.Status == entity.TicketInProgress
	score, err := rule.PriorityScore(severity, planScore, ticket.CreatedAt, now, customer.SLAMinutes, assigned)
	if err != nil {
		return entity.Ticket{}, err
	}
	updated, err := c.tickets.UpdateSeverity(ctx, ticket.ID, severity, score, reason, actor.ID, now)
	if err != nil {
		return entity.Ticket{}, err
	}
	c.audit.Record(ctx, actor.ID, entity.AuditSeverityChanged, "ticket", updated.ID, "緊急度を変更しました")
	publish(c.notices, true, updated.CreatedBy, map[string]any{
		"type":           "severity_changed",
		"ticket_id":      updated.ID,
		"severity":       updated.Severity,
		"priority_score": updated.PriorityScore,
	})
	if updated.Status != entity.TicketOpen {
		return updated, nil
	}
	enqueueCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := c.queue.Enqueue(enqueueCtx, updated.ID, updated.PriorityScore); err != nil {
		log.Printf("待ち順の点数を直せませんでした。緊急度は保存してあります。id=%s: %v", updated.ID, err)
	}
	return updated, nil
}

type ReturnTicketToQueue struct {
	tickets   port.TicketRepository
	customers port.CustomerRepository
	queue     port.TicketQueue
	current   *CurrentUser
	notices   Notifier
	audit     *AuditLog
}

func NewReturnTicketToQueue(tickets port.TicketRepository, customers port.CustomerRepository, queue port.TicketQueue, current *CurrentUser, notices Notifier, audit *AuditLog) *ReturnTicketToQueue {
	return &ReturnTicketToQueue{tickets: tickets, customers: customers, queue: queue, current: current, notices: notices, audit: audit}
}

//管理者が担当を外し、対応中のチケットを待ちに戻す
func (r *ReturnTicketToQueue) Execute(ctx context.Context, authorization, ticketID string) (entity.Ticket, error) {
	actor, err := r.current.Execute(ctx, authorization)
	if err != nil {
		return entity.Ticket{}, err
	}
	if actor.Role != entity.RoleAdmin {
		return entity.Ticket{}, rule.ErrReleaseForbidden
	}
	ticket, err := r.tickets.FindByID(ctx, ticketID)
	if err != nil {
		return entity.Ticket{}, err
	}
	if ticket.Status != entity.TicketInProgress {
		return entity.Ticket{}, rule.ErrNotInProgress
	}
	customer, err := r.customers.FindByID(ctx, ticket.CustomerID)
	if err != nil {
		return entity.Ticket{}, err
	}
	planScore, err := rule.PlanScore(customer)
	if err != nil {
		return entity.Ticket{}, err
	}
	now := time.Now()
	score, err := rule.PriorityScore(ticket.Severity, planScore, ticket.CreatedAt, now, customer.SLAMinutes, false)
	if err != nil {
		return entity.Ticket{}, err
	}
	released, err := r.tickets.Release(ctx, ticket.ID, score, now)
	if err != nil {
		return entity.Ticket{}, err
	}
	enqueueCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := r.queue.Enqueue(enqueueCtx, released.ID, released.PriorityScore); err != nil {
		log.Printf("待ち順に載せられませんでした。チケットは待ちに戻してあります。id=%s: %v", released.ID, err)
	}
	publish(r.notices, true, released.CreatedBy, map[string]any{
		"type":   "ticket_returned",
		"ticket": ticketView(released, customer.Name, string(customer.Plan)),
	})
	r.audit.Record(ctx, actor.ID, entity.AuditTicketReleased, "ticket", released.ID, "担当を外して待ちに戻しました")
	return released, nil
}

type SavedWorkNote struct {
	ID         string
	TicketID   string
	Body       string
	AuthorName string
	CreatedAt  time.Time
}

type AddWorkNote struct {
	tickets port.TicketRepository
	current *CurrentUser
}

func NewAddWorkNote(tickets port.TicketRepository, current *CurrentUser) *AddWorkNote {
	return &AddWorkNote{tickets: tickets, current: current}
}

//対応メモを保存する
func (a *AddWorkNote) Execute(ctx context.Context, authorization, ticketID, body string) (SavedWorkNote, error) {
	body, err := rule.WorkNoteText(body)
	if err != nil {
		return SavedWorkNote{}, err
	}
	actor, err := a.current.Execute(ctx, authorization)
	if err != nil {
		return SavedWorkNote{}, err
	}
	if actor.Role != entity.RoleAgent && actor.Role != entity.RoleAdmin {
		return SavedWorkNote{}, rule.ErrWorkNoteRole
	}
	ticket, err := a.tickets.FindByID(ctx, ticketID)
	if err != nil {
		return SavedWorkNote{}, err
	}
	if err := rule.CanAddWorkNote(actor.Role, actor.ID, ticket); err != nil {
		return SavedWorkNote{}, err
	}
	saved, err := a.tickets.AddWorkNote(ctx, ticket.ID, actor.ID, body, time.Now())
	if err != nil {
		return SavedWorkNote{}, err
	}
	return SavedWorkNote{
		ID:         saved.ID,
		TicketID:   saved.TicketID,
		Body:       saved.Body,
		AuthorName: actor.Name,
		CreatedAt:  saved.CreatedAt,
	}, nil
}
