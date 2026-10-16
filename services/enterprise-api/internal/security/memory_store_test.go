package security

import (
	"context"
	"strings"
	"time"
)

type memoryStore struct {
	logins      []LoginLog
	operates    []OperateLog
	permissions []PermissionLog
	policy      Policy
	hasPolicy   bool
}

func newMemoryStore() *memoryStore {
	return &memoryStore{}
}

func (s *memoryStore) Transaction(_ context.Context, fn func(Store) error) error {
	return fn(s)
}

func (s *memoryStore) InsertLogin(_ context.Context, item LoginLog) error {
	s.logins = append(s.logins, item)
	return nil
}

func (s *memoryStore) ListLogins(_ context.Context, keyword string, result *int, page, pageSize int) ([]LoginLog, int64, error) {
	matched := make([]LoginLog, 0)
	for i := len(s.logins) - 1; i >= 0; i-- {
		item := s.logins[i]
		if result != nil && item.Result != *result {
			continue
		}
		if !containsFold(keyword, item.Username, item.Message, item.IP) {
			continue
		}
		matched = append(matched, item)
	}
	return pageSlice(matched, page, pageSize), int64(len(matched)), nil
}

func (s *memoryStore) ListLoginAttempts(_ context.Context, username string, since time.Time) ([]LoginLog, error) {
	items := make([]LoginLog, 0)
	for _, item := range s.logins {
		if item.Username == username && !item.CreateTime.Before(since) {
			items = append(items, item)
		}
	}
	return items, nil
}

func (s *memoryStore) InsertOperate(_ context.Context, item OperateLog) error {
	s.operates = append(s.operates, item)
	return nil
}

func (s *memoryStore) ListOperates(_ context.Context, keyword string, page, pageSize int) ([]OperateLog, int64, error) {
	matched := make([]OperateLog, 0)
	for i := len(s.operates) - 1; i >= 0; i-- {
		item := s.operates[i]
		if containsFold(keyword, item.Username, item.Module, item.Action, item.RequestURL) {
			matched = append(matched, item)
		}
	}
	return pageSlice(matched, page, pageSize), int64(len(matched)), nil
}

func (s *memoryStore) InsertPermission(_ context.Context, item PermissionLog) error {
	s.permissions = append(s.permissions, item)
	return nil
}

func (s *memoryStore) ListPermissions(_ context.Context, keyword string, page, pageSize int) ([]PermissionLog, int64, error) {
	matched := make([]PermissionLog, 0)
	for i := len(s.permissions) - 1; i >= 0; i-- {
		item := s.permissions[i]
		if containsFold(keyword, item.OperatorName, item.RoleName, item.Summary) {
			matched = append(matched, item)
		}
	}
	return pageSlice(matched, page, pageSize), int64(len(matched)), nil
}

func (s *memoryStore) FindPolicy(context.Context) (Policy, error) {
	if !s.hasPolicy {
		return DefaultPolicy(), nil
	}
	return s.policy, nil
}

func (s *memoryStore) SavePolicy(_ context.Context, item Policy) error {
	s.policy = item
	s.hasPolicy = true
	return nil
}

func containsFold(keyword string, values ...string) bool {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return true
	}
	keyword = strings.ToLower(keyword)
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), keyword) {
			return true
		}
	}
	return false
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
