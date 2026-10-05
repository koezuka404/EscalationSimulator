package controller

import (
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"escalator/domain"
	"escalator/usecase"
)

// TicketAPI は申請者のチケット起票を受ける。
type TicketAPI struct {
	create *usecase.CreateTicket
}

func NewTicketAPI(create *usecase.CreateTicket) *TicketAPI {
	return &TicketAPI{create: create}
}

func (a *TicketAPI) Create(c echo.Context) error {
	var body createTicketJSON
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, messageJSON{Message: "送られた内容を読み取れませんでした。件名、詳細、種類、緊急度を入力してください"})
	}
	ticket, err := a.create.Execute(
		c.Request().Context(),
		c.Request().Header.Get("Authorization"),
		body.Title,
		body.Description,
		body.Category,
		body.Severity,
	)
	if err != nil {
		return writeCreateTicketError(c, err)
	}
	return c.JSON(http.StatusCreated, toTicketJSON(ticket))
}

type createTicketJSON struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Severity    int    `json:"severity"`
}

type ticketJSON struct {
	ID            string    `json:"id"`
	CustomerID    string    `json:"customer_id"`
	CreatedBy     string    `json:"created_by"`
	Title         string    `json:"title"`
	Description   string    `json:"description"`
	Category      string    `json:"category"`
	Severity      int       `json:"severity"`
	Status        string    `json:"status"`
	AssigneeID    string    `json:"assignee_id"`
	PriorityScore int       `json:"priority_score"`
	CreatedAt     time.Time `json:"created_at"`
}

func toTicketJSON(ticket domain.Ticket) ticketJSON {
	return ticketJSON{
		ID:            ticket.ID,
		CustomerID:    ticket.CustomerID,
		CreatedBy:     ticket.CreatedBy,
		Title:         ticket.Title,
		Description:   ticket.Description,
		Category:      string(ticket.Category),
		Severity:      ticket.Severity,
		Status:        string(ticket.Status),
		AssigneeID:    ticket.AssigneeID,
		PriorityScore: ticket.PriorityScore,
		CreatedAt:     ticket.CreatedAt,
	}
}

func writeCreateTicketError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, domain.ErrUnauthenticated):
		return c.JSON(http.StatusUnauthorized, messageJSON{Message: err.Error()})
	case errors.Is(err, domain.ErrTicketApplicant):
		return c.JSON(http.StatusForbidden, messageJSON{Message: err.Error()})
	case errors.Is(err, domain.ErrCustomerRequired):
		return c.JSON(http.StatusConflict, messageJSON{Message: err.Error()})
	case errors.Is(err, domain.ErrCustomerNotFound):
		return c.JSON(http.StatusNotFound, messageJSON{Message: err.Error()})
	case errors.Is(err, domain.ErrInvalidTitle),
		errors.Is(err, domain.ErrInvalidDescription),
		errors.Is(err, domain.ErrInvalidCategory),
		errors.Is(err, domain.ErrInvalidSeverity):
		return c.JSON(http.StatusBadRequest, messageJSON{Message: err.Error()})
	default:
		return c.JSON(http.StatusInternalServerError, messageJSON{Message: "チケットを起票できませんでした。しばらくしてから、もう一度試してください"})
	}
}
