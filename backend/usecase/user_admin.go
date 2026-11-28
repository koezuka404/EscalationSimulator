package usecase

import (
	"context"
	"log"
	"time"

	"escalator/entity"
	"escalator/port"
	"escalator/rule"
)

type CreateAgent struct {
	users    port.UserRepository
	current  *CurrentUser
	password port.PasswordHasher
}

func NewCreateAgent(users port.UserRepository, current *CurrentUser, password port.PasswordHasher) *CreateAgent {
	return &CreateAgent{users: users, current: current, password: password}
}

//管理者が担当者を登録する
func (c *CreateAgent) Execute(ctx context.Context, authorization, name, email, password string) (entity.User, error) {
	actor, err := c.current.Execute(ctx, authorization)
	if err != nil {
		return entity.User{}, err
	}
	if actor.Role != entity.RoleAdmin {
		return entity.User{}, rule.ErrForbidden
	}
	user, err := rule.NewAgent(name, email, password)
	if err != nil {
		return entity.User{}, err
	}
	hash, err := c.password.Hash(password)
	if err != nil {
		return entity.User{}, err
	}
	user.PasswordHash = hash
	return c.users.SaveAgent(ctx, user)
}

type ListUsers struct {
	users   port.UserRepository
	current *CurrentUser
}

func NewListUsers(users port.UserRepository, current *CurrentUser) *ListUsers {
	return &ListUsers{users: users, current: current}
}

//管理者が利用者の一覧を取る
func (l *ListUsers) Execute(ctx context.Context, authorization string) ([]entity.User, error) {
	actor, err := l.current.Execute(ctx, authorization)
	if err != nil {
		return nil, err
	}
	if actor.Role != entity.RoleAdmin {
		return nil, rule.ErrForbidden
	}
	return l.users.List(ctx)
}

type SuspendUser struct {
	users   port.UserRepository
	current *CurrentUser
	screens ScreenConnection
	audit   *AuditLog
}

func NewSuspendUser(users port.UserRepository, current *CurrentUser, screens ScreenConnection, audit *AuditLog) *SuspendUser {
	return &SuspendUser{users: users, current: current, screens: screens, audit: audit}
}

//管理者だけが利用者を停止し、その人の画面の接続を切る
func (s *SuspendUser) Execute(ctx context.Context, authorization, userID string) (entity.User, error) {
	actor, err := s.current.Execute(ctx, authorization)
	if err != nil {
		return entity.User{}, err
	}
	if actor.Role != entity.RoleAdmin {
		return entity.User{}, rule.ErrForbidden
	}
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return entity.User{}, err
	}
	if actor.ID == user.ID {
		return entity.User{}, rule.ErrSuspendSelf
	}
	suspended, err := rule.Suspend(user)
	if err != nil {
		return entity.User{}, err
	}
	if err := s.users.UpdateStatus(ctx, suspended); err != nil {
		return entity.User{}, err
	}
	if s.screens != nil {
		s.screens.CloseUser(suspended.ID)
		log.Printf("利用者を停止したため、その人の画面の接続を切りました。user_id=%s", suspended.ID)
		s.audit.Record(ctx, actor.ID, entity.AuditConnectionClosed, "user", suspended.ID, "画面の接続を切りました")
	}
	return suspended, nil
}

type ResumeUser struct {
	users   port.UserRepository
	current *CurrentUser
}

func NewResumeUser(users port.UserRepository, current *CurrentUser) *ResumeUser {
	return &ResumeUser{users: users, current: current}
}

//管理者だけが、停止中の利用者を利用中に戻す
func (r *ResumeUser) Execute(ctx context.Context, authorization, userID string) (entity.User, error) {
	actor, err := r.current.Execute(ctx, authorization)
	if err != nil {
		return entity.User{}, err
	}
	if actor.Role != entity.RoleAdmin {
		return entity.User{}, rule.ErrForbidden
	}
	user, err := r.users.FindByID(ctx, userID)
	if err != nil {
		return entity.User{}, err
	}
	resumed, err := rule.Resume(user)
	if err != nil {
		return entity.User{}, err
	}
	if err := r.users.UpdateStatus(ctx, resumed); err != nil {
		return entity.User{}, err
	}
	return resumed, nil
}

type LinkApplicant struct {
	users     port.UserRepository
	customers port.CustomerRepository
	current   *CurrentUser
	audit     *AuditLog
}

func NewLinkApplicant(users port.UserRepository, customers port.CustomerRepository, current *CurrentUser, audit *AuditLog) *LinkApplicant {
	return &LinkApplicant{users: users, customers: customers, current: current, audit: audit}
}

//管理者だけが、申請者を顧客に結びつける
func (l *LinkApplicant) Execute(ctx context.Context, authorization, userID, customerID string) (entity.User, error) {
	actor, err := l.current.Execute(ctx, authorization)
	if err != nil {
		return entity.User{}, err
	}
	if actor.Role != entity.RoleAdmin {
		return entity.User{}, rule.ErrForbidden
	}
	user, err := l.users.FindByID(ctx, userID)
	if err != nil {
		return entity.User{}, err
	}
	linked, err := rule.LinkCustomer(user, customerID)
	if err != nil {
		return entity.User{}, err
	}
	if _, err := l.customers.FindByID(ctx, customerID); err != nil {
		return entity.User{}, err
	}
	if linked.AuthVersion == user.AuthVersion {
		return user, nil
	}
	if err := l.users.UpdateCustomer(ctx, linked); err != nil {
		return entity.User{}, err
	}
	l.audit.Record(ctx, actor.ID, entity.AuditCustomerChanged, "user", linked.ID, "申請者の所属顧客を変更しました")
	return linked, nil
}

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

//管理者だけが利用者の役割を変え、担当者を申請者にするときは対応中を先に待ちへ戻す
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
