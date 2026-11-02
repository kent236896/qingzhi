package chat

import (
	"context"
	"time"
)

// SessionOwner is the platform ownership row for a WeKnora chat session.
type SessionOwner struct {
	SessionID       string
	UserID          string
	TenantID        string
	WeKnoraTenantID string
	Title           string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// SessionOwnerStore persists chat session ownership in PostgreSQL.
type SessionOwnerStore interface {
	Insert(ctx context.Context, owner SessionOwner) error
	OwnedBy(ctx context.Context, userID, sessionID string) (bool, error)
	Get(ctx context.Context, userID, sessionID string) (SessionOwner, error)
	ListByUser(ctx context.Context, userID string, page, pageSize int) ([]SessionOwner, int, error)
	UpdateTitle(ctx context.Context, userID, sessionID, title string) error
	Delete(ctx context.Context, userID, sessionID string) error
}
