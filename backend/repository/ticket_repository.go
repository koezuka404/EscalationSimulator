package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"escalator/entity"
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
	ClaimedAt     *time.Time
	ClosedAt      *time.Time
	CloseComment  string
	SlaNotifiedAt *time.Time
	UpdatedAt     time.Time
}

func (ticketRow) TableName() string { return "tickets" }

type ticketRepository struct {
	db *gorm.DB
}

func NewTicketRepository(db *gorm.DB) TicketRepository {
	return &ticketRepository{db: db}
}

func MigrateTickets(db *gorm.DB) error {
	return db.AutoMigrate(&ticketRow{})
}

type severityHistoryRow struct {
	ID           string `gorm:"primaryKey"`
	TicketID     string `gorm:"index"`
	FromSeverity int
	ToSeverity   int
	Reason       string
	ChangedBy    string `gorm:"index"`
	CreatedAt    time.Time
}

func (severityHistoryRow) TableName() string { return "ticket_severity_histories" }

func MigrateSeverityHistories(db *gorm.DB) error {
	return db.AutoMigrate(&severityHistoryRow{})
}

type workNoteRow struct {
	ID        string `gorm:"primaryKey"`
	TicketID  string `gorm:"index"`
	UserID    string `gorm:"index"`
	Body      string
	Kind      string `gorm:"index"`
	CreatedAt time.Time
}

func (workNoteRow) TableName() string { return "ticket_comments" }

func MigrateWorkNotes(db *gorm.DB) error {
	return db.AutoMigrate(&workNoteRow{})
}

//チケットを保存する
func (r *ticketRepository) Save(ctx context.Context, ticket entity.Ticket) (entity.Ticket, error) {
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
		return entity.Ticket{}, err
	}
	ticket.CreatedAt = row.CreatedAt
	return ticket, nil
}

func (r *ticketRepository) FindByID(ctx context.Context, id string) (entity.Ticket, error) {
	var row ticketRow
	err := r.db.WithContext(ctx).First(&row, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return entity.Ticket{}, entity.ErrTicketNotFound
	}
	if err != nil {
		return entity.Ticket{}, err
	}
	return toTicket(row), nil
}

//対応中のチケットを完了にし担当者を待機中に戻す
func (r *ticketRepository) Close(ctx context.Context, id, comment string, closedAt time.Time) (entity.Ticket, error) {
	var closed entity.Ticket
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row ticketRow
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "id = ?", id).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.ErrTicketNotFound
		}
		if err != nil {
			return err
		}
		if row.Status != string(entity.TicketInProgress) {
			return entity.ErrNotInProgress
		}
		result := tx.Model(&ticketRow{}).Where("id = ? AND status = ?", id, string(entity.TicketInProgress)).Updates(map[string]any{
			"status":        string(entity.TicketClosed),
			"closed_at":     closedAt,
			"close_comment": comment,
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return entity.ErrNotInProgress
		}
		if row.AssigneeID != "" {
			if err := setAgentAvailable(tx, row.AssigneeID); err != nil {
				return err
			}
		}
		if err := tx.First(&row, "id = ?", id).Error; err != nil {
			return err
		}
		closed = toTicket(row)
		return nil
	})
	if err != nil {
		return entity.Ticket{}, err
	}
	return closed, nil
}

//対応中を待ちに戻し、担当者を待機中にする
func (r *ticketRepository) Release(ctx context.Context, id string, planScore, slaMinutes int, now time.Time) (entity.Ticket, error) {
	var released entity.Ticket
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row ticketRow
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "id = ?", id).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.ErrTicketNotFound
		}
		if err != nil {
			return err
		}
		if row.Status != string(entity.TicketInProgress) {
			return entity.ErrNotInProgress
		}
		score, err := entity.PriorityScore(row.Severity, planScore, row.CreatedAt, now, slaMinutes, false)
		if err != nil {
			return err
		}
		result := tx.Model(&ticketRow{}).Where("id = ? AND status = ?", id, string(entity.TicketInProgress)).Updates(map[string]any{
			"status":         string(entity.TicketOpen),
			"assignee_id":    "",
			"claimed_at":     nil,
			"priority_score": score,
			"updated_at":     now,
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return entity.ErrNotInProgress
		}
		if row.AssigneeID != "" {
			if err := setAgentAvailable(tx, row.AssigneeID); err != nil {
				return err
			}
		}
		if err := tx.First(&row, "id = ?", id).Error; err != nil {
			return err
		}
		released = toTicket(row)
		return nil
	})
	if err != nil {
		return entity.Ticket{}, err
	}
	return released, nil
}

