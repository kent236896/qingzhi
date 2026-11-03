package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"qingzhi/enterprise-api/internal/identity"
	"qingzhi/enterprise-api/internal/scoped"
	"qingzhi/enterprise-api/internal/weknora"
)

// Identity looks up memberships for scope validation and visibility.
type Identity interface {
	ListAssignmentRecords(ctx context.Context, tenantID, userID string) ([]identity.OrganizationMembership, error)
	ListOrganizationRecords(ctx context.Context, tenantID string) ([]identity.Organization, error)
	ListUserRoleIDs(ctx context.Context, tenantID, userID string) ([]string, error)
	FindOrganizationByID(ctx context.Context, id string) (identity.Organization, error)
	GetRole(ctx context.Context, tenantID, id string) (identity.Role, error)
}

// AgentRemote creates/updates/deletes WeKnora agents.
type AgentRemote interface {
	CreateRemoteAgent(ctx context.Context, input weknora.CreateRemoteAgentInput) (weknora.RemoteAgent, error)
	UpdateRemoteAgent(ctx context.Context, id string, input weknora.CreateRemoteAgentInput) (weknora.RemoteAgent, error)
	DeleteRemoteAgent(ctx context.Context, id string) error
	CopyRemoteAgent(ctx context.Context, id string) (weknora.RemoteAgent, error)
}

// MCPRemote creates/updates/deletes WeKnora MCP services.
type MCPRemote interface {
	CreateRemoteMCPService(ctx context.Context, input weknora.CreateRemoteMCPInput) (weknora.RemoteMCPService, error)
	UpdateRemoteMCPService(ctx context.Context, id string, updates map[string]any) (weknora.RemoteMCPService, error)
	DeleteRemoteMCPService(ctx context.Context, id string) error
	TestRemoteMCPService(ctx context.Context, id string) (map[string]any, error)
	PutRemoteMCPCredentials(ctx context.Context, id string, values map[string]string) (weknora.MCPCredentialsStatus, error)
	DeleteRemoteMCPCredential(ctx context.Context, id, field string) error
	RefreshRemoteMCPMetadata(ctx context.Context, id string) (map[string]any, error)
	GetRemoteMCPMetadata(ctx context.Context, id string) (any, error)
	GetRemoteMCPOAuthStatus(ctx context.Context, id, authorizationAttempt string) (weknora.MCPOAuthStatus, error)
	CreateRemoteMCPAuthorizeURL(ctx context.Context, id, redirectURI, frontendRedirect string) (weknora.MCPAuthorizeURLResult, error)
	RevokeRemoteMCPOAuthToken(ctx context.Context, id string) error
}

// ArtifactRemote lists/deletes WeKnora artifacts.
type ArtifactRemote interface {
	ListArtifactsAdmin(ctx context.Context, keyword string, page, pageSize int) (weknora.ArtifactPage, error)
	ListArtifactsUser(ctx context.Context, keyword string, page, pageSize int) (weknora.ArtifactPage, error)
	DeleteArtifactAdmin(ctx context.Context, sessionID, messageID string, index int) error
	DeleteArtifactUser(ctx context.Context, sessionID, messageID string, index int) error
}

// Service manages agents, MCP toolbox, and artifacts.
type Service struct {
	agents         Store
	mcp            Store
	identity       Identity
	knowledge      KnowledgeRemoteMapper
	agentAPI       AgentRemote
	mcpAPI         MCPRemote
	artifacts      ArtifactRemote
	weknoraBaseURL string
	log            *slog.Logger
}

func NewService(agents, mcp Store, id Identity, remote *weknora.HTTPClient, log *slog.Logger) *Service {
	return &Service{
		agents: agents, mcp: mcp, identity: id,
		agentAPI: remote, mcpAPI: remote, artifacts: remote,
		log: log,
	}
}

// SetWeKnoraBaseURL sets the WeKnora origin used to build MCP OAuth redirect_uri.
func (s *Service) SetWeKnoraBaseURL(base string) {
	if s == nil {
		return
	}
	s.weknoraBaseURL = strings.TrimRight(strings.TrimSpace(base), "/")
}

// SetKnowledgeMapper wires platform KB id → WeKnora remote id for agent config.
func (s *Service) SetKnowledgeMapper(m KnowledgeRemoteMapper) {
	if s == nil {
		return
	}
	s.knowledge = m
}

