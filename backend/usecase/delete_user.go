package usecase

import (
	"context"
	"log"
	"time"

	"escalator/entity"
	"escalator/port"
	"escalator/rule"
)

type DeleteUser struct {
	users     port.UserRepository
	customers port.CustomerRepository
	statuses  port.AgentStatusRepository
	queue     port.TicketQueue
	current   *CurrentUser
	notices   Notifier
	audit     *AuditLog
}

func NewDeleteUser(users port.UserRepository, customers port.CustomerRepository, statuses port.AgentStatusRepository, queue port.TicketQueue, current *CurrentUser, notices Notifier, audit *AuditLog) *DeleteUser {
	return &DeleteUser{users: users, customers: customers, statuses: statuses, queue: queue, current: current, notices: notices, audit: audit}
}

//管理者だけが停止中の利用者を削除する
func (d *DeleteUser) Execute(ctx context.Context, authorization, userID string) (entity.User, error) {
	actor, err := d.current.Execute(ctx, authorization)
	if err != nil {
		return entity.User{}, err
	}
	if actor.Role != entity.RoleAdmin {
		return entity.User{}, rule.ErrForbidden
	}
	user, err := d.users.FindByID(ctx, userID)
	if err != nil {
		return entity.User{}, err
	}
	deleted, err := rule.Delete(user)
	if err != nil {
		return entity.User{}, err
	}
	var released []entity.Ticket
	if user.Role == entity.RoleAgent {
		released, err = d.returnOpenWork(ctx, user)
		if err != nil {
			return entity.User{}, err
		}
	}
	if err := d.users.MarkDeleted(ctx, deleted); err != nil {
		return entity.User{}, err
	}
	for _, ticket := range released {
		d.audit.Record(ctx, actor.ID, entity.AuditTicketReleased, "ticket", ticket.ID, "利用者を削除するため、担当を外して待ちに戻しました")
	}
	return deleted, nil
}

func (d *DeleteUser) returnOpenWork(ctx context.Context, user entity.User) ([]entity.Ticket, error) {
	now := time.Now()
	released, err := d.statuses.GoOffline(ctx, user.ID, now, func(ticket entity.Ticket) (int, error) {
		customer, err := d.customers.FindByID(ctx, ticket.CustomerID)
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
		return nil, err
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
	return released, nil
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
