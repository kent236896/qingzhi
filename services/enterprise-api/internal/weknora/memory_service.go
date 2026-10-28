package weknora

import (
	"context"
	"fmt"
	"strings"
	"time"

	"qingzhi/enterprise-api/internal/auth"
)

// MemoryRemote reads and writes workspace memory configuration.
type MemoryRemote interface {
	GetMemoryConfig(ctx context.Context) (MemoryConfig, error)
	UpdateMemoryConfig(ctx context.Context, cfg MemoryConfig) (MemoryConfig, error)
}

// TenantLookup resolves the WeKnora space id for signed tokens.
type TenantLookup interface {
	WeKnoraTenantID(ctx context.Context) (string, error)
}

// MemoryService manages workspace memory config and terminal-user tokens.
type MemoryService struct {
	remote       MemoryRemote
	tenants      TenantLookup
	tokens       *ExternalUserTokenIssuer
	fallbackTID  string
}

// NewMemoryService wires workspace memory APIs.
func NewMemoryService(remote MemoryRemote, tenants TenantLookup, tokens *ExternalUserTokenIssuer, fallbackTenantID string) *MemoryService {
	return &MemoryService{remote: remote, tenants: tenants, tokens: tokens, fallbackTID: strings.TrimSpace(fallbackTenantID)}
}

func (s *MemoryService) GetConfig(ctx context.Context) (MemoryConfig, error) {
	if s == nil || s.remote == nil {
		return MemoryConfig{}, fmt.Errorf("WeKnora 未配置")
	}
	return s.remote.GetMemoryConfig(ctx)
}

func (s *MemoryService) UpdateConfig(ctx context.Context, patch MemoryConfig) (MemoryConfig, error) {
	if s == nil || s.remote == nil {
		return MemoryConfig{}, fmt.Errorf("WeKnora 未配置")
	}
	current, err := s.remote.GetMemoryConfig(ctx)
	if err != nil {
		return MemoryConfig{}, err
	}
	merged := MergeMemoryConfig(current, patch)
	if err := ValidateMemoryConfig(merged); err != nil {
		return MemoryConfig{}, err
	}
	saved, err := s.remote.UpdateMemoryConfig(ctx, merged)
	if err != nil {
		return MemoryConfig{}, err
	}
	// Re-read to confirm persistence.
	confirmed, err := s.remote.GetMemoryConfig(ctx)
	if err != nil {
		return saved, nil
	}
	return confirmed, nil
}

// ResolveIdentity builds a WeKnora identity from the authenticated principal.
func (s *MemoryService) ResolveIdentity(ctx context.Context, principal auth.Principal) (PlatformIdentity, error) {
	tid, err := s.resolveWeKnoraTenantID(ctx)
	if err != nil {
		return PlatformIdentity{}, err
	}
	return ResolvePlatformIdentity(principal.UserID, principal.TenantID, principal.Username, principal.AuthMethod, tid)
}

// IssueExternalUserToken signs X-External-User-Token for the current principal.
func (s *MemoryService) IssueExternalUserToken(ctx context.Context, principal auth.Principal) (string, time.Time, PlatformIdentity, error) {
	identity, err := s.ResolveIdentity(ctx, principal)
	if err != nil {
		return "", time.Time{}, PlatformIdentity{}, err
	}
	if s.tokens == nil || !s.tokens.Enabled() {
		return "", time.Time{}, identity, fmt.Errorf("未配置 WEKNORA_EXTERNAL_USER_HMAC，无法签发终端用户令牌")
	}
	token, exp, err := s.tokens.Issue(identity, time.Now().UTC())
	return token, exp, identity, err
}

func (s *MemoryService) resolveWeKnoraTenantID(ctx context.Context) (string, error) {
	if s.tenants != nil {
		id, err := s.tenants.WeKnoraTenantID(ctx)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(id) != "" {
			return strings.TrimSpace(id), nil
		}
	}
	if s.fallbackTID != "" {
		return s.fallbackTID, nil
	}
	return "", fmt.Errorf("未配置 WeKnora 空间 ID（系统配置 weknora_tenant_id 或 WEKNORA_TENANT_ID）")
}

// PersonalStatus exposes whether personal memory pages may be enabled.
func (s *MemoryService) PersonalStatus() PersonalMemoryIsolationStatus {
	return PersonalMemoryIsolation()
}
