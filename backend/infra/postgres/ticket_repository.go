package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"escalator/domain"
)

type ticketRow struct {
	ID            string `gorm:"primaryKey"`
	CustomerID    string `gorm:"index"`
	CreatedBy     string `gorm:"index"`
	Title         string
	Description   string
	Severity      int
	Category      string
	Status        string `gorm:"index"`
	AssigneeID    string
	PriorityScore int
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (ticketRow) TableName() string { return "tickets" }

type TicketRepository struct {
	db *gorm.DB
}

func NewTicketRepository(db *gorm.DB) *TicketRepository {
	return &TicketRepository{db: db}
}

func MigrateTickets(db *gorm.DB) error {
	return db.AutoMigrate(&ticketRow{})
}

//チケットを保存する
func (r *TicketRepository) Save(ctx context.Context, ticket domain.Ticket) (domain.Ticket, error) {
	if ticket.ID == "" {
		ticket.ID = uuid.NewString()
	}
	row := ticketRow{
		ID:            ticket.ID,
		CustomerID:    ticket.CustomerID,
		CreatedBy:     ticket.CreatedBy,
		Title:         ticket.Title,
		Description:   ticket.Description,
		Severity:      ticket.Severity,
		Category:      string(ticket.Category),
		Status:        string(ticket.Status),
		AssigneeID:    ticket.AssigneeID,
		PriorityScore: ticket.PriorityScore,
		CreatedAt:     ticket.CreatedAt,
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return domain.Ticket{}, err
	}
	ticket.CreatedAt = row.CreatedAt
	return ticket, nil
}

//対応待ちのチケットを返す詳細の本文は読まない
func (r *TicketRepository) ListOpen(ctx context.Context) ([]domain.Ticket, error) {
	var rows []ticketRow
	err := r.db.WithContext(ctx).
		Select("id", "customer_id", "created_by", "title", "severity", "category", "status", "assignee_id", "priority_score", "created_at").
		Where("status = ?", string(domain.TicketOpen)).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	tickets := make([]domain.Ticket, 0, len(rows))
	for _, row := range rows {
		tickets = append(tickets, toTicket(row))
	}
	return tickets, nil
}

func toTicket(row ticketRow) domain.Ticket {
	return domain.Ticket{
		ID:            row.ID,
		CustomerID:    row.CustomerID,
		CreatedBy:     row.CreatedBy,
		Title:         row.Title,
		Severity:      row.Severity,
		Category:      domain.Category(row.Category),
		Status:        domain.TicketStatus(row.Status),
		AssigneeID:    row.AssigneeID,
		PriorityScore: row.PriorityScore,
		CreatedAt:     row.CreatedAt,
	}
}
