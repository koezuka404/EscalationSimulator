package entity

import "time"

type LoginAttempt struct {
	ID           string
	Email        string
	IPAddress    string
	FailedCount  int
	LockedUntil  time.Time
	LastFailedAt time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
