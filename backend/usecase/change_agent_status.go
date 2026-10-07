package usecase

import (
	"context"
	"log"
	"time"

	"escalator/entity"
	"escalator/repository"
)

type AgentStatus struct {
	UserID string
	Status entity.AgentAvailability
}

type ChangeAgentStatus struct {
	users     repository.UserRepository
	customers repository.CustomerRepository
	statuses  repository.AgentStatusRepository
	queue     repository.TicketQueue
	current   *CurrentUser
}

func NewChangeAgentStatus(users repository.UserRepository, customers repository.CustomerRepository, statuses repository.AgentStatusRepository, queue repository.TicketQueue, current *CurrentUser) *ChangeAgentStatus {
	return &ChangeAgentStatus{users: users, customers: customers, statuses: statuses, queue: queue, current: current}
}

//稼働を待機中か離席に切り替える離席なら対応中を待ちに戻す
func (c *ChangeAgentStatus) Execute(ctx context.Context, authorization, targetID, rawStatus string) (AgentStatus, error) {
	next, err := entity.ParseAvailability(rawStatus)
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
	targetRole := actor.Role
	if targetID != actor.ID {
		target, err := c.users.FindByID(ctx, targetID)
		if err != nil {
			return AgentStatus{}, err
		}
		targetRole = target.Role
	}
	if err := entity.CanSetAvailability(actor.Role, actor.ID, targetID, targetRole, next); err != nil {
		return AgentStatus{}, err
	}
	now := time.Now()
	if next == entity.AgentAvailable {
		if err := c.statuses.GoAvailable(ctx, targetID, now); err != nil {
			return AgentStatus{}, err
		}
		return AgentStatus{UserID: targetID, Status: next}, nil
	}
	released, err := c.statuses.GoOffline(ctx, targetID, now, func(ticket entity.Ticket) (int, error) {
		customer, err := c.customers.FindByID(ctx, ticket.CustomerID)
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
		return AgentStatus{}, err
	}
	for _, ticket := range released {
		enqueueCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		err := c.queue.Enqueue(enqueueCtx, ticket.ID, ticket.PriorityScore)
		cancel()
		if err != nil {
			log.Printf("待ち順に載せられませんでした。チケットは待ちに戻してあります。id=%s: %v", ticket.ID, err)
		}
	}
	return AgentStatus{UserID: targetID, Status: next}, nil
}
