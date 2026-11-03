package catalog

import (
	"context"
	"errors"

	"qingzhi/enterprise-api/internal/knowledge"
)

// KnowledgeServiceMapper adapts knowledge.Service for agent config KB binding.
type KnowledgeServiceMapper struct {
	Service *knowledge.Service
}

func (m KnowledgeServiceMapper) MapVisibleRemoteIDs(
	ctx context.Context, userID, username string, tenantIDs []string, platformIDs []string,
) ([]string, error) {
	if m.Service == nil {
		return nil, notReady("知识库服务不可用")
	}
	return m.Service.ResolveRemoteKnowledgeBaseIDs(ctx, knowledge.Actor{
		UserID: userID, Username: username, TenantIDs: tenantIDs,
	}, platformIDs)
}

func mapKnowledgeErr(err error) error {
	if err == nil {
		return nil
	}
	var domain *knowledge.Error
	if errors.As(err, &domain) {
		switch {
		case errors.Is(domain.Kind, knowledge.ErrNotFound):
			return notFound(domain.Message)
		case errors.Is(domain.Kind, knowledge.ErrInvalid):
			return invalid(domain.Message)
		case errors.Is(domain.Kind, knowledge.ErrForbidden):
			return forbidden(domain.Message)
		case errors.Is(domain.Kind, knowledge.ErrNotReady):
			return notReady(domain.Message)
		}
	}
	return err
}
