package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"

	"escalator/entity"
	"escalator/usecase"
)

type AdminTicketAPI struct {
	search *usecase.SearchTickets
}

func NewAdminTicketAPI(search *usecase.SearchTickets) *AdminTicketAPI {
	return &AdminTicketAPI{search: search}
}

//管理者の全件の絞り込みを受ける
func (a *AdminTicketAPI) Search(c echo.Context) error {
	severity, err := querySeverity(c.QueryParam("severity"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, messageJSON{Message: err.Error()})
	}
	tickets, err := a.search.Execute(
		c.Request().Context(),
		c.Request().Header.Get("Authorization"),
		strings.TrimSpace(c.QueryParam("status")),
		severity,
		strings.TrimSpace(c.QueryParam("customer_id")),
		strings.TrimSpace(c.QueryParam("assignee_id")),
	)
	if err != nil {
		return writeAdminTicketError(c, err)
	}
	body := make([]adminTicketJSON, 0, len(tickets))
	for _, ticket := range tickets {
		body = append(body, toAdminTicketJSON(ticket))
	}
	return c.JSON(http.StatusOK, body)
}

func querySeverity(raw string) (int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	severity, err := strconv.Atoi(raw)
	if err != nil || severity < 1 || severity > 4 {
		return 0, entity.ErrInvalidSeverity
	}
	return severity, nil
}

type adminTicketJSON struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	Severity         int       `json:"severity"`
	Status           string    `json:"status"`
	CustomerID       string    `json:"customer_id"`
	CustomerName     string    `json:"customer_name"`
	Plan             string    `json:"plan"`
	AssigneeID       string    `json:"assignee_id,omitempty"`
	AssigneeName     string    `json:"assignee_name,omitempty"`
	PriorityScore    int       `json:"priority_score"`
	CreatedAt        time.Time `json:"created_at"`
	RemainingMinutes int       `json:"remaining_minutes"`
	Overdue          bool      `json:"overdue"`
	OverdueMinutes   int       `json:"overdue_minutes"`
}

func toAdminTicketJSON(ticket usecase.FoundTicket) adminTicketJSON {
	return adminTicketJSON{
		ID:               ticket.ID,
		Title:            ticket.Title,
		Severity:         ticket.Severity,
		Status:           string(ticket.Status),
		CustomerID:       ticket.CustomerID,
		CustomerName:     ticket.CustomerName,
		Plan:             string(ticket.Plan),
		AssigneeID:       ticket.AssigneeID,
		AssigneeName:     ticket.AssigneeName,
		PriorityScore:    ticket.PriorityScore,
		CreatedAt:        ticket.CreatedAt,
		RemainingMinutes: ticket.RemainingMinutes,
		Overdue:          ticket.Overdue,
		OverdueMinutes:   ticket.OverdueMinutes,
	}
}

func writeAdminTicketError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, entity.ErrUnauthenticated):
		return c.JSON(http.StatusUnauthorized, messageJSON{Message: err.Error()})
	case errors.Is(err, entity.ErrForbidden):
		return c.JSON(http.StatusForbidden, messageJSON{Message: err.Error()})
	case errors.Is(err, entity.ErrInvalidTicketStatus), errors.Is(err, entity.ErrInvalidSeverity):
		return c.JSON(http.StatusBadRequest, messageJSON{Message: err.Error()})
	default:
		return c.JSON(http.StatusInternalServerError, messageJSON{Message: "チケットの一覧をまとめられませんでした。しばらくしてから、もう一度試してください"})
	}
}
