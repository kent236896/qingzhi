package knowledge

import (
	"context"
	"time"
)

// Base is a local knowledge base. Callers see the local id, not the remote id.
type Base struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	Description   string     `json:"description"`
	RemoteID      string     `json:"-"`
	ScopeType     string     `json:"scope_type"`
	ScopeID       string     `json:"scope_id"`
	ScopeName     string     `json:"scope_name,omitempty"`
	WikiEnabled   bool       `json:"wiki_enabled"`
	Status        int        `json:"status"`
	CreatorID     string     `json:"creator_id"`
	TenantIDs     []string   `json:"tenant_ids"`
	ShareUserIDs  []string   `json:"share_user_ids,omitempty"`
	ShareOrgIDs   []string   `json:"share_org_ids,omitempty"`
	ShareRoleIDs  []string   `json:"share_role_ids,omitempty"`
	WikiWaitStart *time.Time `json:"-"`
	Creator       string     `json:"creator"`
	CreateTime    time.Time  `json:"create_time"`
	Updater       string     `json:"updater"`
	UpdateTime    time.Time  `json:"update_time"`
	Deleted       bool       `json:"deleted"`
}

// Shares bundles share targets for access checks.
func (b Base) Shares() Shares {
	return Shares{UserIDs: b.ShareUserIDs, OrgIDs: b.ShareOrgIDs, RoleIDs: b.ShareRoleIDs}
}

// File is one uploaded document.
type File struct {
	ID          string    `json:"id"`
	BaseID      string    `json:"knowledge_base_id"`
	RemoteID    string    `json:"-"`
	FileName    string    `json:"file_name"`
	FolderPath  string    `json:"folder_path"`
	FileSize    int64     `json:"file_size"`
	ParseStatus string    `json:"parse_status"`
	ErrorMsg    string    `json:"error_msg"`
	CreatorID   string    `json:"creator_id"`
	CreateTime  time.Time `json:"create_time"`
	UpdateTime  time.Time `json:"update_time"`
	Deleted     bool      `json:"deleted"`
}

// UploadResult is the outcome of one file in a batch.
type UploadResult struct {
	FileName string `json:"file_name"`
	Status   string `json:"status"`
	FileID   string `json:"file_id,omitempty"`
	RemoteID string `json:"-"`
	Error    string `json:"error,omitempty"`
}

// Page is a paginated collection.
type Page[T any] struct {
	Items    []T   `json:"items"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

// BoundModel is the scene default shown when creating a knowledge base.
type BoundModel struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
}

// KnowledgeModels are the defaults used by a new knowledge base.
type KnowledgeModels struct {
	Embedding     BoundModel `json:"embedding"`
	Summary       BoundModel `json:"summary"`
	WikiSynthesis BoundModel `json:"wiki_synthesis"`
}

// ModelDirectory supplies scene default models. Env values are only a first-start fallback.
type ModelDirectory interface {
	KnowledgeModels(ctx context.Context) (KnowledgeModels, error)
}

// Limits tells the browser which uploads to accept.
type Limits struct {
	MaxFileMB     int      `json:"max_file_mb"`
	MaxFiles      int      `json:"max_files"`
	ZipMaxTotalMB int      `json:"zip_max_total_mb"`
	ZipMaxEntries int      `json:"zip_max_entries"`
	AllowedExts   []string `json:"allowed_exts"`
}

// WikiStatus is the label shown above the wiki tab.
type WikiStatus struct {
	Status      string     `json:"status"`
	PageCount   int        `json:"page_count"`
	DocTotal    int        `json:"doc_total"`
	DocTerminal int        `json:"doc_terminal"`
	UpdatedAt   *time.Time `json:"updated_at"`
}
