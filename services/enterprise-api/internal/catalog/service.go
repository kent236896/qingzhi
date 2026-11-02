package catalog

import (
	"context"
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
}

// MCPRemote creates/updates/deletes WeKnora MCP services.
type MCPRemote interface {
	CreateRemoteMCPService(ctx context.Context, input weknora.CreateRemoteMCPInput) (weknora.RemoteMCPService, error)
	UpdateRemoteMCPService(ctx context.Context, id string, updates map[string]any) (weknora.RemoteMCPService, error)
	DeleteRemoteMCPService(ctx context.Context, id string) error
	TestRemoteMCPService(ctx context.Context, id string) (map[string]any, error)
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
	agents    Store
	mcp       Store
	identity  Identity
	agentAPI  AgentRemote
	mcpAPI    MCPRemote
	artifacts ArtifactRemote
	log       *slog.Logger
}

func NewService(agents, mcp Store, id Identity, remote *weknora.HTTPClient, log *slog.Logger) *Service {
	return &Service{
		agents: agents, mcp: mcp, identity: id,
		agentAPI: remote, mcpAPI: remote, artifacts: remote,
		log: log,
	}
}

func (s *Service) ListAgents(ctx context.Context, actor Actor) ([]Item, error) {
	return s.listVisible(ctx, s.agents, actor)
}

func (s *Service) ListMCP(ctx context.Context, actor Actor) ([]Item, error) {
	return s.listVisible(ctx, s.mcp, actor)
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
	remote, err := s.agentAPI.CreateRemoteAgent(ctx, weknora.CreateRemoteAgentInput{
		Name: input.Name, Description: input.Description, Config: input.Config,
	})
	if err != nil {
		return Item{}, err
	}
	item.RemoteID = remote.ID
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
	remote, err := s.mcpAPI.CreateRemoteMCPService(ctx, weknora.CreateRemoteMCPInput{
		Name: input.Name, Description: input.Description, Enabled: true,
		TransportType: transport, URL: input.URL,
	})
	if err != nil {
		return Item{}, err
	}
	item.RemoteID = remote.ID
	item.TransportType = transport
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
