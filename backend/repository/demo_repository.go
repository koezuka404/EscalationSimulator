package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"escalator/entity"
)

type demoRunRow struct {
	ID             string `gorm:"primaryKey"`
	StartedBy      string
	TotalCount     int
	IntervalSec    int
	Distribution   string
	Status         string `gorm:"index"`
	GeneratedCount int
	StartedAt      time.Time
	StoppedAt      *time.Time
}

func (demoRunRow) TableName() string { return "demo_runs" }

type demoRepository struct {
	db *gorm.DB
}

func NewDemoRepository(db *gorm.DB) DemoRepository {
	return &demoRepository{db: db}
}

func MigrateDemoRuns(db *gorm.DB) error {
	if err := db.AutoMigrate(&demoRunRow{}); err != nil {
		return err
	}
	return db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS demo_runs_one_running ON demo_runs (status) WHERE status = 'running'").Error
}

//実行中のデモを1件だけ作り、すでに実行中なら拒否する
func (r *demoRepository) Start(ctx context.Context, run entity.DemoRun) (entity.DemoRun, error) {
	if run.ID == "" {
		run.ID = uuid.NewString()
	}
	row := demoRunRow{
		ID:           run.ID,
		StartedBy:    run.StartedBy,
		TotalCount:   run.TotalCount,
		IntervalSec:  run.IntervalSec,
		Distribution: run.Distribution,
		Status:       entity.DemoRunning,
		StartedAt:    run.StartedAt,
	}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return entity.DemoRun{}, entity.ErrDemoRunning
		}
		return entity.DemoRun{}, err
	}
	return demoFromRow(row), nil
}

//作った件数を1つ増やす実行中でなければ増やさない
func (r *demoRepository) AddGenerated(ctx context.Context, id string) (entity.DemoRun, error) {
	var row demoRunRow
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "id = ?", id).Error; err != nil {
			return err
		}
		if row.Status != entity.DemoRunning {
			return entity.ErrDemoNotRunning
		}
		row.GeneratedCount++
		return tx.Model(&demoRunRow{}).Where("id = ?", id).Update("generated_count", row.GeneratedCount).Error
	})
	if err != nil {
		return entity.DemoRun{}, err
	}
	return demoFromRow(row), nil
}

//実行中のデモを完了か停止にするすでに止まっていれば変えない
func (r *demoRepository) Finish(ctx context.Context, id, status string, at time.Time) (entity.DemoRun, bool, error) {
	result := r.db.WithContext(ctx).Model(&demoRunRow{}).
		Where("id = ? AND status = ?", id, entity.DemoRunning).
		Updates(map[string]any{
			"status":     status,
			"stopped_at": at,
		})
	if result.Error != nil {
		return entity.DemoRun{}, false, result.Error
	}
	var row demoRunRow
	if err := r.db.WithContext(ctx).First(&row, "id = ?", id).Error; err != nil {
		return entity.DemoRun{}, false, err
	}
	return demoFromRow(row), result.RowsAffected > 0, nil
}

//実行中のデモを止める実行中が無ければエラーにする
func (r *demoRepository) StopRunning(ctx context.Context, at time.Time) (entity.DemoRun, error) {
	var row demoRunRow
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("status = ?", entity.DemoRunning).First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity.ErrDemoNotRunning
		}
		if err != nil {
			return err
		}
		row.Status = entity.DemoStopped
		row.StoppedAt = &at
		return tx.Model(&demoRunRow{}).Where("id = ?", row.ID).Updates(map[string]any{
			"status":     entity.DemoStopped,
			"stopped_at": at,
		}).Error
	})
	if err != nil {
		return entity.DemoRun{}, err
	}
	return demoFromRow(row), nil
}

//起動時に、途中で残った実行中を停止にする
func (r *demoRepository) CloseInterrupted(ctx context.Context, at time.Time) error {
	return r.db.WithContext(ctx).Model(&demoRunRow{}).
		Where("status = ?", entity.DemoRunning).
		Updates(map[string]any{
			"status":     entity.DemoStopped,
			"stopped_at": at,
		}).Error
}

func demoFromRow(row demoRunRow) entity.DemoRun {
	run := entity.DemoRun{
		ID:             row.ID,
		StartedBy:      row.StartedBy,
		TotalCount:     row.TotalCount,
		IntervalSec:    row.IntervalSec,
		Distribution:   row.Distribution,
		Status:         row.Status,
		GeneratedCount: row.GeneratedCount,
		StartedAt:      row.StartedAt,
	}
	if row.StoppedAt != nil {
		run.StoppedAt = *row.StoppedAt
	}
	return run
}
