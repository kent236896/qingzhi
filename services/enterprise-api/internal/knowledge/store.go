package knowledge

import "context"

// Store persists knowledge bases and files.
type Store interface {
	Transaction(ctx context.Context, fn func(Store) error) error
	InsertBase(ctx context.Context, item Base) error
	UpdateBase(ctx context.Context, item Base) error
	DeleteBase(ctx context.Context, id string) error
	FindBase(ctx context.Context, id string) (Base, error)
	ListBases(ctx context.Context) ([]Base, error)
	ReplaceTenants(ctx context.Context, baseID string, tenantIDs []string) error
	ReplaceShares(ctx context.Context, baseID string, userIDs []string) error
	ReplaceShareScopes(ctx context.Context, baseID string, orgIDs, roleIDs []string) error
	InsertFile(ctx context.Context, item File) error
	UpdateFile(ctx context.Context, item File) error
	ListFiles(ctx context.Context, baseID string) ([]File, error)
	FindFiles(ctx context.Context, baseID string, ids []string) ([]File, error)
	DeleteFiles(ctx context.Context, baseID string, ids []string) error
}
