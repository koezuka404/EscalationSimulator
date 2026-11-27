package port

import (
	"context"
	"time"

	"escalator/entity"
)

type CustomerRepository interface {
	Save(ctx context.Context, customer entity.Customer) (entity.Customer, error)
	FindByID(ctx context.Context, id string) (entity.Customer, error)
	List(ctx context.Context) ([]entity.Customer, error)
	Update(ctx context.Context, customer entity.Customer) error
}

type UserRepository interface {
	Save(ctx context.Context, user entity.User) (entity.User, error)
	SaveAgent(ctx context.Context, user entity.User) (entity.User, error)
	FindByID(ctx context.Context, id string) (entity.User, error)
	FindByEmail(ctx context.Context, email string) (entity.User, error)
	UpdateLoginState(ctx context.Context, user entity.User) error
	UpdateCustomer(ctx context.Context, user entity.User) error
	UpdateStatus(ctx context.Context, user entity.User) error
	UpdateRole(ctx context.Context, user entity.User) error
	MarkDeleted(ctx context.Context, user entity.User) error
	BumpAuthVersion(ctx context.Context, id string) error
	ListNames(ctx context.Context, ids []string) (map[string]string, error)
	ListApplicants(ctx context.Context) ([]entity.User, error)
	List(ctx context.Context) ([]entity.User, error)
	Count(ctx context.Context) (int, error)
}

type SessionRepository interface {
	Save(ctx context.Context, session entity.Session) error
	RevokeByHash(ctx context.Context, tokenHash string) error
	FindUserIDByHash(ctx context.Context, tokenHash string) (string, error)
	Rotate(ctx context.Context, oldHash string, next entity.Session, now time.Time) (userID string, reused bool, err error)
}

type TicketRepository interface {
	Save(ctx context.Context, ticket entity.Ticket) (entity.Ticket, error)
	FindByID(ctx context.Context, id string) (entity.Ticket, error)
	ListOpen(ctx context.Context) ([]entity.Ticket, error)
	ListForDashboard(ctx context.Context, since time.Time) ([]entity.Ticket, error)
	ListByCreator(ctx context.Context, createdBy string) ([]entity.Ticket, error)
	Search(ctx context.Context, filter TicketFilter) ([]entity.Ticket, error)
	ListSeverityChanges(ctx context.Context, ticketID string) ([]entity.SeverityChange, error)
	AddWorkNote(ctx context.Context, ticketID, userID, body string, createdAt time.Time) (entity.WorkNote, error)
	ListWorkNotes(ctx context.Context, ticketID string) ([]entity.WorkNote, error)
	Close(ctx context.Context, id, comment string, closedAt time.Time) (entity.Ticket, error)
	UpdateSeverity(ctx context.Context, id string, severity, score int, reason, changedBy string, now time.Time) (entity.Ticket, error)
	UpdateOpenScore(ctx context.Context, id string, score int, now time.Time) (bool, error)
	MarkSLANotified(ctx context.Context, id string, at time.Time) (bool, error)
	Release(ctx context.Context, id string, score int, now time.Time) (entity.Ticket, error)
}

type TicketFilter struct {
	Status     string
	Severity   int
	CustomerID string
	AssigneeID string
}

type TicketQueue interface {
	Enqueue(ctx context.Context, ticketID string, score int) error
	PopMax(ctx context.Context) (ticketID string, score int, ok bool, err error)
	Remove(ctx context.Context, ticketID string) error
	Replace(ctx context.Context, ticketIDs []string, scores []int) error
}

type AgentStatusView struct {
	UserID string
	Status string
}

type AgentStatusRepository interface {
	GoAvailable(ctx context.Context, agentID string, now time.Time) error
	GoOffline(ctx context.Context, agentID string, now time.Time, scoreOf func(entity.Ticket) (int, error)) ([]entity.Ticket, error)
	List(ctx context.Context) ([]AgentStatusView, error)
}

type DemoRepository interface {
	Start(ctx context.Context, run entity.DemoRun) (entity.DemoRun, error)
	AddGenerated(ctx context.Context, id string) (entity.DemoRun, error)
	Finish(ctx context.Context, id, status string, at time.Time) (entity.DemoRun, bool, error)
	StopRunning(ctx context.Context, at time.Time) (entity.DemoRun, error)
	CloseInterrupted(ctx context.Context, at time.Time) error
}

type ClaimStore interface {
	ClaimNext(ctx context.Context, agentID string, queue TicketQueue, overdue func(createdAt time.Time, slaMinutes int, now time.Time) bool) (entity.Ticket, error)
}

type AuditRepository interface {
	Save(ctx context.Context, entry entity.AuditEntry) error
}

type RequestLogRepository interface {
	Save(ctx context.Context, entry entity.RequestLogEntry) error
}

type ConnectionLogRepository interface {
	Save(ctx context.Context, entry entity.ConnectionLogEntry) error
}

type LoginAttemptRepository interface {
	Find(ctx context.Context, email, ip string) (entity.LoginAttempt, error)
	Save(ctx context.Context, attempt entity.LoginAttempt) error
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Matches(hash, password string) bool
}

type TokenIssuer interface {
	Issue(userID string, authVersion int, now time.Time) (string, time.Time, error)
}

type TokenVerifier interface {
	Parse(raw string) (userID string, authVersion int, err error)
}

type SecretTokens interface {
	New() (raw, hash string, err error)
	Hash(raw string) string
}
