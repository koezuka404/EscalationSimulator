package usecase

import (
	"context"
	"time"

	"escalator/entity"
	"escalator/port"
	"escalator/rule"
)

type SavedWorkNote struct {
	ID         string
	TicketID   string
	Body       string
	AuthorName string
	CreatedAt  time.Time
}

type AddWorkNote struct {
	tickets port.TicketRepository
	current *CurrentUser
}

func NewAddWorkNote(tickets port.TicketRepository, current *CurrentUser) *AddWorkNote {
	return &AddWorkNote{tickets: tickets, current: current}
}

// 対応メモを保存する
func (a *AddWorkNote) Execute(ctx context.Context, authorization, ticketID, body string) (SavedWorkNote, error) {
	body, err := rule.WorkNoteText(body)
	if err != nil {
		return SavedWorkNote{}, err
	}
	actor, err := a.current.Execute(ctx, authorization)
	if err != nil {
		return SavedWorkNote{}, err
	}
	if actor.Role != entity.RoleAgent && actor.Role != entity.RoleAdmin {
		return SavedWorkNote{}, rule.ErrWorkNoteRole
	}
	ticket, err := a.tickets.FindByID(ctx, ticketID)
	if err != nil {
		return SavedWorkNote{}, err
	}
	if err := rule.CanAddWorkNote(actor.Role, actor.ID, ticket); err != nil {
		return SavedWorkNote{}, err
	}
	saved, err := a.tickets.AddWorkNote(ctx, ticket.ID, actor.ID, body, time.Now())
	if err != nil {
		return SavedWorkNote{}, err
	}
	return SavedWorkNote{
		ID:         saved.ID,
		TicketID:   saved.TicketID,
		Body:       saved.Body,
		AuthorName: actor.Name,
		CreatedAt:  saved.CreatedAt,
	}, nil
}
