package controller

import (
	"errors"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"escalator/entity"
	"escalator/usecase"
)

type TicketAPI struct {
	create   *usecase.CreateTicket
	close    *usecase.CloseTicket
	severity *usecase.ChangeSeverity
	release  *usecase.ReturnTicketToQueue
	mine     *usecase.ListMyTickets
}

func NewTicketAPI(create *usecase.CreateTicket, close *usecase.CloseTicket, severity *usecase.ChangeSeverity, release *usecase.ReturnTicketToQueue, mine *usecase.ListMyTickets) *TicketAPI {
	return &TicketAPI{create: create, close: close, severity: severity, release: release, mine: mine}
}

//申請者の自分のチケット一覧を返す
func (a *TicketAPI) ListMine(c echo.Context) error {
	tickets, err := a.mine.Execute(c.Request().Context(), c.Request().Header.Get("Authorization"))
	if err != nil {
		return writeMyTicketsError(c, err)
	}
	body := make([]myTicketJSON, 0, len(tickets))
	for _, ticket := range tickets {
		body = append(body, toMyTicketJSON(ticket))
	}
	return c.JSON(http.StatusOK, body)
}

//チケットの起票を受ける
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

//チケットの完了を受ける
func (a *TicketAPI) Close(c echo.Context) error {
	var body closeTicketJSON
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, messageJSON{Message: "送られた内容を読み取れませんでした。終了コメントを入力してください"})
	}
	ticket, err := a.close.Execute(c.Request().Context(), c.Request().Header.Get("Authorization"), c.Param("id"), body.Comment)
	if err != nil {
		return writeCloseTicketError(c, err)
	}
	return c.JSON(http.StatusOK, toTicketJSON(ticket))
}

//緊急度の変更を受ける
func (a *TicketAPI) Escalate(c echo.Context) error {
	var body changeSeverityJSON
	if err := c.Bind(&body); err != nil {
		return c.JSON(http.StatusBadRequest, messageJSON{Message: "送られた内容を読み取れませんでした。緊急度と理由を入力してください"})
	}
	ticket, err := a.severity.Execute(c.Request().Context(), c.Request().Header.Get("Authorization"), c.Param("id"), body.Reason, body.Severity)
	if err != nil {
		return writeChangeSeverityError(c, err)
	}
	return c.JSON(http.StatusOK, toTicketJSON(ticket))
}

//担当を外して待ちに戻す依頼を受ける
func (a *TicketAPI) Release(c echo.Context) error {
	ticket, err := a.release.Execute(c.Request().Context(), c.Request().Header.Get("Authorization"), c.Param("id"))
	if err != nil {
		return writeReleaseError(c, err)
	}
	return c.JSON(http.StatusOK, toTicketJSON(ticket))
}

type changeSeverityJSON struct {
	Severity int    `json:"severity"`
	Reason   string `json:"reason"`
}

