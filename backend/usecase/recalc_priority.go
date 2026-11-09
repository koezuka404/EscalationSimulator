package usecase

import (
	"context"
	"log"
	"time"

	"escalator/entity"
	"escalator/repository"
)

type RecalcOpenScores struct {
	tickets   repository.TicketRepository
	customers repository.CustomerRepository
	queue     repository.TicketQueue
	notices   Notifier
}

func NewRecalcOpenScores(tickets repository.TicketRepository, customers repository.CustomerRepository, queue repository.TicketQueue, notices Notifier) *RecalcOpenScores {
	return &RecalcOpenScores{tickets: tickets, customers: customers, queue: queue, notices: notices}
}

//対応待ちの点数を今の時刻でやり直し、待ち順も同じ点数にする
func (r *RecalcOpenScores) Execute(ctx context.Context) error {
	open, byID, err := r.openTickets(ctx)
	if err != nil {
		return err
	}
	now := time.Now()
	for _, ticket := range open {
		score, ok := r.refreshOpen(ctx, ticket, byID, now)
		if !ok {
			continue
		}
		enqueueCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		err = r.queue.Enqueue(enqueueCtx, ticket.ID, score)
		cancel()
		if err != nil {
			log.Printf("待ち順の点数を直せませんでした。点数は保存してあります。id=%s: %v", ticket.ID, err)
		}
	}
	return nil
}

//待ち順を空にして、今の対応待ちだけを載せ直す
func (r *RecalcOpenScores) Rebuild(ctx context.Context) error {
	open, byID, err := r.openTickets(ctx)
	if err != nil {
		return err
	}
	now := time.Now()
	ids := make([]string, 0, len(open))
	scores := make([]int, 0, len(open))
	for _, ticket := range open {
		score, ok := r.refreshOpen(ctx, ticket, byID, now)
		if !ok {
			continue
		}
		ids = append(ids, ticket.ID)
		scores = append(scores, score)
	}
	replaceCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := r.queue.Replace(replaceCtx, ids, scores); err != nil {
		return err
	}
	log.Printf("待ち順を対応待ちから作り直しました。件数=%d", len(ids))
	return nil
}

func (r *RecalcOpenScores) openTickets(ctx context.Context) ([]entity.Ticket, map[string]entity.Customer, error) {
	open, err := r.tickets.ListOpen(ctx)
	if err != nil {
		return nil, nil, err
	}
	customers, err := r.customers.List(ctx)
	if err != nil {
		return nil, nil, err
	}
	byID := make(map[string]entity.Customer, len(customers))
	for _, customer := range customers {
		byID[customer.ID] = customer
	}
	return open, byID, nil
}

func (r *RecalcOpenScores) refreshOpen(ctx context.Context, ticket entity.Ticket, byID map[string]entity.Customer, now time.Time) (int, bool) {
	customer, ok := byID[ticket.CustomerID]
	if !ok {
		log.Printf("顧客が見つからないため、点数をやり直せませんでした。id=%s", ticket.ID)
		return 0, false
	}
	planScore, err := customer.PlanScore()
	if err != nil {
		log.Printf("点数を計算できませんでした。id=%s: %v", ticket.ID, err)
		return 0, false
	}
	score, err := entity.PriorityScore(ticket.Severity, planScore, ticket.CreatedAt, now, customer.SLAMinutes, false)
	if err != nil {
		log.Printf("点数を計算できませんでした。id=%s: %v", ticket.ID, err)
		return 0, false
	}
	saved, err := r.tickets.UpdateOpenScore(ctx, ticket.ID, score, now)
	if err != nil {
		log.Printf("対応待ちの点数を保存できませんでした。id=%s: %v", ticket.ID, err)
		return 0, false
	}
	if !saved {
		return 0, false
	}
	_, _, overdueMinutes, overdue := entity.SLAProgress(ticket.CreatedAt, now, customer.SLAMinutes, false)
	if overdue && ticket.SlaNotifiedAt.IsZero() {
		marked, err := r.tickets.MarkSLANotified(ctx, ticket.ID, now)
		if err != nil {
			log.Printf("約束時間を過ぎた印を付けられませんでした。id=%s: %v", ticket.ID, err)
		} else if marked {
			publish(r.notices, true, ticket.CreatedBy, map[string]any{
				"type":            "sla_overdue",
				"ticket_id":       ticket.ID,
				"title":           ticket.Title,
				"severity":        ticket.Severity,
				"priority_score":  score,
				"overdue_minutes": overdueMinutes,
			})
		}
	}
	return score, true
}
