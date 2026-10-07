package usecase

import (
	"context"
	"log"
	"time"

	"escalator/entity"
	"escalator/repository"
)

type ReturnTicketToQueue struct {
	tickets   repository.TicketRepository
	customers repository.CustomerRepository
	queue     repository.TicketQueue
	current   *CurrentUser
}

func NewReturnTicketToQueue(tickets repository.TicketRepository, customers repository.CustomerRepository, queue repository.TicketQueue, current *CurrentUser) *ReturnTicketToQueue {
	return &ReturnTicketToQueue{tickets: tickets, customers: customers, queue: queue, current: current}
}

//管理者が担当を外し、対応中のチケットを待ちに戻す
func (r *ReturnTicketToQueue) Execute(ctx context.Context, authorization, ticketID string) (entity.Ticket, error) {
	actor, err := r.current.Execute(ctx, authorization)
	if err != nil {
		return entity.Ticket{}, err
	}
	if actor.Role != entity.RoleAdmin {
		return entity.Ticket{}, entity.ErrReleaseForbidden
	}
	ticket, err := r.tickets.FindByID(ctx, ticketID)
	if err != nil {
		return entity.Ticket{}, err
	}
	if ticket.Status != entity.TicketInProgress {
		return entity.Ticket{}, entity.ErrNotInProgress
	}
	customer, err := r.customers.FindByID(ctx, ticket.CustomerID)
	if err != nil {
		return entity.Ticket{}, err
	}
	planScore, err := customer.PlanScore()
	if err != nil {
		return entity.Ticket{}, err
	}
	now := time.Now()
	released, err := r.tickets.Release(ctx, ticket.ID, planScore, customer.SLAMinutes, now)
	if err != nil {
		return entity.Ticket{}, err
	}
	enqueueCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := r.queue.Enqueue(enqueueCtx, released.ID, released.PriorityScore); err != nil {
		log.Printf("待ち順に載せられませんでした。チケットは待ちに戻してあります。id=%s: %v", released.ID, err)
	}
	return released, nil
}
