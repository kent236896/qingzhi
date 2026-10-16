package dict

import (
	"context"
	"strings"
	"time"
)

type memoryStore struct {
	types []Type
	data  []Data
}

func newMemoryStore() *memoryStore {
	return &memoryStore{}
}

func (s *memoryStore) Transaction(_ context.Context, fn func(Store) error) error {
	return fn(s)
}

func (s *memoryStore) InsertType(_ context.Context, item Type) error {
	s.types = append(s.types, cloneType(item))
	return nil
}

func (s *memoryStore) UpdateType(_ context.Context, item Type) error {
	for i := range s.types {
		if s.types[i].ID == item.ID && !s.types[i].Deleted {
			s.types[i] = cloneType(item)
			return nil
		}
	}
	return ErrNotFound
}

func (s *memoryStore) DeleteType(_ context.Context, id string, deletedAt time.Time) error {
	for i := range s.types {
		if s.types[i].ID == id && !s.types[i].Deleted {
			s.types[i].Deleted = true
			at := deletedAt.UTC()
			s.types[i].DeletedTime = &at
			s.types[i].UpdateTime = at
			return nil
		}
	}
	return ErrNotFound
}

func (s *memoryStore) FindType(_ context.Context, id string) (Type, error) {
	for _, item := range s.types {
		if item.ID == id && !item.Deleted {
			return cloneType(item), nil
		}
	}
	return Type{}, ErrNotFound
}

func (s *memoryStore) FindTypeByCode(_ context.Context, code string) (Type, error) {
	for _, item := range s.types {
		if item.Type == code && !item.Deleted {
			return cloneType(item), nil
		}
	}
	return Type{}, ErrNotFound
}

func (s *memoryStore) ListTypes(_ context.Context, keyword string, page, pageSize int) ([]Type, int64, error) {
	matched := make([]Type, 0)
	for _, item := range s.types {
		if item.Deleted || !containsFold(item.Name, item.Type, keyword) {
			continue
		}
		matched = append(matched, cloneType(item))
	}
	return pageSlice(matched, page, pageSize), int64(len(matched)), nil
}

func (s *memoryStore) InsertData(_ context.Context, item Data) error {
	s.data = append(s.data, item)
	return nil
}

func (s *memoryStore) UpdateData(_ context.Context, item Data) error {
	for i := range s.data {
		if s.data[i].ID == item.ID && !s.data[i].Deleted {
			s.data[i] = item
			return nil
		}
	}
	return ErrNotFound
}

func (s *memoryStore) DeleteData(_ context.Context, id string) error {
	for i := range s.data {
		if s.data[i].ID == id && !s.data[i].Deleted {
			s.data[i].Deleted = true
			s.data[i].UpdateTime = time.Now().UTC()
			return nil
		}
	}
	return ErrNotFound
}

func (s *memoryStore) FindData(_ context.Context, id string) (Data, error) {
	for _, item := range s.data {
		if item.ID == id && !item.Deleted {
			return item, nil
		}
	}
	return Data{}, ErrNotFound
}

func (s *memoryStore) FindDataByValue(_ context.Context, dictType, value string) (Data, error) {
	for _, item := range s.data {
		if item.DictType == dictType && item.Value == value && !item.Deleted {
			return item, nil
		}
	}
	return Data{}, ErrNotFound
}

func (s *memoryStore) ListData(_ context.Context, dictType, keyword string, page, pageSize int) ([]Data, int64, error) {
	matched := make([]Data, 0)
	for _, item := range s.data {
		if item.Deleted || item.DictType != dictType || !containsFold(item.Label, item.Value, keyword) {
			continue
		}
		matched = append(matched, item)
	}
	return pageSlice(matched, page, pageSize), int64(len(matched)), nil
}

func (s *memoryStore) ListEnabledData(_ context.Context, dictType string) ([]Data, error) {
	items := make([]Data, 0)
	for _, item := range s.data {
		if item.Deleted || item.DictType != dictType || item.Status != StatusNormal {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *memoryStore) CountData(_ context.Context, dictType string) (int64, error) {
	var count int64
	for _, item := range s.data {
		if !item.Deleted && item.DictType == dictType {
			count++
		}
	}
	return count, nil
}

func (s *memoryStore) RenameDataType(_ context.Context, from, to string) error {
	now := time.Now().UTC()
	for i := range s.data {
		if s.data[i].DictType == from && !s.data[i].Deleted {
			s.data[i].DictType = to
			s.data[i].UpdateTime = now
		}
	}
	return nil
}

func cloneType(item Type) Type {
	copied := item
	if item.DeletedTime != nil {
		at := *item.DeletedTime
		copied.DeletedTime = &at
	}
	return copied
}

func containsFold(left, right, keyword string) bool {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return true
	}
	keyword = strings.ToLower(keyword)
	return strings.Contains(strings.ToLower(left), keyword) || strings.Contains(strings.ToLower(right), keyword)
}

func pageSlice[T any](items []T, page, pageSize int) []T {
	start := (page - 1) * pageSize
	if start >= len(items) {
		return []T{}
	}
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}
