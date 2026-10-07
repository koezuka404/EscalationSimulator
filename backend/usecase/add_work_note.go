package usecase

import (
	"context"
	"time"

	"escalator/entity"
	"escalator/repository"
)

type SavedWorkNote struct {
	ID         string
	TicketID   string
	Body       string
	AuthorName string
	CreatedAt  time.Time
}

type AddWorkNote struct {
	tickets repository.TicketRepository
	current *CurrentUser
}

func NewAddWorkNote(tickets repository.TicketRepository, current *CurrentUser) *AddWorkNote {
	return &AddWorkNote{tickets: tickets, current: current}
}

//対応メモを保存する
func (a *AddWorkNote) Execute(ctx context.Context, authorization, ticketID, body string) (SavedWorkNote, error) {
	body, err := entity.WorkNoteText(body)
	if err != nil {
		return SavedWorkNote{}, err
	}
	actor, err := a.current.Execute(ctx, authorization)
	if err != nil {
		return SavedWorkNote{}, err
	}
	if actor.Role != entity.RoleAgent && actor.Role != entity.RoleAdmin {
		return SavedWorkNote{}, entity.ErrWorkNoteRole
	}
	ticket, err := a.tickets.FindByID(ctx, ticketID)
	if err != nil {
		return SavedWorkNote{}, err
	}
	if err := entity.CanAddWorkNote(actor.Role, actor.ID, ticket); err != nil {
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
