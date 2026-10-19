package modelcfg

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

// Binding is one scene default model.
type Binding struct {
	Role        string    `json:"role"`
	ModelID     string    `json:"model_id"`
	ModelName   string    `json:"model_name"`
	DisplayName string    `json:"display_name"`
	UpdatedBy   string    `json:"updated_by"`
	UpdatedAt   time.Time `json:"updated_at"`
	Warning     string    `json:"warning,omitempty"`
}

// Store persists scene bindings. The model itself stays in WeKnora.
type Store interface {
	List(ctx context.Context) ([]Binding, error)
	Find(ctx context.Context, role string) (Binding, error)
	Upsert(ctx context.Context, item Binding) error
}

type bindingRow struct {
	Role      string    `gorm:"column:role;primaryKey"`
	ModelID   string    `gorm:"column:model_id"`
	UpdatedBy string    `gorm:"column:updated_by"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

func (bindingRow) TableName() string { return "sys_weknora_model_bindings" }

// GormStore persists bindings in PostgreSQL.
type GormStore struct{ db *gorm.DB }

func NewGormStore(db *gorm.DB) *GormStore { return &GormStore{db: db} }

func (s *GormStore) List(ctx context.Context) ([]Binding, error) {
	var rows []bindingRow
	if err := s.db.WithContext(ctx).Order("role").Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]Binding, 0, len(rows))
	for _, row := range rows {
		items = append(items, Binding{Role: row.Role, ModelID: row.ModelID, UpdatedBy: row.UpdatedBy, UpdatedAt: row.UpdatedAt})
	}
	return items, nil
}

func (s *GormStore) Find(ctx context.Context, role string) (Binding, error) {
	var row bindingRow
	err := s.db.WithContext(ctx).First(&row, "role = ?", role).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Binding{}, ErrNotFound
	}
	if err != nil {
		return Binding{}, err
	}
	return Binding{Role: row.Role, ModelID: row.ModelID, UpdatedBy: row.UpdatedBy, UpdatedAt: row.UpdatedAt}, nil
}

func (s *GormStore) Upsert(ctx context.Context, item Binding) error {
	row := bindingRow{Role: item.Role, ModelID: item.ModelID, UpdatedBy: item.UpdatedBy, UpdatedAt: item.UpdatedAt}
	return s.db.WithContext(ctx).Save(&row).Error
}
