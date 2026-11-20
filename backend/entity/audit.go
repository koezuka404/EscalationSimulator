package entity

import "time"

const (
	AuditLogin            = "login"
	AuditLogout           = "logout"
	AuditRefreshReused    = "refresh_reused"
	AuditLoginLocked      = "login_locked"
	AuditTicketCreated    = "ticket_created"
	AuditTicketClaimed    = "ticket_claimed"
	AuditSeverityChanged  = "severity_changed"
	AuditTicketClosed     = "ticket_closed"
	AuditTicketReleased   = "ticket_released"
	AuditCustomerChanged  = "customer_changed"
	AuditRoleChanged      = "role_changed"
	AuditDemoStarted      = "demo_started"
	AuditDemoStopped      = "demo_stopped"
	AuditRateLimited      = "rate_limited"
	AuditConnectionClosed = "connection_closed"
)

type AuditEntry struct {
	ID         string
	UserID     string
	Action     string
	TargetType string
	TargetID   string
	IPAddress  string
	Details    string
	CreatedAt  time.Time
}
