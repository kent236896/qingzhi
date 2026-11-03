package chat

import (
	"context"

	"qingzhi/enterprise-api/internal/catalog"
)

// CatalogAgentResolver adapts catalog.Service to AgentResolver.
type CatalogAgentResolver struct {
	Catalog *catalog.Service
}

func (r CatalogAgentResolver) ResolveVisibleAgentRemoteID(
	ctx context.Context, userID, username string, tenantIDs []string, canManageAgents bool, platformAgentID string,
) (string, error) {
	if r.Catalog == nil {
		return "", notReady("智能体目录不可用")
	}
	return r.Catalog.ResolveVisibleAgentRemoteID(ctx, catalog.Actor{
		UserID: userID, Username: username, TenantIDs: tenantIDs, CanManage: canManageAgents,
	}, platformAgentID)
}
