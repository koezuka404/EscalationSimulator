package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"escalator/entity"
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

type customerRepository struct {
	db *gorm.DB
}

func NewCustomerRepository(db *gorm.DB) CustomerRepository {
	return &customerRepository{db: db}
}

func MigrateCustomers(db *gorm.DB) error {
	return db.AutoMigrate(&customerRow{})
}

func (r *customerRepository) Save(ctx context.Context, customer entity.Customer) (entity.Customer, error) {
	if customer.ID == "" {
		customer.ID = uuid.NewString()
	}
	row := toRow(customer)
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return entity.Customer{}, err
	}
	return toDomain(row), nil
}

func (r *customerRepository) FindByID(ctx context.Context, id string) (entity.Customer, error) {
	var row customerRow
	err := r.db.WithContext(ctx).First(&row, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return entity.Customer{}, entity.ErrCustomerNotFound
	}
	if err != nil {
		return entity.Customer{}, err
	}
	return toDomain(row), nil
}

func (r *customerRepository) List(ctx context.Context) ([]entity.Customer, error) {
	var rows []customerRow
	if err := r.db.WithContext(ctx).Order("created_at, id").Find(&rows).Error; err != nil {
		return nil, err
	}
	customers := make([]entity.Customer, 0, len(rows))
	for _, row := range rows {
		customers = append(customers, toDomain(row))
	}
	return customers, nil
}

func (r *customerRepository) Update(ctx context.Context, customer entity.Customer) error {
	result := r.db.WithContext(ctx).Model(&customerRow{}).Where("id = ?", customer.ID).Updates(map[string]any{
		"name":        customer.Name,
		"plan":        string(customer.Plan),
		"sla_minutes": customer.SLAMinutes,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return entity.ErrCustomerNotFound
	}
	return nil
}

func toRow(customer entity.Customer) customerRow {
	return customerRow{
		ID:         customer.ID,
		Name:       customer.Name,
		Plan:       string(customer.Plan),
		SLAMinutes: customer.SLAMinutes,
	}
}

func toDomain(row customerRow) entity.Customer {
	return entity.Customer{
		ID:         row.ID,
		Name:       row.Name,
		Plan:       entity.Plan(row.Plan),
		SLAMinutes: row.SLAMinutes,
	}
}