func setAgentAvailable(tx *gorm.DB, agentID string) error {
	result := tx.Model(&agentStatusRow{}).Where("user_id = ?", agentID).Update("status", string(entity.AgentAvailable))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		return nil
	}
	return tx.Create(&agentStatusRow{UserID: agentID, Status: string(entity.AgentAvailable)}).Error
}

//緊急度と点数を保存し、変更の履歴を残す
func (r *ticketRepository) UpdateSeverity(ctx context.Context, id string, severity, planScore, slaMinutes int, reason, changedBy string, now time.Time) (entity.Ticket, error) {
	var updated entity.Ticket
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row ticketRow
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "id = ?", id).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.ErrTicketNotFound
		}
		if err != nil {
			return err
		}
		if row.Status == string(entity.TicketClosed) {
			return entity.ErrSeverityClosed
		}
		if severity == row.Severity {
			return entity.ErrSeveritySame
		}
		assigned := row.Status == string(entity.TicketInProgress)
		score, err := entity.PriorityScore(severity, planScore, row.CreatedAt, now, slaMinutes, assigned)
		if err != nil {
			return err
		}
		result := tx.Model(&ticketRow{}).Where("id = ? AND status <> ?", id, string(entity.TicketClosed)).Updates(map[string]any{
			"severity":       severity,
			"priority_score": score,
			"updated_at":     now,
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return entity.ErrSeverityClosed
		}
		history := severityHistoryRow{
			ID:           uuid.NewString(),
			TicketID:     id,
			FromSeverity: row.Severity,
			ToSeverity:   severity,
			Reason:       reason,
			ChangedBy:    changedBy,
			CreatedAt:    now,
		}
		if err := tx.Create(&history).Error; err != nil {
			return err
		}
		if err := tx.First(&row, "id = ?", id).Error; err != nil {
			return err
		}
		updated = toTicket(row)
		return nil
	})
	if err != nil {
		return entity.Ticket{}, err
	}
	return updated, nil
}

