package entity

import "time"

type Role string

const (
	RoleApplicant Role = "applicant"
	RoleAgent     Role = "agent"
	RoleAdmin     Role = "admin"
)

type AccountStatus string

const (
	StatusActive    AccountStatus = "active"
	StatusSuspended AccountStatus = "suspended"
	StatusDeleted   AccountStatus = "deleted"
)

type User struct {
	ID               string
	Email            string
	PasswordHash     string
	Name             string
	Role             Role
	CustomerID       string
	Status           AccountStatus
	AuthVersion      int
	FailedLoginCount int
	LockedUntil      time.Time
}
