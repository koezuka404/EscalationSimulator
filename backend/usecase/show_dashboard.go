package usecase

import (
	"context"
	"log"
	"time"

	"escalator/entity"
	"escalator/repository"
)

type Dashboard struct {
	WaitingCount                int
	InProgressCount             int
	AverageHandleMinutes        int
	AverageFirstResponseMinutes int
	OverdueCount                int
	Overdue24hCount             int
	AvailableCount              int
	BusyCount                   int
	OfflineCount                int
}

type ShowDashboard struct {
	tickets   repository.TicketRepository
	customers repository.CustomerRepository
	statuses  repository.AgentStatusRepository
	current   *CurrentUser
	notices   Notifier
}

func NewShowDashboard(tickets repository.TicketRepository, customers repository.CustomerRepository, statuses repository.AgentStatusRepository, current *CurrentUser) *ShowDashboard {
	return &ShowDashboard{tickets: tickets, customers: customers, statuses: statuses, current: current}
}

func (s *ShowDashboard) SetNotifier(notices Notifier) {
	s.notices = notices
}

//管理者に、現場の数字を返す
func (s *ShowDashboard) Execute(ctx context.Context, authorization string) (Dashboard, error) {
	actor, err := s.current.Execute(ctx, authorization)
	if err != nil {
		return Dashboard{}, err
	}
	if actor.Role != entity.RoleAdmin {
		return Dashboard{}, entity.ErrForbidden
	}
	return s.Collect(ctx)
}

//待ち、対応中、平均時間、約束時間オーバー、担当者の空きをまとめる
func (s *ShowDashboard) Collect(ctx context.Context) (Dashboard, error) {
	now := time.Now()
	since := now.Add(-24 * time.Hour)
	tickets, err := s.tickets.ListForDashboard(ctx, since)
	if err != nil {
		return Dashboard{}, err
	}
	customers, err := s.customers.List(ctx)
	if err != nil {
		return Dashboard{}, err
	}
	statuses, err := s.statuses.List(ctx)
	if err != nil {
		return Dashboard{}, err
	}
	byID := make(map[string]entity.Customer, len(customers))
	for _, customer := range customers {
		byID[customer.ID] = customer
	}
	var handle []int
	var firstResponse []int
	numbers := Dashboard{}
	for _, ticket := range tickets {
		switch ticket.Status {
		case entity.TicketOpen:
			numbers.WaitingCount++
			customer := byID[ticket.CustomerID]
			_, _, _, overdue := entity.SLAProgress(ticket.CreatedAt, now, customer.SLAMinutes, false)
			if overdue {
				numbers.OverdueCount++
			}
		case entity.TicketInProgress:
			numbers.InProgressCount++
		}
		if !ticket.ClosedAt.IsZero() && !ticket.ClosedAt.Before(since) && !ticket.ClaimedAt.IsZero() && !ticket.ClosedAt.Before(ticket.ClaimedAt) {
			handle = append(handle, elapsedMinutes(ticket.ClaimedAt, ticket.ClosedAt))
		}
		if !ticket.FirstClaimedAt.IsZero() && !ticket.FirstClaimedAt.Before(since) && !ticket.FirstClaimedAt.Before(ticket.CreatedAt) {
			firstResponse = append(firstResponse, elapsedMinutes(ticket.CreatedAt, ticket.FirstClaimedAt))
		}
		if !ticket.OverdueClaimedAt.IsZero() && !ticket.OverdueClaimedAt.Before(since) {
			numbers.Overdue24hCount++
		}
	}
	numbers.AverageHandleMinutes = averageMinutes(handle)
	numbers.AverageFirstResponseMinutes = averageMinutes(firstResponse)
	for _, status := range statuses {
		switch entity.AgentAvailability(status.Status) {
		case entity.AgentAvailable:
			numbers.AvailableCount++
		case entity.AgentBusy:
			numbers.BusyCount++
		case entity.AgentOffline:
			numbers.OfflineCount++
		}
	}
	return numbers, nil
}

//管理者の画面へ、現場の数字を届ける
func (s *ShowDashboard) Publish(ctx context.Context) {
	numbers, err := s.Collect(ctx)
	if err != nil {
		log.Printf("現場の数字をまとめられませんでした: %v", err)
		return
	}
	publishAdmins(s.notices, DashboardMessage(numbers))
}

func DashboardMessage(numbers Dashboard) map[string]any {
	return map[string]any{
		"type":      "dashboard_updated",
		"dashboard": dashboardPayload(numbers),
	}
}

func dashboardPayload(numbers Dashboard) map[string]any {
	return map[string]any{
		"waiting_count":                  numbers.WaitingCount,
		"in_progress_count":              numbers.InProgressCount,
		"average_handle_minutes":         numbers.AverageHandleMinutes,
		"average_first_response_minutes": numbers.AverageFirstResponseMinutes,
		"overdue_count":                  numbers.OverdueCount,
		"overdue_24h_count":              numbers.Overdue24hCount,
		"available_count":                numbers.AvailableCount,
		"busy_count":                     numbers.BusyCount,
		"offline_count":                  numbers.OfflineCount,
	}
}

func elapsedMinutes(from, to time.Time) int {
	minutes := int(to.Sub(from).Minutes())
	if minutes < 0 {
		return 0
	}
	return minutes
}

func averageMinutes(samples []int) int {
	if len(samples) == 0 {
		return 0
	}
	sum := 0
	for _, sample := range samples {
		sum += sample
	}
	return sum / len(samples)
}
