package usecase

import "time"

type SessionPolicy struct {
	RefreshTTL  time.Duration
	MaxFailures int
	Lock        time.Duration
}