func (s *Service) ListAgents(ctx context.Context, actor Actor) ([]Item, error) {
	return s.listVisible(ctx, s.agents, actor)
}

// ResolveVisibleAgentRemoteID returns the WeKnora agent id for a platform agent the actor can view.
func (s *Service) ResolveVisibleAgentRemoteID(ctx context.Context, actor Actor, platformAgentID string) (string, error) {
	platformAgentID = strings.TrimSpace(platformAgentID)
	if platformAgentID == "" {
		return "", invalid("缺少智能体标识")
	}
	item, err := s.visible(ctx, s.agents, actor, platformAgentID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(item.RemoteID) == "" {
		return "", notReady("智能体远端不可用")
	}
	return item.RemoteID, nil
}

func (s *Service) ListMCP(ctx context.Context, actor Actor) ([]Item, error) {
	items, err := s.listVisible(ctx, s.mcp, actor)
	if err != nil {
		return nil, err
	}
	for i := range items {
		// Phase 2: no DB column yet; expose stable default for clients.
		items[i].AuthMode = NormalizeMCPAuthMode(items[i].AuthMode)
	}
	return items, nil
}

func (s *Service) listVisible(ctx context.Context, store Store, actor Actor) ([]Item, error) {
	if store == nil {
		return nil, notReady("目录存储不可用")
	}
	facts, err := s.facts(ctx, actor)
	if err != nil {
		return nil, err
	}
	all, err := store.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Item, 0, len(all))
	for _, item := range all {
		if !scoped.CanView(item.Resource(), item.Shares(), actor, facts) {
			continue
		}
		item.Writable = scoped.CanWrite(item.Resource(), actor)
		item.RemoteID = ""
		out = append(out, item)
	}
	return out, nil
}

func (s *Service) CreateAgent(ctx context.Context, actor Actor, input CreateInput) (Item, error) {
	item, err := s.prepareCreate(ctx, actor, input)
	if err != nil {
		return Item{}, err
	}
	if s.agentAPI == nil {
		return Item{}, notReady("WeKnora 未配置")
	}
	platformCfg, err := NormalizePlatformAgentConfig(input.Config)
	if err != nil {
		return Item{}, err
	}
	remoteCfg, err := s.toRemoteAgentConfig(ctx, actor, platformCfg)
	if err != nil {
		return Item{}, err
	}
	remote, err := s.agentAPI.CreateRemoteAgent(ctx, weknora.CreateRemoteAgentInput{
		Name: input.Name, Description: input.Description, Config: remoteCfg,
	})
	if err != nil {
		return Item{}, err
	}
	item.RemoteID = remote.ID
	item.Config = marshalConfig(platformCfg.AsMap())
	if err := s.agents.Insert(ctx, item); err != nil {
		_ = s.agentAPI.DeleteRemoteAgent(ctx, remote.ID)
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			return Item{}, conflict("同一归属下已有同名智能体")
		}
		return Item{}, err
	}
	item.RemoteID = ""
	item.Writable = true
	return item, nil
}

// ResolveMCPAuthMode returns the effective auth mode for an MCP catalog row.
// Until sys_mcp_services.auth_mode exists, this is always shared unless callers pass a mode.
func (s *Service) ResolveMCPAuthMode(item Item) string {
	return NormalizeMCPAuthMode(item.AuthMode)
}

func (s *Service) CreateMCP(ctx context.Context, actor Actor, input CreateInput) (Item, error) {
	item, err := s.prepareCreate(ctx, actor, input)
	if err != nil {
		return Item{}, err
	}
	if s.mcpAPI == nil {
		return Item{}, notReady("WeKnora 未配置")
	}
	transport := strings.TrimSpace(input.TransportType)
	if transport == "" {
		transport = "sse"
	}
	authMode := NormalizeMCPAuthMode(input.AuthMode)
	remoteAuth := ""
	if authMode == MCPAuthModeUserOAuth {
		remoteAuth = "oauth"
	}
	remote, err := s.mcpAPI.CreateRemoteMCPService(ctx, weknora.CreateRemoteMCPInput{
		Name: input.Name, Description: input.Description, Enabled: true,
		TransportType: transport, URL: input.URL, AuthType: remoteAuth,
	})
	if err != nil {
		return Item{}, err
	}
	item.RemoteID = remote.ID
	item.TransportType = transport
	item.URL = strings.TrimSpace(input.URL)
	item.AuthMode = authMode
	if err := s.mcp.Insert(ctx, item); err != nil {
		_ = s.mcpAPI.DeleteRemoteMCPService(ctx, remote.ID)
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			return Item{}, conflict("同一归属下已有同名 MCP")
		}
		return Item{}, err
	}
	item.RemoteID = ""
	item.Writable = true
	return item, nil
}

