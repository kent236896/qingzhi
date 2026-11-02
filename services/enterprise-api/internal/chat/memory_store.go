package chat

import (
	"context"
	"sort"
	"sync"
	"time"
)

// MemorySessionOwnerStore is an in-memory SessionOwnerStore for unit tests.
type MemorySessionOwnerStore struct {
	mu   sync.Mutex
	byID map[string]SessionOwner
}

func NewMemorySessionOwnerStore() *MemorySessionOwnerStore {
	return &MemorySessionOwnerStore{byID: map[string]SessionOwner{}}
}

func (s *MemorySessionOwnerStore) Insert(ctx context.Context, owner SessionOwner) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.byID[owner.SessionID]; ok {
		return invalid("会话归属已存在")
	}
	now := time.Now().UTC()
	if owner.CreatedAt.IsZero() {
		owner.CreatedAt = now
	}
	if owner.UpdatedAt.IsZero() {
		owner.UpdatedAt = now
	}
	s.byID[owner.SessionID] = owner
	return nil
}

func (s *MemorySessionOwnerStore) OwnedBy(ctx context.Context, userID, sessionID string) (bool, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	owner, ok := s.byID[sessionID]
	return ok && owner.UserID == userID, nil
}

func (s *MemorySessionOwnerStore) Get(ctx context.Context, userID, sessionID string) (SessionOwner, error) {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	owner, ok := s.byID[sessionID]
	if !ok || owner.UserID != userID {
		return SessionOwner{}, ErrNotFound
	}
	return owner, nil
}

func (s *MemorySessionOwnerStore) ListByUser(ctx context.Context, userID string, page, pageSize int) ([]SessionOwner, int, error) {
	_ = ctx
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	items := make([]SessionOwner, 0)
	for _, owner := range s.byID {
		if owner.UserID == userID {
			items = append(items, owner)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].UpdatedAt.After(items[j].UpdatedAt)
	})
	total := len(items)
	start := (page - 1) * pageSize
	if start >= total {
		return []SessionOwner{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return items[start:end], total, nil
}

func (s *MemorySessionOwnerStore) UpdateTitle(ctx context.Context, userID, sessionID, title string) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	owner, ok := s.byID[sessionID]
	if !ok || owner.UserID != userID {
		return ErrNotFound
	}
	owner.Title = title
	owner.UpdatedAt = time.Now().UTC()
	s.byID[sessionID] = owner
	return nil
}

func (s *MemorySessionOwnerStore) Delete(ctx context.Context, userID, sessionID string) error {
	_ = ctx
	s.mu.Lock()
	defer s.mu.Unlock()
	owner, ok := s.byID[sessionID]
	if !ok || owner.UserID != userID {
		return ErrNotFound
	}
	delete(s.byID, sessionID)
	return nil
}
