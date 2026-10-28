package usecase

import (
	"context"
	"log"
	"time"

	"escalator/entity"
	"escalator/repository"
)

type ChangeRole struct {
	users     repository.UserRepository
	customers repository.CustomerRepository
	statuses  repository.AgentStatusRepository
	queue     repository.TicketQueue
	current   *CurrentUser
	notices   Notifier
}

func NewChangeRole(users repository.UserRepository, customers repository.CustomerRepository, statuses repository.AgentStatusRepository, queue repository.TicketQueue, current *CurrentUser, notices Notifier) *ChangeRole {
	return &ChangeRole{
		users:     users,
		customers: customers,
		statuses:  statuses,
		queue:     queue,
		current:   current,
		notices:   notices,
	}
}

//管理者だけが利用者の役割を変え、担当者を申請者にするときは対応中を先に待ちへ戻す
func (c *ChangeRole) Execute(ctx context.Context, authorization, userID, role, customerID string) (entity.User, error) {
	actor, err := c.current.Execute(ctx, authorization)
	if err != nil {
		return entity.User{}, err
	}
	if actor.Role != entity.RoleAdmin {
		return entity.User{}, entity.ErrForbidden
	}
	user, err := c.users.FindByID(ctx, userID)
	if err != nil {
		return entity.User{}, err
	}
	if actor.ID == user.ID {
		return entity.User{}, entity.ErrChangeOwnRole
	}
	changed, err := user.ChangeRole(entity.Role(role), customerID)
	if err != nil {
		return entity.User{}, err
	}
	if changed.Role == entity.RoleApplicant {
		if _, err := c.customers.FindByID(ctx, changed.CustomerID); err != nil {
			return entity.User{}, err
		}
	}
	if user.Role == entity.RoleAgent && changed.Role == entity.RoleApplicant {
		if err := c.returnOpenWork(ctx, user); err != nil {
			return entity.User{}, err
		}
	}
	if err := c.users.UpdateRole(ctx, changed); err != nil {
		return entity.User{}, err
	}
	return changed, nil
}

func (c *ChangeRole) returnOpenWork(ctx context.Context, user entity.User) error {
	now := time.Now()
	released, err := c.statuses.GoOffline(ctx, user.ID, now, func(ticket entity.Ticket) (int, error) {
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
		return err
	}
	for _, ticket := range released {
		enqueueCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		err := c.queue.Enqueue(enqueueCtx, ticket.ID, ticket.PriorityScore)
		cancel()
		if err != nil {
			log.Printf("待ち順に載せられませんでした。チケットは待ちに戻してあります。id=%s: %v", ticket.ID, err)
		}
		c.notifyReturned(ctx, ticket)
	}
	return nil
}

func (c *ChangeRole) notifyReturned(ctx context.Context, ticket entity.Ticket) {
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