func (s *Service) UpdateAgent(ctx context.Context, actor Actor, id string, input UpdateInput) (Item, error) {
	item, err := s.writable(ctx, s.agents, actor, id)
	if err != nil {
		return Item{}, err
	}
	name := strings.TrimSpace(input.Name)
	desc := strings.TrimSpace(input.Description)
	if name == "" || utf8.RuneCountInString(name) > 100 {
		return Item{}, invalid("名称长度必须在 1 到 100 个字符之间")
	}
	if utf8.RuneCountInString(desc) > 500 {
		return Item{}, invalid("描述不能超过 500 个字符")
	}
	if item.RemoteID == "" || s.agentAPI == nil {
		return Item{}, notReady("WeKnora 未配置")
	}
	rawCfg := input.Config
	if rawCfg == nil {
		rawCfg = map[string]any{}
		_ = json.Unmarshal(item.Config, &rawCfg)
	}
	platformCfg, err := NormalizePlatformAgentConfig(rawCfg)
	if err != nil {
		return Item{}, err
	}
	remoteCfg, err := s.toRemoteAgentConfig(ctx, actor, platformCfg)
	if err != nil {
		return Item{}, err
	}
	if _, err := s.agentAPI.UpdateRemoteAgent(ctx, item.RemoteID, weknora.CreateRemoteAgentInput{
		Name: name, Description: desc, Config: remoteCfg,
	}); err != nil {
		return Item{}, err
	}
	item.Name = name
	item.Description = desc
	item.Config = marshalConfig(platformCfg.AsMap())
	item.Updater = actor.Username
	item.UpdateTime = time.Now().UTC()
	if item.ScopeType == scoped.ScopePersonal {
		shares := []string{}
		for _, uid := range input.ShareUserIDs {
			uid = strings.TrimSpace(uid)
			if uid == "" || uid == actor.UserID {
				continue
			}
			shares = append(shares, uid)
		}
		item.ShareUserIDs = shares
	}
	if err := s.agents.Update(ctx, item); err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			return Item{}, conflict("同一归属下已有同名智能体")
		}
		return Item{}, err
	}
	item.RemoteID = ""
	item.Writable = true
	return item, nil
}

func (s *Service) CopyAgent(ctx context.Context, actor Actor, id string, input CopyInput) (Item, error) {
	source, err := s.visible(ctx, s.agents, actor, id)
	if err != nil {
		return Item{}, err
	}
	if source.RemoteID == "" || s.agentAPI == nil {
		return Item{}, notReady("WeKnora 未配置")
	}
	remote, err := s.agentAPI.CopyRemoteAgent(ctx, source.RemoteID)
	if err != nil {
		return Item{}, err
	}
	scopeType := strings.TrimSpace(input.ScopeType)
	if scopeType == "" {
		scopeType = scoped.ScopePersonal
	}
	createIn := CreateInput{
		Name: strings.TrimSpace(input.Name), Description: source.Description,
		ScopeType: scopeType, ScopeID: input.ScopeID, TenantIDs: input.TenantIDs, ShareUserIDs: input.ShareUserIDs,
	}
	if createIn.Name == "" {
		createIn.Name = strings.TrimSpace(remote.Name)
	}
	if createIn.Name == "" {
		createIn.Name = source.Name + "（副本）"
	}
	if len(createIn.TenantIDs) == 0 {
		createIn.TenantIDs = source.TenantIDs
	}
	if scopeType != scoped.ScopePersonal && !actor.CanManage {
		_ = s.agentAPI.DeleteRemoteAgent(ctx, remote.ID)
		return Item{}, forbidden("租户/组织/角色智能体需要管理权限")
	}
	item, err := s.prepareCreate(ctx, actor, createIn)
	if err != nil {
		_ = s.agentAPI.DeleteRemoteAgent(ctx, remote.ID)
		return Item{}, err
	}
	item.RemoteID = remote.ID
	item.Config = source.Config
	item.Description = source.Description
	if err := s.agents.Insert(ctx, item); err != nil {
		_ = s.agentAPI.DeleteRemoteAgent(ctx, remote.ID)
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			return Item{}, conflict("同一归属下已有同名智能体")
		}
		return Item{}, err
	}
	item.RemoteID = ""
	item.Writable = true
	return item, nil
}

