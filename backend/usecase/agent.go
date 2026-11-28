package usecase

import (
	"context"
	"errors"
	"log"
	"time"

	"escalator/entity"
	"escalator/port"
	"escalator/rule"
)

type AgentStatus struct {
	UserID string
	Status entity.AgentAvailability
}

type ChangeAgentStatus struct {
	users     port.UserRepository
	customers port.CustomerRepository
	statuses  port.AgentStatusRepository
	queue     port.TicketQueue
	current   *CurrentUser
	notices   Notifier
	audit     *AuditLog
}

func NewChangeAgentStatus(users port.UserRepository, customers port.CustomerRepository, statuses port.AgentStatusRepository, queue port.TicketQueue, current *CurrentUser, notices Notifier, audit *AuditLog) *ChangeAgentStatus {
	return &ChangeAgentStatus{users: users, customers: customers, statuses: statuses, queue: queue, current: current, notices: notices, audit: audit}
}

//稼働を待機中か離席に切り替える離席なら対応中を待ちに戻す
func (c *ChangeAgentStatus) Execute(ctx context.Context, authorization, targetID, rawStatus string) (AgentStatus, error) {
	next, err := rule.ParseAvailability(rawStatus)
	if err != nil {
		return AgentStatus{}, err
	}
	actor, err := c.current.Execute(ctx, authorization)
	if err != nil {
		return AgentStatus{}, err
	}
	if targetID == "" {
		targetID = actor.ID
	}
	targetName := actor.Name
	targetRole := actor.Role
	if targetID != actor.ID {
		target, err := c.users.FindByID(ctx, targetID)
		if err != nil {
			return AgentStatus{}, err
		}
		targetRole = target.Role
		targetName = target.Name
	}
	if err := rule.CanSetAvailability(actor.Role, actor.ID, targetID, targetRole, next); err != nil {
		return AgentStatus{}, err
	}
	now := time.Now()
	if next == entity.AgentAvailable {
		if err := c.statuses.GoAvailable(ctx, targetID, now); err != nil {
			return AgentStatus{}, err
		}
		publish(c.notices, true, "", map[string]any{
			"type":    "agent_status_changed",
			"user_id": targetID,
			"name":    targetName,
			"status":  next,
		})
		return AgentStatus{UserID: targetID, Status: next}, nil
	}
	released, err := c.statuses.GoOffline(ctx, targetID, now, func(ticket entity.Ticket) (int, error) {
		customer, err := c.customers.FindByID(ctx, ticket.CustomerID)
		if err != nil {
			return 0, err
		}
		planScore, err := rule.PlanScore(customer)
		if err != nil {
			return 0, err
		}
		return rule.PriorityScore(ticket.Severity, planScore, ticket.CreatedAt, now, customer.SLAMinutes, false)
	})
	if err != nil {
		return AgentStatus{}, err
	}
	for _, ticket := range released {
		enqueueCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		err := c.queue.Enqueue(enqueueCtx, ticket.ID, ticket.PriorityScore)
		cancel()
		if err != nil {
			log.Printf("待ち順に載せられませんでした。チケットは待ちに戻してあります。id=%s: %v", ticket.ID, err)
		}
		c.notifyReturned(ctx, ticket)
		c.audit.Record(ctx, actor.ID, entity.AuditTicketReleased, "ticket", ticket.ID, "離席にしたため、担当を外して待ちに戻しました")
	}
	publish(c.notices, true, "", map[string]any{
		"type":    "agent_status_changed",
		"user_id": targetID,
		"name":    targetName,
		"status":  next,
	})
	return AgentStatus{UserID: targetID, Status: next}, nil
}

func (c *ChangeAgentStatus) notifyReturned(ctx context.Context, ticket entity.Ticket) {
	customerName := ""
	plan := ""
	customer, err := c.customers.FindByID(ctx, ticket.CustomerID)
	if err != nil {
		log.Printf("顧客が見つからないため、待ちに戻した知らせに顧客名を付けられませんでした。id=%s: %v", ticket.ID, err)
	} else {
		customerName = customer.Name
		plan = string(customer.Plan)
	}
	publish(c.notices, true, ticket.CreatedBy, map[string]any{
		"type":   "ticket_returned",
		"ticket": ticketView(ticket, customerName, plan),
	})
}

