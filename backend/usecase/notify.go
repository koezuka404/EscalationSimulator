package usecase

import (
	"encoding/json"
	"log"

	"escalator/entity"
)

type Message struct {
	Staff       bool
	ApplicantID string
	Body        []byte
}

type Notifier interface {
	Send(Message)
}

func publish(notices Notifier, staff bool, applicantID string, payload any) {
	if notices == nil {
		return
	}
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("画面への知らせを作れませんでした: %v", err)
		return
	}
	notices.Send(Message{Staff: staff, ApplicantID: applicantID, Body: body})
}

func ticketView(ticket entity.Ticket, customerName, plan string) map[string]any {
	return map[string]any{
		"id":             ticket.ID,
		"title":          ticket.Title,
		"severity":       ticket.Severity,
		"status":         ticket.Status,
		"priority_score": ticket.PriorityScore,
		"created_at":     ticket.CreatedAt,
		"customer_name":  customerName,
		"plan":           plan,
	}
}
