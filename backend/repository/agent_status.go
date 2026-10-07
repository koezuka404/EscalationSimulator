package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"escalator/entity"
)

type agentStatusRepository struct {
	db *gorm.DB
}

func NewAgentStatusRepository(db *gorm.DB) AgentStatusRepository {
	return &agentStatusRepository{db: db}
}

//担当者を待機中にする対応中のチケットがあるときは変えない
func (r *agentStatusRepository) GoAvailable(ctx context.Context, agentID string, now time.Time) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := ensureAgentAvailable(tx, agentID); err != nil {
			return err
		}
		if err := lockAgentStatus(tx, agentID); err != nil {
			return err
		}
		var working int64
		err := tx.Model(&ticketRow{}).Where("assignee_id = ? AND status = ?", agentID, string(entity.TicketInProgress)).Count(&working).Error
		if err != nil {
			return err
		}
		if working > 0 {
			return entity.ErrCannotBecomeAvailable
		}
		return setAgentStatus(tx, agentID, entity.AgentAvailable, now)
	})
}

//担当者を離席にし、対応中のチケットを待ちに戻す
func (r *agentStatusRepository) GoOffline(ctx context.Context, agentID string, now time.Time, scoreOf func(entity.Ticket) (int, error)) ([]entity.Ticket, error) {
	var released []entity.Ticket
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := ensureAgentAvailable(tx, agentID); err != nil {
			return err
		}
		if err := lockAgentStatus(tx, agentID); err != nil {
			return err
		}
		for {
			var row ticketRow
			err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("assignee_id = ? AND status = ?", agentID, string(entity.TicketInProgress)).
				First(&row).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				break
			}
			if err != nil {
				return err
			}
			score, err := scoreOf(toTicket(row))
			if err != nil {
				return err
			}
			result := tx.Model(&ticketRow{}).Where("id = ? AND status = ?", row.ID, string(entity.TicketInProgress)).Updates(map[string]any{
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
				continue
			}
			if err := tx.First(&row, "id = ?", row.ID).Error; err != nil {
				return err
			}
			released = append(released, toTicket(row))
		}
		return setAgentStatus(tx, agentID, entity.AgentOffline, now)
	})
	if err != nil {
		return nil, err
	}
	if released == nil {
		released = []entity.Ticket{}
	}
	return released, nil
}

func lockAgentStatus(tx *gorm.DB, agentID string) error {
	var status agentStatusRow
	return tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&status, "user_id = ?", agentID).Error
}

func setAgentStatus(tx *gorm.DB, agentID string, status entity.AgentAvailability, now time.Time) error {
	return tx.Model(&agentStatusRow{}).Where("user_id = ?", agentID).Updates(map[string]any{
		"status":     string(status),
		"updated_at": now,
	}).Error
}
