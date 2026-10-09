package usecase

import (
	"context"
	"errors"
	"log"
	"time"

	"escalator/entity"
	"escalator/repository"
)

type AgentPresence interface {
	AgentsGoneSince(before time.Time) []string
	StillGone(userID string) bool
	ForgetGone(userID string)
}

type ReleaseDisconnectedAgents struct {
	users     repository.UserRepository
	customers repository.CustomerRepository
	statuses  repository.AgentStatusRepository
	queue     repository.TicketQueue
	presence  AgentPresence
	notices   Notifier
	grace     time.Duration
}

func NewReleaseDisconnectedAgents(users repository.UserRepository, customers repository.CustomerRepository, statuses repository.AgentStatusRepository, queue repository.TicketQueue, presence AgentPresence, notices Notifier, grace time.Duration) *ReleaseDisconnectedAgents {
	return &ReleaseDisconnectedAgents{
		users:     users,
		customers: customers,
		statuses:  statuses,
		queue:     queue,
		presence:  presence,
		notices:   notices,
		grace:     grace,
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
		if errors.Is(err, entity.ErrUserNotFound) {
			r.presence.ForgetGone(id)
			log.Printf("接続が切れた担当者が見つからなかったため、離席の確認をやめました。id=%s", id)
			return nil
		}
		return err
	}
	if user.Role != entity.RoleAgent {
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
		planScore, err := customer.PlanScore()
		if err != nil {
			return 0, err
		}
		return entity.PriorityScore(ticket.Severity, planScore, ticket.CreatedAt, now, customer.SLAMinutes, false)
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
