package catalog

import (
	"time"

	"qingzhi/enterprise-api/internal/scoped"
)

// Item is a platform catalog row for agents or MCP toolbox entries.
type Item struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	RemoteID    string    `json:"-"`
	ScopeType   string    `json:"scope_type"`
	ScopeID     string    `json:"scope_id"`
	Status      int       `json:"status"`
	CreatorID   string    `json:"creator_id"`
	TenantIDs   []string  `json:"tenant_ids"`
	ShareUserIDs []string `json:"share_user_ids,omitempty"`
	TransportType string  `json:"transport_type,omitempty"`
	Creator     string    `json:"creator"`
	Updater     string    `json:"updater"`
	CreateTime  time.Time `json:"create_time"`
	UpdateTime  time.Time `json:"update_time"`
	Writable    bool      `json:"writable,omitempty"`
}

func (i Item) Resource() scoped.Resource {
	return scoped.Resource{
		ScopeType: i.ScopeType,
		ScopeID:   i.ScopeID,
		CreatorID: i.CreatorID,
		TenantIDs: i.TenantIDs,
	}
}

func (i Item) Shares() scoped.Shares {
	return scoped.Shares{UserIDs: i.ShareUserIDs}
}

// CreateInput is shared create payload.
type CreateInput struct {
	Name          string
	Description   string
	ScopeType     string
	ScopeID       string
	TenantIDs     []string
	ShareUserIDs  []string
	TransportType string
	URL           string
	Config        map[string]any
}

// Actor aliases scoped.Actor for catalog services.
type Actor = scoped.Actor
