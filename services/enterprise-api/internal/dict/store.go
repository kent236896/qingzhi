package dict

import (
	"context"
	"time"
)

// Store persists dictionary types and data.
type Store interface {
	Transaction(ctx context.Context, fn func(Store) error) error
	InsertType(ctx context.Context, item Type) error
	UpdateType(ctx context.Context, item Type) error
	DeleteType(ctx context.Context, id string, deletedAt time.Time) error
	FindType(ctx context.Context, id string) (Type, error)
	FindTypeByCode(ctx context.Context, code string) (Type, error)
	ListTypes(ctx context.Context, keyword string, page, pageSize int) ([]Type, int64, error)
	InsertData(ctx context.Context, item Data) error
	UpdateData(ctx context.Context, item Data) error
	DeleteData(ctx context.Context, id string) error
	FindData(ctx context.Context, id string) (Data, error)
	FindDataByValue(ctx context.Context, dictType, value string) (Data, error)
	ListData(ctx context.Context, dictType, keyword string, page, pageSize int) ([]Data, int64, error)
	ListEnabledData(ctx context.Context, dictType string) ([]Data, error)
	CountData(ctx context.Context, dictType string) (int64, error)
	RenameDataType(ctx context.Context, from, to string) error
}
