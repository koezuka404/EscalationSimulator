package repository

import (
	"context"
	"log"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"escalator/entity"
)

type agentStatusRow struct {
	UserID    string `gorm:"primaryKey"`
	Status    string
	UpdatedAt time.Time
}

func (agentStatusRow) TableName() string { return "agent_statuses" }

type claimRepository struct {
	db *gorm.DB
}

func NewClaimRepository(db *gorm.DB) ClaimStore {
	return &claimRepository{db: db}
}

func MigrateAgentStatuses(db *gorm.DB) error {
	return db.AutoMigrate(&agentStatusRow{})
}

//待機中の担当者に点数がいちばん高い対応待ちを渡す
func (r *claimRepository) ClaimNext(ctx context.Context, agentID string, queue TicketQueue) (entity.Ticket, error) {
	var claimed entity.Ticket
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := ensureAgentAvailable(tx, agentID); err != nil {
			return err
		}
		var status agentStatusRow
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&status, "user_id = ?", agentID).Error
		if err != nil {
			return err
		}
		if status.Status != string(entity.AgentAvailable) {
			return entity.ErrNotWaiting
		}
		var working int64
		err = tx.Model(&ticketRow{}).Where("assignee_id = ? AND status = ?", agentID, string(entity.TicketInProgress)).Count(&working).Error
		if err != nil {
			return err
		}
		if working > 0 {
			return entity.ErrAgentBusy
		}
		now := time.Now()
		for attempt := 0; attempt < 3; attempt++ {
			id, score, ok, err := popTicket(ctx, queue)
			if err != nil {
				return entity.ErrQueueUnavailable
			}
			if !ok {
				return entity.ErrQueueEmpty
			}
			result := tx.Model(&ticketRow{}).Where("id = ? AND status = ?", id, string(entity.TicketOpen)).Updates(map[string]any{
				"status":      string(entity.TicketInProgress),
				"assignee_id": agentID,
				"claimed_at":  now,
			})
			if result.Error != nil {
				putTicketBack(ctx, queue, id, score)
				return result.Error
			}
			if result.RowsAffected == 0 {
				continue
			}
			err = tx.Model(&agentStatusRow{}).Where("user_id = ?", agentID).Update("status", string(entity.AgentBusy)).Error
			if err != nil {
				putTicketBack(ctx, queue, id, score)
				return err
			}
			var row ticketRow
			if err := tx.First(&row, "id = ?", id).Error; err != nil {
				putTicketBack(ctx, queue, id, score)
				return err
			}
			claimed = toTicket(row)
			return nil
		}
		return entity.ErrQueueUnavailable
	})
	if err != nil {
		return entity.Ticket{}, err
	}
	return claimed, nil
}

func ensureAgentAvailable(tx *gorm.DB, agentID string) error {
	row := agentStatusRow{UserID: agentID, Status: string(entity.AgentAvailable)}
	return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}}, DoNothing: true}).Create(&row).Error
}

func popTicket(ctx context.Context, queue TicketQueue) (string, int, bool, error) {
	popCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return queue.PopMax(popCtx)
}

func putTicketBack(ctx context.Context, queue TicketQueue, ticketID string, score int) {
	putCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := queue.Enqueue(putCtx, ticketID, score); err != nil {
		log.Printf("待ち順へ戻せませんでした。チケットは担当者に渡していません。id=%s: %v", ticketID, err)
	}
}
