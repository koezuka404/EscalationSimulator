package usecase

import (
	"context"
	"time"

	"escalator/repository"
)

type SnapshotAgent struct {
	UserID string
	Name   string
	Status string
}

type QueuePicture struct {
	Tickets []WaitingTicket
	Agents  []SnapshotAgent
}

type QueueSnapshot struct {
	tickets   repository.TicketRepository
	customers repository.CustomerRepository
	users     repository.UserRepository
	statuses  repository.AgentStatusRepository
}

func NewQueueSnapshot(tickets repository.TicketRepository, customers repository.CustomerRepository, users repository.UserRepository, statuses repository.AgentStatusRepository) *QueueSnapshot {
	return &QueueSnapshot{tickets: tickets, customers: customers, users: users, statuses: statuses}
}

//つながった担当者と管理者に、今の待ち順と稼働を返す
func (s *QueueSnapshot) Execute(ctx context.Context) (QueuePicture, error) {
	open, err := s.tickets.ListOpen(ctx)
	if err != nil {
		return QueuePicture{}, err
	}
	customers, err := s.customers.List(ctx)
	if err != nil {
		return QueuePicture{}, err
	}
	statuses, err := s.statuses.List(ctx)
	if err != nil {
		return QueuePicture{}, err
	}
	ids := make([]string, 0, len(statuses))
	for _, status := range statuses {
		ids = append(ids, status.UserID)
	}
	names, err := s.users.ListNames(ctx, ids)
	if err != nil {
		return QueuePicture{}, err
	}
	agents := make([]SnapshotAgent, 0, len(statuses))
	for _, status := range statuses {
		agents = append(agents, SnapshotAgent{
			UserID: status.UserID,
			Name:   names[status.UserID],
			Status: status.Status,
		})
	}
	return QueuePicture{
		Tickets: arrangeWaiting(open, customers, time.Now()),
		Agents:  agents,
	}, nil
}
