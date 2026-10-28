package usecase

import (
	"context"
	"log"
	"time"

	"escalator/entity"
	"escalator/repository"
)

type DeleteUser struct {
	users     repository.UserRepository
	customers repository.CustomerRepository
	statuses  repository.AgentStatusRepository
	queue     repository.TicketQueue
	current   *CurrentUser
	notices   Notifier
}

func NewDeleteUser(users repository.UserRepository, customers repository.CustomerRepository, statuses repository.AgentStatusRepository, queue repository.TicketQueue, current *CurrentUser, notices Notifier) *DeleteUser {
	return &DeleteUser{users: users, customers: customers, statuses: statuses, queue: queue, current: current, notices: notices}
}

//管理者だけが停止中の利用者を削除する
func (d *DeleteUser) Execute(ctx context.Context, authorization, userID string) (entity.User, error) {
	actor, err := d.current.Execute(ctx, authorization)
	if err != nil {
		return entity.User{}, err
	}
	if actor.Role != entity.RoleAdmin {
		return entity.User{}, entity.ErrForbidden
	}
	user, err := d.users.FindByID(ctx, userID)
	if err != nil {
		return entity.User{}, err
	}
	deleted, err := user.Delete()
	if err != nil {
		return entity.User{}, err
	}
	if user.Role == entity.RoleAgent {
		if err := d.returnOpenWork(ctx, user); err != nil {
			return entity.User{}, err
		}
	}
	if err := d.users.MarkDeleted(ctx, deleted); err != nil {
		return entity.User{}, err
	}
	return deleted, nil
}

func (d *DeleteUser) returnOpenWork(ctx context.Context, user entity.User) error {
	now := time.Now()
	released, err := d.statuses.GoOffline(ctx, user.ID, now, func(ticket entity.Ticket) (int, error) {
		customer, err := d.customers.FindByID(ctx, ticket.CustomerID)
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
	for _, ticket := range released {
		enqueueCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		err := d.queue.Enqueue(enqueueCtx, ticket.ID, ticket.PriorityScore)
		cancel()
		if err != nil {
			log.Printf("待ち順に載せられませんでした。チケットは待ちに戻してあります。id=%s: %v", ticket.ID, err)
		}
		d.notifyReturned(ctx, ticket)
	}
	return nil
}

func (d *DeleteUser) notifyReturned(ctx context.Context, ticket entity.Ticket) {
	customerName := ""
	plan := ""
	customer, err := d.customers.FindByID(ctx, ticket.CustomerID)
	if err != nil {
		log.Printf("顧客が見つからないため、待ちに戻した知らせに顧客名を付けられませんでした。id=%s: %v", ticket.ID, err)
	} else {
		customerName = customer.Name
		plan = string(customer.Plan)
	}
	publish(d.notices, true, ticket.CreatedBy, map[string]any{
		"type":   "ticket_returned",
		"ticket": ticketView(ticket, customerName, plan),
	})
}
