package entity

import "time"

type TicketStatus string

const (
	TicketOpen       TicketStatus = "open"
	TicketInProgress TicketStatus = "in_progress"
	TicketClosed     TicketStatus = "closed"
)

type Category string

const (
	CategoryIncident Category = "incident"
	CategoryBug      Category = "bug"
	CategoryQuestion Category = "question"
	CategoryRequest  Category = "request"
	CategoryOther    Category = "other"
)

const WorkNoteKind = "work_note"

type WorkNote struct {
	ID        string
	TicketID  string
	UserID    string
	Body      string
	CreatedAt time.Time
}

type SeverityChange struct {
	FromSeverity int
	ToSeverity   int
	Reason       string
	ChangedBy    string
	CreatedAt    time.Time
}

type Ticket struct {
	ID               string
	CustomerID       string
	CreatedBy        string
	Title            string
	Description      string
	Severity         int
	Category         Category
	Status           TicketStatus
	AssigneeID       string
	PriorityScore    int
	CreatedAt        time.Time
	ClaimedAt        time.Time
	FirstClaimedAt   time.Time
	OverdueClaimedAt time.Time
	ClosedAt         time.Time
	CloseComment     string
	SlaNotifiedAt    time.Time
}