type AgentPresence interface {
	AgentsGoneSince(before time.Time) []string
	StillGone(userID string) bool
	ForgetGone(userID string)
}

type ReleaseDisconnectedAgents struct {
	users     port.UserRepository
	customers port.CustomerRepository
	statuses  port.AgentStatusRepository
	queue     port.TicketQueue
	presence  AgentPresence
	notices   Notifier
	grace     time.Duration
	audit     *AuditLog
}

func NewReleaseDisconnectedAgents(users port.UserRepository, customers port.CustomerRepository, statuses port.AgentStatusRepository, queue port.TicketQueue, presence AgentPresence, notices Notifier, grace time.Duration, audit *AuditLog) *ReleaseDisconnectedAgents {
	return &ReleaseDisconnectedAgents{
		users:     users,
		customers: customers,
		statuses:  statuses,
		queue:     queue,
		presence:  presence,
		notices:   notices,
		grace:     grace,
		audit:     audit,
	}
}

//接続がすべて切れてから一定時間戻らない担当者を離席にし、対応中を待ちに戻す
func (r *ReleaseDisconnectedAgents) Execute(ctx context.Context) error {
	due := r.presence.AgentsGoneSince(time.Now().Add(-r.grace))
	for _, id := range due {
		if err := r.releaseOne(ctx, id); err != nil {
			log.Printf("接続が切れた担当者を離席にできませんでした。次の確認でもう一度試します。id=%s: %v", id, err)
		}
	}
	return nil
}

func (r *ReleaseDisconnectedAgents) releaseOne(ctx context.Context, id string) error {
	if !r.presence.StillGone(id) {
		return nil
	}
	user, err := r.users.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, rule.ErrUserNotFound) {
			r.presence.ForgetGone(id)
			log.Printf("接続が切れた担当者が見つからなかったため、離席の確認をやめました。id=%s", id)
			return nil
		}
		return err
	}
	if user.Status == entity.StatusDeleted || user.Role != entity.RoleAgent {
		r.presence.ForgetGone(id)
		return nil
	}
	if !r.presence.StillGone(id) {
		return nil
	}
	now := time.Now()
	released, err := r.statuses.GoOffline(ctx, id, now, func(ticket entity.Ticket) (int, error) {
		customer, err := r.customers.FindByID(ctx, ticket.CustomerID)
		if err != nil {
			return 0, err
		}
		planScore, err := rule.PlanScore(customer)
		if err != nil {
			return 0, err
		}
		return rule.PriorityScore(ticket.Severity, planScore, ticket.CreatedAt, now, customer.SLAMinutes, false)
	})
	if err != nil {
		return err
	}
	r.presence.ForgetGone(id)
	for _, ticket := range released {
		enqueueCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		err := r.queue.Enqueue(enqueueCtx, ticket.ID, ticket.PriorityScore)
		cancel()
		if err != nil {
			log.Printf("待ち順に載せられませんでした。チケットは待ちに戻してあります。id=%s: %v", ticket.ID, err)
		}
		r.notifyReturned(ctx, ticket)
		r.audit.Record(ctx, id, entity.AuditTicketReleased, "ticket", ticket.ID, "接続が切れたまま戻らなかったため、担当を外して待ちに戻しました")
	}
	publish(r.notices, true, "", map[string]any{
		"type":    "agent_status_changed",
		"user_id": id,
		"name":    user.Name,
		"status":  entity.AgentOffline,
	})
	return nil
}

func (r *ReleaseDisconnectedAgents) notifyReturned(ctx context.Context, ticket entity.Ticket) {
	customerName := ""
	plan := ""
	customer, err := r.customers.FindByID(ctx, ticket.CustomerID)
	if err != nil {
		log.Printf("顧客が見つからないため、待ちに戻した知らせに顧客名を付けられませんでした。id=%s: %v", ticket.ID, err)
	} else {
		customerName = customer.Name
		plan = string(customer.Plan)
	}
	publish(r.notices, true, ticket.CreatedBy, map[string]any{
		"type":   "ticket_returned",
		"ticket": ticketView(ticket, customerName, plan),
	})
}
