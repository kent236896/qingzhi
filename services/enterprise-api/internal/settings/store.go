package settings

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// Record is the stored WeKnora connection. Secrets never leave the server.
type Record struct {
	BaseURL         string
	APIKey          string
	WeKnoraTenantID string
	AdminEmail      string
	AdminPassword   string
	UpdatedBy       string
	UpdatedAt       time.Time
}

// Store persists the singleton connection settings.
type Store interface {
	Get(ctx context.Context) (Record, error)
	Save(ctx context.Context, item Record) error
}

type settingsRow struct {
	ID              int       `gorm:"column:id;primaryKey"`
	BaseURL         string    `gorm:"column:base_url"`
	APIKey          string    `gorm:"column:api_key"`
	WeKnoraTenantID string    `gorm:"column:weknora_tenant_id"`
	AdminEmail      string    `gorm:"column:admin_email"`
	AdminPassword   string    `gorm:"column:admin_password"`
	UpdatedBy       string    `gorm:"column:updated_by"`
	UpdatedAt       time.Time `gorm:"column:updated_at"`
}

func (settingsRow) TableName() string { return "sys_weknora_settings" }

// GormStore persists settings in PostgreSQL.
type GormStore struct{ db *gorm.DB }

func NewGormStore(db *gorm.DB) *GormStore { return &GormStore{db: db} }

func (s *GormStore) Get(ctx context.Context) (Record, error) {
	var row settingsRow
	err := s.db.WithContext(ctx).First(&row, "id = ?", 1).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Record{}, nil
	}
	if err != nil {
		return Record{}, err
	}
	return Record{
		BaseURL: row.BaseURL, APIKey: row.APIKey, WeKnoraTenantID: row.WeKnoraTenantID,
		AdminEmail: row.AdminEmail, AdminPassword: row.AdminPassword,
		UpdatedBy: row.UpdatedBy, UpdatedAt: row.UpdatedAt,
	}, nil
}

func (s *GormStore) Save(ctx context.Context, item Record) error {
	row := settingsRow{
		ID: 1, BaseURL: item.BaseURL, APIKey: item.APIKey, WeKnoraTenantID: item.WeKnoraTenantID,
		AdminEmail: item.AdminEmail, AdminPassword: item.AdminPassword,
		UpdatedBy: item.UpdatedBy, UpdatedAt: item.UpdatedAt,
	}
	return s.db.WithContext(ctx).Save(&row).Error
}
