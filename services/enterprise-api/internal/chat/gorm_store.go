package chat

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

type gormSessionOwner struct {
	SessionID       string    `gorm:"column:weknora_session_id;primaryKey"`
	UserID          string    `gorm:"column:user_id"`
	TenantID        string    `gorm:"column:tenant_id"`
	WeKnoraTenantID string    `gorm:"column:weknora_tenant_id"`
	Title           string    `gorm:"column:title"`
	CreatedAt       time.Time `gorm:"column:created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at"`
}

func (gormSessionOwner) TableName() string { return "chat_session_owners" }

// GormSessionOwnerStore implements SessionOwnerStore with PostgreSQL.
type GormSessionOwnerStore struct {
	db *gorm.DB
}

func NewGormSessionOwnerStore(db *gorm.DB) *GormSessionOwnerStore {
	return &GormSessionOwnerStore{db: db}
}

func (s *GormSessionOwnerStore) Insert(ctx context.Context, owner SessionOwner) error {
	now := time.Now().UTC()
	if owner.CreatedAt.IsZero() {
		owner.CreatedAt = now
	}
	if owner.UpdatedAt.IsZero() {
		owner.UpdatedAt = now
	}
	row := gormSessionOwner{
		SessionID: owner.SessionID, UserID: owner.UserID, TenantID: owner.TenantID,
		WeKnoraTenantID: owner.WeKnoraTenantID, Title: owner.Title,
		CreatedAt: owner.CreatedAt, UpdatedAt: owner.UpdatedAt,
	}
	return s.db.WithContext(ctx).Create(&row).Error
}

func (s *GormSessionOwnerStore) OwnedBy(ctx context.Context, userID, sessionID string) (bool, error) {
	var n int64
	err := s.db.WithContext(ctx).Model(&gormSessionOwner{}).
		Where("weknora_session_id = ? AND user_id = ?", sessionID, userID).
		Count(&n).Error
	return n > 0, err
}

func (s *GormSessionOwnerStore) Get(ctx context.Context, userID, sessionID string) (SessionOwner, error) {
	var row gormSessionOwner
	err := s.db.WithContext(ctx).
		Where("weknora_session_id = ? AND user_id = ?", sessionID, userID).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return SessionOwner{}, ErrNotFound
	}
	if err != nil {
		return SessionOwner{}, err
	}
	return mapOwner(row), nil
}

func (s *GormSessionOwnerStore) ListByUser(ctx context.Context, userID string, page, pageSize int) ([]SessionOwner, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	var total int64
	q := s.db.WithContext(ctx).Model(&gormSessionOwner{}).Where("user_id = ?", userID)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []gormSessionOwner
	err := q.Order("updated_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	out := make([]SessionOwner, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapOwner(row))
	}
	return out, int(total), nil
}

func (s *GormSessionOwnerStore) UpdateTitle(ctx context.Context, userID, sessionID, title string) error {
	res := s.db.WithContext(ctx).Model(&gormSessionOwner{}).
		Where("weknora_session_id = ? AND user_id = ?", sessionID, userID).
		Updates(map[string]any{"title": title, "updated_at": time.Now().UTC()})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *GormSessionOwnerStore) Delete(ctx context.Context, userID, sessionID string) error {
	res := s.db.WithContext(ctx).
		Where("weknora_session_id = ? AND user_id = ?", sessionID, userID).
		Delete(&gormSessionOwner{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func mapOwner(row gormSessionOwner) SessionOwner {
	return SessionOwner{
		SessionID: row.SessionID, UserID: row.UserID, TenantID: row.TenantID,
		WeKnoraTenantID: row.WeKnoraTenantID, Title: row.Title,
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}
