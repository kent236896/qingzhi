package security

import (
	"context"
	"time"
)

// Store persists security logs and the policy row.
type Store interface {
	Transaction(ctx context.Context, fn func(Store) error) error
	InsertLogin(ctx context.Context, item LoginLog) error
	ListLogins(ctx context.Context, keyword string, result *int, page, pageSize int) ([]LoginLog, int64, error)
	ListLoginAttempts(ctx context.Context, username string, since time.Time) ([]LoginLog, error)
	InsertOperate(ctx context.Context, item OperateLog) error
	ListOperates(ctx context.Context, keyword string, page, pageSize int) ([]OperateLog, int64, error)
	InsertPermission(ctx context.Context, item PermissionLog) error
	ListPermissions(ctx context.Context, keyword string, page, pageSize int) ([]PermissionLog, int64, error)
	FindPolicy(ctx context.Context) (Policy, error)
	SavePolicy(ctx context.Context, item Policy) error
}