//対応待ちの点数だけを保存する対応中や完了は変えない
func (r *ticketRepository) UpdateOpenScore(ctx context.Context, id string, score int, now time.Time) (bool, error) {
	result := r.db.WithContext(ctx).Model(&ticketRow{}).
		Where("id = ? AND status = ?", id, string(entity.TicketOpen)).
		Updates(map[string]any{
			"priority_score": score,
			"updated_at":     now,
		})
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

//まだ知らせていない約束時間オーバーに、一度だけ印を付ける
func (r *ticketRepository) MarkSLANotified(ctx context.Context, id string, at time.Time) (bool, error) {
	result := r.db.WithContext(ctx).Model(&ticketRow{}).
		Where("id = ? AND status = ? AND sla_notified_at IS NULL", id, string(entity.TicketOpen)).
		Update("sla_notified_at", at)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

//申請者が起票したチケットを、更新が新しい順で返す詳細の本文は読まない
func (r *ticketRepository) ListByCreator(ctx context.Context, createdBy string) ([]entity.Ticket, error) {
	var rows []ticketRow
	err := r.db.WithContext(ctx).
		Select("id", "customer_id", "created_by", "title", "severity", "category", "status", "assignee_id", "priority_score", "created_at").
		Where("created_by = ?", createdBy).
		Order("updated_at DESC").
		Order("id DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	tickets := make([]entity.Ticket, 0, len(rows))
	for _, row := range rows {
		tickets = append(tickets, toTicket(row))
	}
	return tickets, nil
}

//緊急度の変更履歴を古い順で返す
func (r *ticketRepository) ListSeverityChanges(ctx context.Context, ticketID string) ([]entity.SeverityChange, error) {
	var rows []severityHistoryRow
	err := r.db.WithContext(ctx).
		Where("ticket_id = ?", ticketID).
		Order("created_at ASC").
		Order("id ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	changes := make([]entity.SeverityChange, 0, len(rows))
	for _, row := range rows {
		changes = append(changes, entity.SeverityChange{
			FromSeverity: row.FromSeverity,
			ToSeverity:   row.ToSeverity,
			Reason:       row.Reason,
			ChangedBy:    row.ChangedBy,
			CreatedAt:    row.CreatedAt,
		})
	}
	return changes, nil
}

//対応メモを保存する完了したチケットには書かない
func (r *ticketRepository) AddWorkNote(ctx context.Context, ticketID, userID, body string, createdAt time.Time) (entity.WorkNote, error) {
	var saved entity.WorkNote
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row ticketRow
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "id = ?", ticketID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.ErrTicketNotFound
		}
		if err != nil {
			return err
		}
		if row.Status == string(entity.TicketClosed) {
			return entity.ErrWorkNoteClosed
		}
		note := workNoteRow{
			ID:        uuid.NewString(),
			TicketID:  ticketID,
			UserID:    userID,
			Body:      body,
			Kind:      entity.WorkNoteKind,
			CreatedAt: createdAt,
		}
		if err := tx.Create(&note).Error; err != nil {
			return err
		}
		if err := tx.Model(&ticketRow{}).Where("id = ?", ticketID).Update("updated_at", createdAt).Error; err != nil {
			return err
		}
		saved = entity.WorkNote{
			ID:        note.ID,
			TicketID:  note.TicketID,
			UserID:    note.UserID,
			Body:      note.Body,
			CreatedAt: note.CreatedAt,
		}
		return nil
	})
	if err != nil {
		return entity.WorkNote{}, err
	}
	return saved, nil
}

//対応メモを書いた順で返す
func (r *ticketRepository) ListWorkNotes(ctx context.Context, ticketID string) ([]entity.WorkNote, error) {
	var rows []workNoteRow
	err := r.db.WithContext(ctx).
		Where("ticket_id = ? AND kind = ?", ticketID, entity.WorkNoteKind).
		Order("created_at ASC").
		Order("id ASC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	notes := make([]entity.WorkNote, 0, len(rows))
	for _, row := range rows {
		notes = append(notes, entity.WorkNote{
			ID:        row.ID,
			TicketID:  row.TicketID,
			UserID:    row.UserID,
			Body:      row.Body,
			CreatedAt: row.CreatedAt,
		})
	}
	return notes, nil
}

//対応待ちのチケットを返す詳細の本文は読まない
func (r *ticketRepository) ListOpen(ctx context.Context) ([]entity.Ticket, error) {
	var rows []ticketRow
	err := r.db.WithContext(ctx).
		Select("id", "customer_id", "created_by", "title", "severity", "category", "status", "assignee_id", "priority_score", "created_at", "sla_notified_at").
		Where("status = ?", string(entity.TicketOpen)).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	tickets := make([]entity.Ticket, 0, len(rows))
	for _, row := range rows {
		tickets = append(tickets, toTicket(row))
	}
	return tickets, nil
}

func toTicket(row ticketRow) entity.Ticket {
	ticket := entity.Ticket{
		ID:            row.ID,
		CustomerID:    row.CustomerID,
		CreatedBy:     row.CreatedBy,
		Title:         row.Title,
		Description:   row.Description,
		Severity:      row.Severity,
		Category:      entity.Category(row.Category),
		Status:        entity.TicketStatus(row.Status),
		AssigneeID:    row.AssigneeID,
		PriorityScore: row.PriorityScore,
		CreatedAt:     row.CreatedAt,
		CloseComment:  row.CloseComment,
	}
	if row.ClaimedAt != nil {
		ticket.ClaimedAt = *row.ClaimedAt
	}
	if row.ClosedAt != nil {
		ticket.ClosedAt = *row.ClosedAt
	}
	if row.SlaNotifiedAt != nil {
		ticket.SlaNotifiedAt = *row.SlaNotifiedAt
	}
	return ticket
}
