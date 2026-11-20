package entity

import "time"

const (
	DemoRunning   = "running"
	DemoStopped   = "stopped"
	DemoCompleted = "completed"
	DemoEven      = "even"
	DemoIncident  = "incident"
)

type DemoRun struct {
	ID             string
	StartedBy      string
	TotalCount     int
	IntervalSec    int
	Distribution   string
	Status         string
	GeneratedCount int
	StartedAt      time.Time
	StoppedAt      time.Time
}
