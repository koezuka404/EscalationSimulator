package entity

import "time"

const (
	ConnectionOpened    = "connected"
	ConnectionClosed    = "disconnected"
	ConnectionForcedOff = "forced_close"
)

type ConnectionLogEntry struct {
	ID           string
	UserID       string
	ConnectionID string
	EventType    string
	Reason       string
	IPAddress    string
	CreatedAt    time.Time
}
