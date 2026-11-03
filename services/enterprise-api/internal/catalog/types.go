package catalog

import (
	"encoding/json"
	"strings"
	"time"

	"qingzhi/enterprise-api/internal/scoped"
	"qingzhi/enterprise-api/internal/weknora"
)

// MCP auth modes (logic constants until sys_mcp_services.auth_mode migration in a later phase).
const (
	MCPAuthModeShared    = weknora.MCPAuthModeShared
	MCPAuthModeUserOAuth = weknora.MCPAuthModeUserOAuth
)

// NormalizeMCPAuthMode returns shared or user_oauth; empty defaults to shared.
func NormalizeMCPAuthMode(mode string) string {
	switch strings.TrimSpace(strings.ToLower(mode)) {
	case MCPAuthModeUserOAuth:
		return MCPAuthModeUserOAuth
	default:
		return MCPAuthModeShared
	}
}

// Item is a platform catalog row for agents or MCP toolbox entries.
type Item struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	RemoteID      string          `json:"-"`
	ScopeType     string          `json:"scope_type"`
	ScopeID       string          `json:"scope_id"`
	Status        int             `json:"status"`
	CreatorID     string          `json:"creator_id"`
	TenantIDs     []string        `json:"tenant_ids"`
	ShareUserIDs  []string        `json:"share_user_ids,omitempty"`
	TransportType string          `json:"transport_type,omitempty"`
	URL           string          `json:"url,omitempty"`
	AuthMode      string          `json:"auth_mode,omitempty"`
	Config        json.RawMessage `json:"config,omitempty"`
	Creator       string          `json:"creator"`
	Updater       string          `json:"updater"`
	CreateTime    time.Time       `json:"create_time"`
	UpdateTime    time.Time       `json:"update_time"`
	Writable      bool            `json:"writable,omitempty"`
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
	AuthMode      string
	Config        map[string]any
}

// UpdateInput updates name/description/config (and MCP url/auth) without changing scope.
type UpdateInput struct {
	Name          string
	Description   string
	ShareUserIDs  []string
	TransportType string
	URL           string
	AuthMode      string
	Config        map[string]any
}

// CopyInput optionally overrides name/scope for a copied agent; empty uses personal scope.
type CopyInput struct {
	Name         string
	ScopeType    string
	ScopeID      string
	TenantIDs    []string
	ShareUserIDs []string
}

// Actor aliases scoped.Actor for catalog services.
type Actor = scoped.Actor
