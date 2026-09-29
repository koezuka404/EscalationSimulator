package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"escalator/domain"
)

type customerRow struct {
	ID         string `gorm:"primaryKey"`
	Name       string
	Plan       string
	SLAMinutes int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (customerRow) TableName() string { return "customers" }

type CustomerRepository struct {
	db *gorm.DB
}

func NewCustomerRepository(db *gorm.DB) *CustomerRepository {
	return &CustomerRepository{db: db}
}

func MigrateCustomers(db *gorm.DB) error {
	return db.AutoMigrate(&customerRow{})
}

func (r *CustomerRepository) Save(ctx context.Context, customer domain.Customer) (domain.Customer, error) {
	if customer.ID == "" {
		customer.ID = uuid.NewString()
	}
	row := toRow(customer)
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return domain.Customer{}, err
	}
	return toDomain(row), nil
}

func (r *CustomerRepository) FindByID(ctx context.Context, id string) (domain.Customer, error) {
	var row customerRow
	err := r.db.WithContext(ctx).First(&row, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.Customer{}, domain.ErrCustomerNotFound
	}
	if err != nil {
		return domain.Customer{}, err
	}
	return toDomain(row), nil
}

func (r *CustomerRepository) List(ctx context.Context) ([]domain.Customer, error) {
	var rows []customerRow
	if err := r.db.WithContext(ctx).Order("created_at, id").Find(&rows).Error; err != nil {
		return nil, err
	}
	customers := make([]domain.Customer, 0, len(rows))
	for _, row := range rows {
		customers = append(customers, toDomain(row))
	}
	return customers, nil
}

func (r *CustomerRepository) Update(ctx context.Context, customer domain.Customer) error {
	result := r.db.WithContext(ctx).Model(&customerRow{}).Where("id = ?", customer.ID).Updates(map[string]any{
		"name":        customer.Name,
		"plan":        string(customer.Plan),
		"sla_minutes": customer.SLAMinutes,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrCustomerNotFound
	}
	return nil
}

func toRow(customer domain.Customer) customerRow {
	return customerRow{
		ID:         customer.ID,
		Name:       customer.Name,
		Plan:       string(customer.Plan),
		SLAMinutes: customer.SLAMinutes,
	}
}

func toDomain(row customerRow) domain.Customer {
	return domain.Customer{
		ID:         row.ID,
		Name:       row.Name,
		Plan:       domain.Plan(row.Plan),
		SLAMinutes: row.SLAMinutes,
	}
}
