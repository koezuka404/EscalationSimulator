package websocket

type snapshotEvent struct {
	Type    string           `json:"type"`
	Tickets []snapshotTicket `json:"tickets"`
	Agents  []snapshotAgent  `json:"agents"`
}

type snapshotTicket struct {
	Rank             int    `json:"rank"`
	ID               string `json:"id"`
	Title            string `json:"title"`
	Severity         int    `json:"severity"`
	Plan             string `json:"plan"`
	WaitMinutes      int    `json:"wait_minutes"`
	PriorityScore    int    `json:"priority_score"`
	RemainingMinutes int    `json:"remaining_minutes"`
	Overdue          bool   `json:"overdue"`
	OverdueMinutes   int    `json:"overdue_minutes"`
	CustomerName     string `json:"customer_name"`
}

type snapshotAgent struct {
	UserID string `json:"user_id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}
