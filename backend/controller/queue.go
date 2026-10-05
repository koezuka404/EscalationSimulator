package controller

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"escalator/domain"
	"escalator/usecase"
)

type QueueAPI struct {
	list *usecase.ListWaitingTickets
}

func NewQueueAPI(list *usecase.ListWaitingTickets) *QueueAPI {
	return &QueueAPI{list: list}
}

//待ち順の一覧を返す
func (a *QueueAPI) List(c echo.Context) error {
	waiting, err := a.list.Execute(c.Request().Context(), c.Request().Header.Get("Authorization"))
	if err != nil {
		return writeQueueError(c, err)
	}
	body := make([]waitingTicketJSON, 0, len(waiting))
	for _, ticket := range waiting {
		body = append(body, toWaitingTicketJSON(ticket))
	}
	return c.JSON(http.StatusOK, body)
}

type waitingTicketJSON struct {
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

func toWaitingTicketJSON(ticket usecase.WaitingTicket) waitingTicketJSON {
	return waitingTicketJSON{
		Rank:             ticket.Rank,
		ID:               ticket.ID,
		Title:            ticket.Title,
		Severity:         ticket.Severity,
		Plan:             string(ticket.Plan),
		WaitMinutes:      ticket.WaitMinutes,
		PriorityScore:    ticket.PriorityScore,
		RemainingMinutes: ticket.RemainingMinutes,
		Overdue:          ticket.Overdue,
		OverdueMinutes:   ticket.OverdueMinutes,
		CustomerName:     ticket.CustomerName,
	}
}

func writeQueueError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, domain.ErrUnauthenticated):
		return c.JSON(http.StatusUnauthorized, messageJSON{Message: err.Error()})
	case errors.Is(err, domain.ErrQueueForbidden):
		return c.JSON(http.StatusForbidden, messageJSON{Message: err.Error()})
	default:
		return c.JSON(http.StatusInternalServerError, messageJSON{Message: "待ち順を表示できませんでした。しばらくしてから、もう一度試してください"})
	}
}
