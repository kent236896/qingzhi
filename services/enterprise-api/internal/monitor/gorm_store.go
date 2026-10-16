package monitor

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// GormStore persists alerts in PostgreSQL.
type GormStore struct {
	db *gorm.DB
}

// NewGormStore returns a PostgreSQL alert store.
func NewGormStore(db *gorm.DB) *GormStore {
	return &GormStore{db: db}
}

// Sync upserts firing alerts and resolves those that cleared.
func (s *GormStore) Sync(ctx context.Context, desired []Draft) ([]Alert, error) {
	var open []Alert
	if err := s.db.WithContext(ctx).Where("status = ?", "firing").Find(&open).Error; err != nil {
		return nil, err
	}
	now := time.Now()
	seen := map[string]bool{}
	for _, draft := range desired {
		key := draft.Kind + "\x00" + draft.Target
		seen[key] = true
		var existing *Alert
		for i := range open {
			if open[i].Kind == draft.Kind && open[i].Target == draft.Target {
				existing = &open[i]
				break
			}
		}
		if existing != nil {
			if err := s.db.WithContext(ctx).Model(&Alert{}).Where("id = ?", existing.ID).Updates(map[string]any{
				"level": draft.Level, "title": draft.Title, "message": draft.Message, "last_time": now,
			}).Error; err != nil {
				return nil, err
			}
			continue
		}
		row := Alert{
			ID: newID(), Kind: draft.Kind, Target: draft.Target, Level: draft.Level,
			Title: draft.Title, Message: draft.Message, Status: "firing",
			FirstTime: now, LastTime: now,
		}
		if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
			return nil, err
		}
	}
	for _, item := range open {
		if seen[item.Kind+"\x00"+item.Target] {
			continue
		}
		if err := s.db.WithContext(ctx).Model(&Alert{}).Where("id = ?", item.ID).Updates(map[string]any{
			"status": "resolved", "resolved_time": now, "last_time": now,
		}).Error; err != nil {
			return nil, err
		}
	}
	var firing []Alert
	if err := s.db.WithContext(ctx).Where("status = ?", "firing").Order("first_time asc").Find(&firing).Error; err != nil {
		return nil, err
	}
	if firing == nil {
		firing = []Alert{}
	}
	return firing, nil
}

// List returns alert history, newest first.
func (s *GormStore) List(ctx context.Context, page, pageSize int, status string) (Page[Alert], error) {
	base := s.db.WithContext(ctx).Model(&Alert{})
	if status != "" {
		base = base.Where("status = ?", status)
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return Page[Alert]{}, err
	}
	var items []Alert
	if err := base.Order("last_time desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		return Page[Alert]{}, err
	}
	if items == nil {
		items = []Alert{}
	}
	return Page[Alert]{Items: items, Page: page, PageSize: pageSize, Total: total}, nil
}