func (s *Service) UpdateMCP(ctx context.Context, actor Actor, id string, input UpdateInput) (Item, error) {
	item, err := s.writable(ctx, s.mcp, actor, id)
	if err != nil {
		return Item{}, err
	}
	name := strings.TrimSpace(input.Name)
	desc := strings.TrimSpace(input.Description)
	url := strings.TrimSpace(input.URL)
	if name == "" || utf8.RuneCountInString(name) > 100 {
		return Item{}, invalid("名称长度必须在 1 到 100 个字符之间")
	}
	if utf8.RuneCountInString(desc) > 500 {
		return Item{}, invalid("描述不能超过 500 个字符")
	}
	if url == "" {
		return Item{}, invalid("请填写 MCP 服务地址")
	}
	transport := strings.TrimSpace(input.TransportType)
	if transport == "" {
		transport = item.TransportType
	}
	if transport != "sse" && transport != "http-streamable" {
		return Item{}, invalid("工具箱仅支持 MCP（sse / http-streamable）")
	}
	authMode := NormalizeMCPAuthMode(input.AuthMode)
	if item.RemoteID == "" || s.mcpAPI == nil {
		return Item{}, notReady("WeKnora 未配置")
	}
	updates := map[string]any{
		"name": name, "description": desc, "transport_type": transport, "url": url, "enabled": true,
	}
	if authMode == MCPAuthModeUserOAuth {
		updates["auth_config"] = map[string]any{"auth_type": "oauth"}
	}
	if _, err := s.mcpAPI.UpdateRemoteMCPService(ctx, item.RemoteID, updates); err != nil {
		return Item{}, err
	}
	item.Name = name
	item.Description = desc
	item.URL = url
	item.TransportType = transport
	item.AuthMode = authMode
	item.Updater = actor.Username
	item.UpdateTime = time.Now().UTC()
	if item.ScopeType == scoped.ScopePersonal {
		shares := []string{}
		for _, uid := range input.ShareUserIDs {
			uid = strings.TrimSpace(uid)
			if uid == "" || uid == actor.UserID {
				continue
			}
			shares = append(shares, uid)
		}
		item.ShareUserIDs = shares
	}
	if err := s.mcp.Update(ctx, item); err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			return Item{}, conflict("同一归属下已有同名 MCP")
		}
		return Item{}, err
	}
	item.RemoteID = ""
	item.Writable = true
	return item, nil
}

func (s *Service) PutMCPCredentials(ctx context.Context, actor Actor, id string, values map[string]string) (weknora.MCPCredentialsStatus, error) {
	item, err := s.writable(ctx, s.mcp, actor, id)
	if err != nil {
		return weknora.MCPCredentialsStatus{}, err
	}
	if NormalizeMCPAuthMode(item.AuthMode) != MCPAuthModeShared {
		return weknora.MCPCredentialsStatus{}, invalid("仅 shared 模式可写入静态凭证；user_oauth 请走用户授权")
	}
	if item.RemoteID == "" || s.mcpAPI == nil {
		return weknora.MCPCredentialsStatus{}, notReady("MCP 远端不可用")
	}
	return s.mcpAPI.PutRemoteMCPCredentials(ctx, item.RemoteID, values)
}

func (s *Service) DeleteMCPCredential(ctx context.Context, actor Actor, id, field string) error {
	item, err := s.writable(ctx, s.mcp, actor, id)
	if err != nil {
		return err
	}
	if NormalizeMCPAuthMode(item.AuthMode) != MCPAuthModeShared {
		return invalid("仅 shared 模式可管理静态凭证")
	}
	if item.RemoteID == "" || s.mcpAPI == nil {
		return notReady("MCP 远端不可用")
	}
	return s.mcpAPI.DeleteRemoteMCPCredential(ctx, item.RemoteID, field)
}

