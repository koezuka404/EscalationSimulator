package usecase

import (
	"context"
	"errors"
	"log"
	"time"

	"escalator/entity"
	"escalator/port"
	"escalator/rule"
)

type ClaimNextTicket struct {
	claims  port.ClaimStore
	queue   port.TicketQueue
	current *CurrentUser
	notices Notifier
	scores  *RecalcOpenScores
	locks   PriorityLock
	every   time.Duration
	audit   *AuditLog
}

func NewClaimNextTicket(claims port.ClaimStore, queue port.TicketQueue, current *CurrentUser, notices Notifier, scores *RecalcOpenScores, locks PriorityLock, every time.Duration, audit *AuditLog) *ClaimNextTicket {
	return &ClaimNextTicket{claims: claims, queue: queue, current: current, notices: notices, scores: scores, locks: locks, every: every, audit: audit}
}

//担当者がいちばん先の対応待ちを引き取る
func (c *ClaimNextTicket) Execute(ctx context.Context, authorization string) (entity.Ticket, error) {
	actor, err := c.current.Execute(ctx, authorization)
	if err != nil {
		return entity.Ticket{}, err
	}
	if actor.Role != entity.RoleAgent {
		return entity.Ticket{}, rule.ErrClaimAgent
	}
	ticket, err := c.claims.ClaimNext(ctx, actor.ID, c.queue, func(createdAt time.Time, slaMinutes int, at time.Time) bool {
		_, _, _, late := rule.SLAProgress(createdAt, at, slaMinutes, false)
		return late
	})
	if errors.Is(err, rule.ErrQueueMismatch) {
		c.rebuildAfterMismatch(ctx)
		return entity.Ticket{}, rule.ErrQueueUnavailable
	}
	if err != nil {
		return entity.Ticket{}, err
	}
	publish(c.notices, true, ticket.CreatedBy, map[string]any{
		"type":          "ticket_claimed",
		"ticket_id":     ticket.ID,
		"title":         ticket.Title,
		"assignee_id":   actor.ID,
		"assignee_name": actor.Name,
	})
	c.audit.Record(ctx, actor.ID, entity.AuditTicketClaimed, "ticket", ticket.ID, "チケットを引き取りました")
	return ticket, nil
}

//3回担当を付けられなかったとき、対応待ちから待ち順を作り直す
func (c *ClaimNextTicket) rebuildAfterMismatch(ctx context.Context) {
	WithPriorityLock(ctx, c.locks, c.every, "待ち順の作り直しは、別のやり直しの途中だったため飛ばしました", func() {
		if err := c.scores.Rebuild(ctx); err != nil {
			log.Printf("待ち順を作り直せませんでした: %v", err)
		}
	})
}