type myTicketJSON struct {
	ID               string    `json:"id"`
	Title            string    `json:"title"`
	Severity         int       `json:"severity"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	RemainingMinutes int       `json:"remaining_minutes"`
	Overdue          bool      `json:"overdue"`
	OverdueMinutes   int       `json:"overdue_minutes"`
	AssigneeName     string    `json:"assignee_name"`
}

func toMyTicketJSON(ticket usecase.MyTicket) myTicketJSON {
	return myTicketJSON{
		ID:               ticket.ID,
		Title:            ticket.Title,
		Severity:         ticket.Severity,
		Status:           string(ticket.Status),
		CreatedAt:        ticket.CreatedAt,
		RemainingMinutes: ticket.RemainingMinutes,
		Overdue:          ticket.Overdue,
		OverdueMinutes:   ticket.OverdueMinutes,
		AssigneeName:     ticket.AssigneeName,
	}
}

func writeMyTicketsError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, entity.ErrUnauthenticated):
		return c.JSON(http.StatusUnauthorized, messageJSON{Message: err.Error()})
	case errors.Is(err, entity.ErrMyTicketsForbidden):
		return c.JSON(http.StatusForbidden, messageJSON{Message: err.Error()})
	default:
		return c.JSON(http.StatusInternalServerError, messageJSON{Message: "チケットの一覧を返せませんでした。しばらくしてから、もう一度試してください"})
	}
}

type closeTicketJSON struct {
	Comment string `json:"comment"`
}

type createTicketJSON struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Severity    int    `json:"severity"`
}

type ticketJSON struct {
	ID            string     `json:"id"`
	CustomerID    string     `json:"customer_id"`
	CreatedBy     string     `json:"created_by"`
	Title         string     `json:"title"`
	Description   string     `json:"description"`
	Category      string     `json:"category"`
	Severity      int        `json:"severity"`
	Status        string     `json:"status"`
	AssigneeID    string     `json:"assignee_id"`
	PriorityScore int        `json:"priority_score"`
	CreatedAt     time.Time  `json:"created_at"`
	ClaimedAt     *time.Time `json:"claimed_at,omitempty"`
	ClosedAt      *time.Time `json:"closed_at,omitempty"`
	CloseComment  string     `json:"close_comment,omitempty"`
}

func toTicketJSON(ticket entity.Ticket) ticketJSON {
	body := ticketJSON{
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
	if !ticket.ClaimedAt.IsZero() {
		claimedAt := ticket.ClaimedAt
		body.ClaimedAt = &claimedAt
	}
	if !ticket.ClosedAt.IsZero() {
		closedAt := ticket.ClosedAt
		body.ClosedAt = &closedAt
	}
	body.CloseComment = ticket.CloseComment
	return body
}

func writeCreateTicketError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, entity.ErrUnauthenticated):
		return c.JSON(http.StatusUnauthorized, messageJSON{Message: err.Error()})
	case errors.Is(err, entity.ErrTicketApplicant):
		return c.JSON(http.StatusForbidden, messageJSON{Message: err.Error()})
	case errors.Is(err, entity.ErrCustomerRequired):
		return c.JSON(http.StatusConflict, messageJSON{Message: err.Error()})
	case errors.Is(err, entity.ErrCustomerNotFound):
		return c.JSON(http.StatusNotFound, messageJSON{Message: err.Error()})
	case errors.Is(err, entity.ErrInvalidTitle),
		errors.Is(err, entity.ErrInvalidDescription),
		errors.Is(err, entity.ErrInvalidCategory),
		errors.Is(err, entity.ErrInvalidSeverity):
		return c.JSON(http.StatusBadRequest, messageJSON{Message: err.Error()})
	default:
		return c.JSON(http.StatusInternalServerError, messageJSON{Message: "チケットを起票できませんでした。しばらくしてから、もう一度試してください"})
	}
}

func writeCloseTicketError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, entity.ErrUnauthenticated):
		return c.JSON(http.StatusUnauthorized, messageJSON{Message: err.Error()})
	case errors.Is(err, entity.ErrCloseForbidden):
		return c.JSON(http.StatusForbidden, messageJSON{Message: err.Error()})
	case errors.Is(err, entity.ErrTicketNotFound):
		return c.JSON(http.StatusNotFound, messageJSON{Message: err.Error()})
	case errors.Is(err, entity.ErrNotInProgress):
		return c.JSON(http.StatusConflict, messageJSON{Message: err.Error()})
	case errors.Is(err, entity.ErrInvalidCloseComment):
		return c.JSON(http.StatusBadRequest, messageJSON{Message: err.Error()})
	default:
		return c.JSON(http.StatusInternalServerError, messageJSON{Message: "チケットを完了できませんでした。しばらくしてから、もう一度試してください"})
	}
}

func writeChangeSeverityError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, entity.ErrUnauthenticated):
		return c.JSON(http.StatusUnauthorized, messageJSON{Message: err.Error()})
	case errors.Is(err, entity.ErrSeverityRole),
		errors.Is(err, entity.ErrSeverityLower),
		errors.Is(err, entity.ErrSeverityRaise):
		return c.JSON(http.StatusForbidden, messageJSON{Message: err.Error()})
	case errors.Is(err, entity.ErrTicketNotFound),
		errors.Is(err, entity.ErrCustomerNotFound):
		return c.JSON(http.StatusNotFound, messageJSON{Message: err.Error()})
	case errors.Is(err, entity.ErrSeverityClosed):
		return c.JSON(http.StatusConflict, messageJSON{Message: err.Error()})
	case errors.Is(err, entity.ErrInvalidSeverity),
		errors.Is(err, entity.ErrInvalidSeverityReason),
		errors.Is(err, entity.ErrSeveritySame):
		return c.JSON(http.StatusBadRequest, messageJSON{Message: err.Error()})
	default:
		return c.JSON(http.StatusInternalServerError, messageJSON{Message: "緊急度を変更できませんでした。しばらくしてから、もう一度試してください"})
	}
}

func writeReleaseError(c echo.Context, err error) error {
	switch {
	case errors.Is(err, entity.ErrUnauthenticated):
		return c.JSON(http.StatusUnauthorized, messageJSON{Message: err.Error()})
	case errors.Is(err, entity.ErrReleaseForbidden):
		return c.JSON(http.StatusForbidden, messageJSON{Message: err.Error()})
	case errors.Is(err, entity.ErrTicketNotFound),
		errors.Is(err, entity.ErrCustomerNotFound):
		return c.JSON(http.StatusNotFound, messageJSON{Message: err.Error()})
	case errors.Is(err, entity.ErrNotInProgress):
		return c.JSON(http.StatusConflict, messageJSON{Message: err.Error()})
	default:
		return c.JSON(http.StatusInternalServerError, messageJSON{Message: "チケットを待ちに戻せませんでした。しばらくしてから、もう一度試してください"})
	}
}