func (s *Service) RefreshMCPMetadata(ctx context.Context, actor Actor, id string) (map[string]any, error) {
	item, err := s.writable(ctx, s.mcp, actor, id)
	if err != nil {
		return nil, err
	}
	if item.RemoteID == "" || s.mcpAPI == nil {
		return nil, notReady("MCP 远端不可用")
	}
	return s.mcpAPI.RefreshRemoteMCPMetadata(ctx, item.RemoteID)
}

func (s *Service) GetMCPMetadata(ctx context.Context, actor Actor, id string) (any, error) {
	item, err := s.visible(ctx, s.mcp, actor, id)
	if err != nil {
		return nil, err
	}
	if item.RemoteID == "" || s.mcpAPI == nil {
		return nil, notReady("MCP 远端不可用")
	}
	return s.mcpAPI.GetRemoteMCPMetadata(ctx, item.RemoteID)
}

func (s *Service) requireUserOAuthMCP(ctx context.Context, actor Actor, id string) (Item, error) {
	item, err := s.visible(ctx, s.mcp, actor, id)
	if err != nil {
		return Item{}, err
	}
	if NormalizeMCPAuthMode(item.AuthMode) != MCPAuthModeUserOAuth {
		return Item{}, invalid("该 MCP 未启用用户 OAuth（auth_mode 须为 user_oauth）")
	}
	if item.RemoteID == "" || s.mcpAPI == nil {
		return Item{}, notReady("MCP 远端不可用")
	}
	return item, nil
}

// GetMCPOAuthStatus returns the current principal's OAuth state (ctx must carry External Token).
func (s *Service) GetMCPOAuthStatus(ctx context.Context, actor Actor, id, authorizationAttempt string) (weknora.MCPOAuthStatus, error) {
	item, err := s.requireUserOAuthMCP(ctx, actor, id)
	if err != nil {
		return weknora.MCPOAuthStatus{}, err
	}
	return s.mcpAPI.GetRemoteMCPOAuthStatus(ctx, item.RemoteID, authorizationAttempt)
}

// StartMCPOAuthAuthorize builds WeKnora authorize-url for the current principal.
// frontendRedirect is where WeKnora sends the browser after its own /mcp-oauth/callback.
func (s *Service) StartMCPOAuthAuthorize(ctx context.Context, actor Actor, id, frontendRedirect string) (weknora.MCPAuthorizeURLResult, error) {
	item, err := s.requireUserOAuthMCP(ctx, actor, id)
	if err != nil {
		return weknora.MCPAuthorizeURLResult{}, err
	}
	redirectURI := weknora.MCPCallbackRedirectURI(s.weknoraBaseURL)
	if redirectURI == "" {
		return weknora.MCPAuthorizeURLResult{}, notReady("未配置 WeKnora 地址，无法发起 OAuth")
	}
	return s.mcpAPI.CreateRemoteMCPAuthorizeURL(ctx, item.RemoteID, redirectURI, frontendRedirect)
}

// RevokeMCPOAuthToken drops the current principal's OAuth token for this MCP.
func (s *Service) RevokeMCPOAuthToken(ctx context.Context, actor Actor, id string) error {
	item, err := s.requireUserOAuthMCP(ctx, actor, id)
	if err != nil {
		return err
	}
	return s.mcpAPI.RevokeRemoteMCPOAuthToken(ctx, item.RemoteID)
}

