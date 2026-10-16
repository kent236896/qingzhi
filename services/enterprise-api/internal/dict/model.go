package dict

import "time"

const (
	// StatusNormal is an enabled dictionary type or item.
	StatusNormal = 0
	// StatusDisabled is a disabled dictionary type or item.
	StatusDisabled = 1
)

// Type is one dictionary category. Type code is unique among records that are not deleted.
type Type struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Type        string     `json:"type"`
	Status      int        `json:"status"`
	Remark      string     `json:"remark"`
	Creator     string     `json:"creator"`
	CreateTime  time.Time  `json:"create_time"`
	Updater     string     `json:"updater"`
	UpdateTime  time.Time  `json:"update_time"`
	Deleted     bool       `json:"deleted"`
	DeletedTime *time.Time `json:"deleted_time"`
}

// Data is one label/value pair under a dictionary type.
type Data struct {
	ID         string    `json:"id"`
	Sort       int       `json:"sort"`
	Label      string    `json:"label"`
	Value      string    `json:"value"`
	DictType   string    `json:"dict_type"`
	Status     int       `json:"status"`
	ColorType  string    `json:"color_type"`
	CSSClass   string    `json:"css_class"`
	Remark     string    `json:"remark"`
	Creator    string    `json:"creator"`
	CreateTime time.Time `json:"create_time"`
	Updater    string    `json:"updater"`
	UpdateTime time.Time `json:"update_time"`
	Deleted    bool      `json:"deleted"`
}

// Page is a paginated collection.
type Page[T any] struct {
	Items    []T   `json:"items"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

// ListQuery is the shared list request.
type ListQuery struct {
	Page     int
	PageSize int
	Keyword  string
}
