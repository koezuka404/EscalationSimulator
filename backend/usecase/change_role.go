package usecase

import (
	"context"
	"log"
	"time"

	"escalator/entity"
	"escalator/port"
	"escalator/rule"
)

type ChangeRole struct {
	users     port.UserRepository
	customers port.CustomerRepository
	statuses  port.AgentStatusRepository
	queue     port.TicketQueue
	current   *CurrentUser
	notices   Notifier
	audit     *AuditLog
}

func NewChangeRole(users port.UserRepository, customers port.CustomerRepository, statuses port.AgentStatusRepository, queue port.TicketQueue, current *CurrentUser, notices Notifier, audit *AuditLog) *ChangeRole {
	return &ChangeRole{
		users:     users,
		customers: customers,
		statuses:  statuses,
		queue:     queue,
		current:   current,
		notices:   notices,
		audit:     audit,
	}
}

// 管理者だけが利用者の役割を変え、担当者を申請者にするときは対応中を先に待ちへ戻す
func (c *ChangeRole) Execute(ctx context.Context, authorization, userID, role, customerID string) (entity.User, error) {
	actor, err := c.current.Execute(ctx, authorization)
	if err != nil {
		return entity.User{}, err
	}
	if actor.Role != entity.RoleAdmin {
		return entity.User{}, rule.ErrForbidden
	}
	user, err := c.users.FindByID(ctx, userID)
	if err != nil {
		return entity.User{}, err
	}
	if actor.ID == user.ID {
		return entity.User{}, rule.ErrChangeOwnRole
	}
	changed, err := rule.ChangeRole(user, entity.Role(role), customerID)
	if err != nil {
		return entity.User{}, err
	}
	if changed.Role == entity.RoleApplicant {
		if _, err := c.customers.FindByID(ctx, changed.CustomerID); err != nil {
			return entity.User{}, err
		}
	}
	var released []entity.Ticket
	if user.Role == entity.RoleAgent && changed.Role == entity.RoleApplicant {
		released, err = c.returnOpenWork(ctx, user)
		if err != nil {
			return entity.User{}, err
		}
	}
	if err := c.users.UpdateRole(ctx, changed); err != nil {
		return entity.User{}, err
	}
	c.audit.Record(ctx, actor.ID, entity.AuditRoleChanged, "user", changed.ID, "役割を"+roleLabel(user.Role)+"から"+roleLabel(changed.Role)+"にしました")
	for _, ticket := range released {
		c.audit.Record(ctx, actor.ID, entity.AuditTicketReleased, "ticket", ticket.ID, "役割を変えるため、担当を外して待ちに戻しました")
	}
	return changed, nil
}

func roleLabel(role entity.Role) string {
	switch role {
	case entity.RoleApplicant:
		return "申請者"
	case entity.RoleAgent:
		return "担当者"
	case entity.RoleAdmin:
		return "管理者"
	default:
		return string(role)
	}
}

func (c *ChangeRole) returnOpenWork(ctx context.Context, user entity.User) ([]entity.Ticket, error) {
	now := time.Now()
	released, err := c.statuses.GoOffline(ctx, user.ID, now, func(ticket entity.Ticket) (int, error) {
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
		return nil, err
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
	return released, nil
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
