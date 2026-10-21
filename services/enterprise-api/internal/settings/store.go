package settings

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// Record is the stored WeKnora connection. APIKey never leaves the server.
type Record struct {
	BaseURL   string
	APIKey    string
	UpdatedBy string
	UpdatedAt time.Time
}

// Store persists the singleton connection settings.
type Store interface {
	Get(ctx context.Context) (Record, error)
	Save(ctx context.Context, item Record) error
}

type settingsRow struct {
	ID        int       `gorm:"column:id;primaryKey"`
	BaseURL   string    `gorm:"column:base_url"`
	APIKey    string    `gorm:"column:api_key"`
	UpdatedBy string    `gorm:"column:updated_by"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
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
	return Record{BaseURL: row.BaseURL, APIKey: row.APIKey, UpdatedBy: row.UpdatedBy, UpdatedAt: row.UpdatedAt}, nil
}

func (s *GormStore) Save(ctx context.Context, item Record) error {
	row := settingsRow{ID: 1, BaseURL: item.BaseURL, APIKey: item.APIKey, UpdatedBy: item.UpdatedBy, UpdatedAt: item.UpdatedAt}
	return s.db.WithContext(ctx).Save(&row).Error
}
