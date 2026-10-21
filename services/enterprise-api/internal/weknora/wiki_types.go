package weknora

// WikiFolder is a directory node in the WeKnora wiki tree.
type WikiFolder struct {
	ID              string  `json:"id"`
	TenantID        int64   `json:"tenant_id,omitempty"`
	KnowledgeBaseID string  `json:"knowledge_base_id,omitempty"`
	ParentID        string  `json:"parent_id"`
	Name            string  `json:"name"`
	Path            string  `json:"path"`
	Depth           int     `json:"depth"`
	SortOrder       int     `json:"sort_order"`
	PageCount       int     `json:"page_count"`
	HasChildren     bool    `json:"has_children"`
	CreatedAt       string  `json:"created_at,omitempty"`
	UpdatedAt       string  `json:"updated_at,omitempty"`
	DeletedAt       *string `json:"deleted_at,omitempty"`
}

// WikiFolderList is the folders endpoint payload.
type WikiFolderList struct {
	ParentID string       `json:"parent_id"`
	Folders  []WikiFolder `json:"folders"`
}

// WikiPage is a full wiki page from WeKnora (raw object, no success envelope).
type WikiPage struct {
	ID              string         `json:"id"`
	TenantID        int64          `json:"tenant_id,omitempty"`
	KnowledgeBaseID string         `json:"knowledge_base_id,omitempty"`
	Slug            string         `json:"slug"`
	Title           string         `json:"title"`
	PageType        string         `json:"page_type"`
	Status          string         `json:"status,omitempty"`
	Content         string         `json:"content"`
	Summary         string         `json:"summary"`
	Aliases         []string       `json:"aliases"`
	FolderID        string         `json:"folder_id,omitempty"`
	CategoryPath    []string       `json:"category_path,omitempty"`
	WikiPath        string         `json:"wiki_path,omitempty"`
	Depth           int            `json:"depth,omitempty"`
	SourceRefs      []string       `json:"source_refs,omitempty"`
	ChunkRefs       []string       `json:"chunk_refs,omitempty"`
	InLinks         []string       `json:"in_links"`
	OutLinks        []string       `json:"out_links"`
	PageMetadata    map[string]any `json:"page_metadata,omitempty"`
	Version         int            `json:"version"`
	LastEditSource  string         `json:"last_edit_source"`
	CreatedAt       string         `json:"created_at,omitempty"`
	UpdatedAt       string         `json:"updated_at,omitempty"`
	DeletedAt       *string        `json:"deleted_at,omitempty"`
}

// WikiPageList is a paginated page list.
type WikiPageList struct {
	Pages     []WikiPage `json:"pages"`
	Total     int64      `json:"total"`
	Page      int        `json:"page"`
	PageSize  int        `json:"page_size"`
	TotalPage int        `json:"total_pages"`
}

// WikiIndexItem is one entry under an index group.
type WikiIndexItem struct {
	Slug         string   `json:"slug"`
	Title        string   `json:"title"`
	Summary      string   `json:"summary"`
	CategoryPath []string `json:"category_path,omitempty"`
	WikiPath     string   `json:"wiki_path,omitempty"`
	Depth        int      `json:"depth,omitempty"`
}

// WikiIndexGroup is one page-type section on the index page.
type WikiIndexGroup struct {
	Type       string          `json:"type"`
	Total      int             `json:"total"`
	Items      []WikiIndexItem `json:"items"`
	NextCursor string          `json:"next_cursor,omitempty"`
}

// WikiIndex is the wiki index payload.
type WikiIndex struct {
	Intro   string           `json:"intro"`
	Version int              `json:"version"`
	Groups  []WikiIndexGroup `json:"groups"`
}

// WikiRevision is one historical version metadata row.
type WikiRevision struct {
	Version    int    `json:"version"`
	EditSource string `json:"edit_source"`
	Editor     string `json:"editor,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
}

// WikiRevisionList is the revisions endpoint payload.
type WikiRevisionList struct {
	Revisions      []WikiRevision `json:"revisions"`
	Total          int            `json:"total"`
	CurrentVersion int            `json:"current_version"`
}

// WikiTitle is a lightweight slug→title entry for reverse-link resolution.
type WikiTitle struct {
	Slug     string `json:"slug"`
	Title    string `json:"title"`
	PageType string `json:"page_type,omitempty"`
}