func marshalConfig(cfg map[string]any) json.RawMessage {
	if cfg == nil {
		return json.RawMessage(`{}`)
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return raw
}

func (s *Service) DeleteAgent(ctx context.Context, actor Actor, id string) error {
	item, err := s.writable(ctx, s.agents, actor, id)
	if err != nil {
		return err
	}
	if item.RemoteID != "" && s.agentAPI != nil {
		if delErr := s.agentAPI.DeleteRemoteAgent(ctx, item.RemoteID); delErr != nil && !errors.Is(delErr, weknora.ErrNotFound) {
			return delErr
		}
	}
	return s.agents.Delete(ctx, id)
}

func (s *Service) DeleteMCP(ctx context.Context, actor Actor, id string) error {
	item, err := s.writable(ctx, s.mcp, actor, id)
	if err != nil {
		return err
	}
	if item.RemoteID != "" && s.mcpAPI != nil {
		if delErr := s.mcpAPI.DeleteRemoteMCPService(ctx, item.RemoteID); delErr != nil && !errors.Is(delErr, weknora.ErrNotFound) {
			return delErr
		}
	}
	return s.mcp.Delete(ctx, id)
}

func (s *Service) TestMCP(ctx context.Context, actor Actor, id string) (map[string]any, error) {
	item, err := s.visible(ctx, s.mcp, actor, id)
	if err != nil {
		return nil, err
	}
	if item.RemoteID == "" || s.mcpAPI == nil {
		return nil, notReady("MCP 远端不可用")
	}
	return s.mcpAPI.TestRemoteMCPService(ctx, item.RemoteID)
}

func (s *Service) ListArtifactsWithToken(ctx context.Context, keyword string, page, pageSize int) (weknora.ArtifactPage, error) {
	if s.artifacts == nil {
		return weknora.ArtifactPage{}, notReady("WeKnora 未配置")
	}
	return s.artifacts.ListArtifactsUser(ctx, keyword, page, pageSize)
}

func (s *Service) ListArtifactsAdmin(ctx context.Context, keyword string, page, pageSize int) (weknora.ArtifactPage, error) {
	if s.artifacts == nil {
		return weknora.ArtifactPage{}, notReady("WeKnora 未配置")
	}
	return s.artifacts.ListArtifactsAdmin(ctx, keyword, page, pageSize)
}

func (s *Service) DeleteArtifactAdmin(ctx context.Context, sessionID, messageID string, index int) error {
	if s.artifacts == nil {
		return notReady("WeKnora 未配置")
	}
	return s.artifacts.DeleteArtifactAdmin(ctx, sessionID, messageID, index)
}

func (s *Service) DeleteArtifactUser(ctx context.Context, sessionID, messageID string, index int) error {
	if s.artifacts == nil {
		return notReady("WeKnora 未配置")
	}
	return s.artifacts.DeleteArtifactUser(ctx, sessionID, messageID, index)
}

func (s *Service) prepareCreate(ctx context.Context, actor Actor, input CreateInput) (Item, error) {
	name := strings.TrimSpace(input.Name)
	desc := strings.TrimSpace(input.Description)
	if name == "" || utf8.RuneCountInString(name) > 100 {
		return Item{}, invalid("名称长度必须在 1 到 100 个字符之间")
	}
	if utf8.RuneCountInString(desc) > 500 {
		return Item{}, invalid("描述不能超过 500 个字符")
	}
	if !scoped.ValidScope(input.ScopeType) {
		return Item{}, invalid("归属类型不正确（支持 personal / tenant / org_node / role）")
	}
	tenants := input.TenantIDs
	if len(tenants) == 0 {
		if len(actor.TenantIDs) == 0 {
			return Item{}, invalid("至少选择一个租户")
		}
		tenants = []string{actor.TenantIDs[0]}
	}
	for _, tid := range tenants {
		ok := false
		for _, own := range actor.TenantIDs {
			if own == tid {
				ok = true
				break
			}
		}
		if !ok {
			return Item{}, forbidden("不能选择未加入的租户")
		}
	}
	scopeID, err := s.validateScope(ctx, actor, input.ScopeType, input.ScopeID, tenants)
	if err != nil {
		return Item{}, err
	}
	shares := []string{}
	if input.ScopeType == scoped.ScopePersonal {
		for _, uid := range input.ShareUserIDs {
			uid = strings.TrimSpace(uid)
			if uid == "" || uid == actor.UserID {
				continue
			}
			shares = append(shares, uid)
		}
	}
	now := time.Now().UTC()
	return Item{
		ID: uuid.NewString(), Name: name, Description: desc,
		ScopeType: input.ScopeType, ScopeID: scopeID, Status: 0, CreatorID: actor.UserID,
		TenantIDs: tenants, ShareUserIDs: shares, Creator: actor.Username, Updater: actor.Username,
		CreateTime: now, UpdateTime: now,
	}, nil
}

func (s *Service) validateScope(ctx context.Context, actor Actor, scopeType, scopeID string, tenants []string) (string, error) {
	switch scopeType {
	case scoped.ScopePersonal:
		return actor.UserID, nil
	case scoped.ScopeTenant:
		if !actor.CanManage {
			return "", forbidden("租户级资源只能由管理员创建")
		}
		if scopeID == "" {
			scopeID = tenants[0]
		}
		for _, tid := range tenants {
			if tid == scopeID {
				return scopeID, nil
			}
		}
		return "", invalid("租户归属必须属于已选租户")
	case scoped.ScopeOrg, scoped.ScopeRole:
		if !actor.CanManage {
			return "", forbidden("组织/角色资源只能由管理员创建")
		}
		if strings.TrimSpace(scopeID) == "" {
			return "", invalid("请选择归属对象")
		}
		if s.identity == nil {
			return "", notReady("身份服务不可用")
		}
		if scopeType == scoped.ScopeOrg {
			org, err := s.identity.FindOrganizationByID(ctx, scopeID)
			if errors.Is(err, identity.ErrNotFound) {
				return "", invalid("组织不存在")
			}
			if err != nil {
				return "", err
			}
			ok := false
			for _, tid := range tenants {
				if tid == org.TenantID {
					ok = true
					break
				}
			}
			if !ok {
				return "", invalid("所选组织不属于已选租户")
			}
			return scopeID, nil
		}
		found := false
		for _, tid := range tenants {
			if _, err := s.identity.GetRole(ctx, tid, scopeID); err == nil {
				found = true
				break
			} else if !errors.Is(err, identity.ErrNotFound) {
				return "", err
			}
		}
		if !found {
			return "", invalid("角色不存在或不属于已选租户")
		}
		return scopeID, nil
	default:
		return "", invalid("归属类型不正确")
	}
}

func (s *Service) facts(ctx context.Context, actor Actor) (scoped.ScopeFacts, error) {
	facts := scoped.ScopeFacts{OrgIDs: map[string]struct{}{}, RoleIDs: map[string]struct{}{}}
	if s.identity == nil {
		return facts, nil
	}
	for _, tenantID := range actor.TenantIDs {
		assignments, err := s.identity.ListAssignmentRecords(ctx, tenantID, actor.UserID)
		if err != nil {
			return scoped.ScopeFacts{}, err
		}
		orgs, err := s.identity.ListOrganizationRecords(ctx, tenantID)
		if err != nil {
			return scoped.ScopeFacts{}, err
		}
		parent := map[string]string{}
		for _, org := range orgs {
			parent[org.ID] = org.ParentID
		}
		assigned := make([]string, 0, len(assignments))
		for _, item := range assignments {
			assigned = append(assigned, item.OrganizationID)
		}
		for id := range scopedVisibleOrgs(assigned, parent) {
			facts.OrgIDs[id] = struct{}{}
		}
		roleIDs, err := s.identity.ListUserRoleIDs(ctx, tenantID, actor.UserID)
		if err != nil {
			return scoped.ScopeFacts{}, err
		}
		for _, id := range roleIDs {
			facts.RoleIDs[id] = struct{}{}
		}
	}
	return facts, nil
}

func scopedVisibleOrgs(assigned []string, parentOf map[string]string) map[string]struct{} {
	children := map[string][]string{}
	for id, parent := range parentOf {
		children[parent] = append(children[parent], id)
	}
	out := map[string]struct{}{}
	var walkDown func(id string)
	walkDown = func(id string) {
		if id == "" {
			return
		}
		if _, seen := out[id]; seen {
			return
		}
		out[id] = struct{}{}
		for _, child := range children[id] {
			walkDown(child)
		}
	}
	for _, id := range assigned {
		for cur := parentOf[id]; cur != ""; cur = parentOf[cur] {
			out[cur] = struct{}{}
		}
		walkDown(id)
	}
	return out
}

func (s *Service) visible(ctx context.Context, store Store, actor Actor, id string) (Item, error) {
	item, err := store.Get(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return Item{}, notFound("资源不存在或无权访问")
	}
	if err != nil {
		return Item{}, err
	}
	facts, err := s.facts(ctx, actor)
	if err != nil {
		return Item{}, err
	}
	if !scoped.CanView(item.Resource(), item.Shares(), actor, facts) {
		return Item{}, notFound("资源不存在或无权访问")
	}
	return item, nil
}

func (s *Service) writable(ctx context.Context, store Store, actor Actor, id string) (Item, error) {
	item, err := s.visible(ctx, store, actor, id)
	if err != nil {
		return Item{}, err
	}
	if !scoped.CanWrite(item.Resource(), actor) {
		return Item{}, notFound("资源不存在或无权访问")
	}
	return item, nil
}
