package entity

import "time"

type RequestLogEntry struct {
	ID        string
	RequestID string
	UserID    string
	Method    string
	Path      string
	Status    int
	LatencyMS int
	ErrorCode string
	CreatedAt time.Time
}
