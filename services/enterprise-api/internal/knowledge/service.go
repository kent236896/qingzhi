package knowledge

import (
	"context"
	"errors"
	"log/slog"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"qingzhi/enterprise-api/internal/config"
	"qingzhi/enterprise-api/internal/identity"
	"qingzhi/enterprise-api/internal/weknora"
)

// Service owns local knowledge bases and calls WeKnora for storage and wiki.
type Service struct {
	store     Store
	identity  *identity.Service
	remote    weknora.Client
	cfg       config.KnowledgeConfig
	remoteCfg config.WeKnoraConfig
	connMu    sync.RWMutex
	log       *slog.Logger
	directory ModelDirectory
}

func NewService(store Store, users *identity.Service, remote weknora.Client, knowledge config.KnowledgeConfig, remoteCfg config.WeKnoraConfig, log *slog.Logger) *Service {
	return &Service{store: store, identity: users, remote: remote, cfg: knowledge, remoteCfg: remoteCfg, log: log}
}

// SetModelDirectory makes scene bindings the source of default model ids.
func (s *Service) SetModelDirectory(directory ModelDirectory) {
	if s != nil {
		s.directory = directory
	}
}

// SetConnection marks WeKnora ready after the system settings page saves an address and key.
func (s *Service) SetConnection(baseURL, apiKey string) {
	if s == nil {
		return
	}
	s.connMu.Lock()
	s.remoteCfg.BaseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	s.remoteCfg.APIKey = strings.TrimSpace(apiKey)
	s.connMu.Unlock()
}

func (s *Service) remoteReady() bool {
	if s == nil || s.remote == nil {
		return false
	}
	s.connMu.RLock()
	defer s.connMu.RUnlock()
	return s.remoteCfg.Enabled()
}

func (s *Service) Models(ctx context.Context) (KnowledgeModels, error) {
	if s.directory == nil {
		return KnowledgeModels{}, nil
	}
	return s.directory.KnowledgeModels(ctx)
}

func (s *Service) Limits() Limits {
	return Limits{
		MaxFileMB: s.cfg.MaxFileMB, MaxFiles: s.cfg.MaxFiles, ZipMaxTotalMB: s.cfg.ZipMaxTotalMB,
		ZipMaxEntries: s.cfg.ZipMaxEntries, AllowedExts: append([]string(nil), s.cfg.AllowedExts...),
	}
}

type CreateInput struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	ScopeType    string   `json:"scope_type"`
	ScopeID      string   `json:"scope_id"`
	TenantIDs    []string `json:"tenant_ids"`
	ShareUserIDs []string `json:"share_user_ids"`
	WikiEnabled  bool     `json:"wiki_enabled"`
}

// ChatKnowledgeBase is a sanitized list item for the workbench chat selector.
type ChatKnowledgeBase struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	ScopeType   string `json:"scope_type"`
}

// ListChatKnowledgeBases returns knowledge bases the actor may use for RAG.
// It reuses the same visibility rules as List but does not require knowledge:query.
func (s *Service) ListChatKnowledgeBases(ctx context.Context, actor Actor, keyword string) ([]ChatKnowledgeBase, error) {
	page, err := s.List(ctx, actor, "", keyword, 1, 100)
	if err != nil {
		return nil, err
	}
	out := make([]ChatKnowledgeBase, 0, len(page.Items))
	for _, item := range page.Items {
		out = append(out, ChatKnowledgeBase{
			ID: item.ID, Name: item.Name, Description: item.Description, ScopeType: item.ScopeType,
		})
	}
	return out, nil
}

// ResolveRemoteKnowledgeBaseIDs maps platform KB ids to WeKnora remote ids after authz.
// Any missing, invisible, or unbound id fails the whole request.
func (s *Service) ResolveRemoteKnowledgeBaseIDs(ctx context.Context, actor Actor, platformIDs []string) ([]string, error) {
	if len(platformIDs) == 0 {
		return nil, invalid("请至少选择一个知识库")
	}
	seen := map[string]struct{}{}
	remoteIDs := make([]string, 0, len(platformIDs))
	for _, id := range platformIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			return nil, invalid("知识库标识不合法")
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		item, _, err := s.visible(ctx, actor, id)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(item.RemoteID) == "" {
			return nil, invalid("知识库尚未同步到 WeKnora，无法用于对话")
		}
		remoteIDs = append(remoteIDs, item.RemoteID)
	}
	if len(remoteIDs) == 0 {
		return nil, invalid("请至少选择一个知识库")
	}
	return remoteIDs, nil
}

func (s *Service) List(ctx context.Context, actor Actor, scopeType, keyword string, page, pageSize int) (Page[Base], error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	facts, err := s.facts(ctx, actor)
	if err != nil {
		return Page[Base]{}, err
	}
	items, err := s.store.ListBases(ctx)
	if err != nil {
		return Page[Base]{}, err
	}
	filtered := make([]Base, 0)
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	for _, item := range items {
		if scopeType != "" && item.ScopeType != scopeType {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToLower(item.Name), keyword) {
			continue
		}
		if !CanView(item, item.Shares(), actor, facts) {
			continue
		}
		item.RemoteID = ""
		filtered = append(filtered, item)
	}
	start := (page - 1) * pageSize
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	pageItems := filtered[start:end]
	if pageItems == nil {
		pageItems = []Base{}
	}
	return Page[Base]{Items: pageItems, Page: page, PageSize: pageSize, Total: int64(len(filtered))}, nil
}

func (s *Service) Get(ctx context.Context, actor Actor, id string) (Base, error) {
	item, facts, err := s.visible(ctx, actor, id)
	if err != nil {
		return Base{}, err
	}
	_ = facts
	item.RemoteID = ""
	return item, nil
}

func (s *Service) Create(ctx context.Context, actor Actor, input CreateInput) (Base, error) {
	if !s.remoteReady() {
		return Base{}, notReady("请先在系统配置中填写 WeKnora 地址和 API Key")
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)
	if input.Name == "" || utf8.RuneCountInString(input.Name) > 100 {
		return Base{}, invalid("知识库名称长度必须在 1 到 100 个字符之间")
	}
	if utf8.RuneCountInString(input.Description) > 500 {
		return Base{}, invalid("描述不能超过 500 个字符")
	}
	tenants, err := s.allowedTenants(ctx, actor, input.TenantIDs)
	if err != nil {
		return Base{}, err
	}
	scopeID, err := s.validateScope(ctx, actor, input.ScopeType, input.ScopeID, tenants)
	if err != nil {
		return Base{}, err
	}
	defaults, err := s.Models(ctx)
	if err != nil {
		return Base{}, notReady("无法读取默认模型")
	}
	if input.WikiEnabled && defaults.WikiSynthesis.ID == "" {
		return Base{}, invalid("已勾选 Wiki，但未在系统配置中绑定 Wiki 生成模型")
	}
	if defaults.Embedding.ID == "" || defaults.Summary.ID == "" {
		return Base{}, invalid("请先在系统配置中设置向量模型和摘要模型")
	}
	shares := []string{}
	if input.ScopeType == ScopePersonal {
		shares, err = s.cleanUsers(ctx, input.ShareUserIDs, actor.UserID)
		if err != nil {
			return Base{}, err
		}
	}
	remoteInput := weknora.CreateKnowledgeBaseInput{
		Name: input.Name, Description: input.Description,
		EmbeddingModelID: defaults.Embedding.ID, SummaryModelID: defaults.Summary.ID,
		WikiEnabled: input.WikiEnabled,
	}
	if input.WikiEnabled {
		remoteInput.Wiki = &weknora.WikiConfig{SynthesisModelID: defaults.WikiSynthesis.ID, ExtractionGranularity: "standard"}
	}
	started := time.Now()
	remoteBase, err := s.remote.CreateKnowledgeBase(ctx, remoteInput)
	if err != nil {
		return Base{}, remoteError(err)
	}
	now := time.Now().UTC()
	item := Base{
		ID: newID(), Name: input.Name, Description: input.Description, RemoteID: remoteBase.ID,
		ScopeType: input.ScopeType, ScopeID: scopeID, WikiEnabled: input.WikiEnabled, CreatorID: actor.UserID,
		TenantIDs: tenants, ShareUserIDs: shares, Creator: actor.Username, Updater: actor.Username,
		CreateTime: now, UpdateTime: now,
	}
	err = s.store.Transaction(ctx, func(tx Store) error {
		if err := tx.InsertBase(ctx, item); err != nil {
			return err
		}
		if len(shares) > 0 {
			return tx.ReplaceShares(ctx, item.ID, shares)
		}
		return nil
	})
	if err != nil {
		if deleteErr := s.remote.DeleteKnowledgeBase(ctx, remoteBase.ID); deleteErr != nil && s.log != nil {
			s.log.Error("rollback remote knowledge base failed", slog.String("component", "knowledge"), slog.String("weknora_kb_id", remoteBase.ID), slog.String("error", deleteErr.Error()))
		}
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			return Base{}, conflict("同一归属下已有同名知识库")
		}
		return Base{}, err
	}
	if s.log != nil {
		s.log.Info("knowledge base created", slog.String("component", "knowledge"), slog.String("kb_id", item.ID), slog.String("weknora_kb_id", remoteBase.ID), slog.Int64("duration_ms", time.Since(started).Milliseconds()))
	}
	item.RemoteID = ""
	return item, nil
}

func (s *Service) Update(ctx context.Context, actor Actor, id string, name, description *string, tenantIDs, shareUserIDs []string, tenantsSet, sharesSet bool) (Base, error) {
	item, _, err := s.writable(ctx, actor, id)
	if err != nil {
		return Base{}, err
	}
	if name != nil {
		item.Name = strings.TrimSpace(*name)
		if item.Name == "" || utf8.RuneCountInString(item.Name) > 100 {
			return Base{}, invalid("知识库名称长度必须在 1 到 100 个字符之间")
		}
	}
	if description != nil {
		item.Description = strings.TrimSpace(*description)
		if utf8.RuneCountInString(item.Description) > 500 {
			return Base{}, invalid("描述不能超过 500 个字符")
		}
	}
	if tenantsSet {
		item.TenantIDs, err = s.allowedTenants(ctx, actor, tenantIDs)
		if err != nil {
			return Base{}, err
		}
	}
	if sharesSet {
		if item.ScopeType != ScopePersonal {
			return Base{}, invalid("请使用分享接口设置组织或角色知识库的分享范围")
		}
		item.ShareUserIDs, err = s.cleanUsers(ctx, shareUserIDs, actor.UserID)
		if err != nil {
			return Base{}, err
		}
	}
	item.Updater = actor.Username
	item.UpdateTime = time.Now().UTC()
	err = s.store.Transaction(ctx, func(tx Store) error {
		if err := tx.UpdateBase(ctx, item); err != nil {
			return err
		}
		if tenantsSet {
			if err := tx.ReplaceTenants(ctx, item.ID, item.TenantIDs); err != nil {
				return err
			}
		}
		if sharesSet {
			return tx.ReplaceShares(ctx, item.ID, item.ShareUserIDs)
		}
		return nil
	})
	item.RemoteID = ""
	return item, err
}

// ShareInput updates who can view a knowledge base beyond its own scope.
type ShareInput struct {
	ShareUserIDs []string `json:"share_user_ids"`
	ShareOrgIDs  []string `json:"share_org_ids"`
	ShareRoleIDs []string `json:"share_role_ids"`
}

// Share replaces share targets for a writable knowledge base.
func (s *Service) Share(ctx context.Context, actor Actor, id string, input ShareInput) (Base, error) {
	item, _, err := s.writable(ctx, actor, id)
	if err != nil {
		return Base{}, err
	}
	if !CanShare(item, actor) {
		return Base{}, forbidden("当前知识库不支持分享或无权分享")
	}
	switch item.ScopeType {
	case ScopePersonal:
		if len(input.ShareOrgIDs) > 0 || len(input.ShareRoleIDs) > 0 {
			return Base{}, invalid("个人知识库只能分享给用户")
		}
		item.ShareUserIDs, err = s.cleanUsers(ctx, input.ShareUserIDs, actor.UserID)
		if err != nil {
			return Base{}, err
		}
		item.ShareOrgIDs = []string{}
		item.ShareRoleIDs = []string{}
	case ScopeOrg, ScopeRole:
		if len(input.ShareUserIDs) > 0 {
			return Base{}, invalid("组织或角色知识库只能分享给组织或角色")
		}
		excludeOrg := ""
		excludeRole := ""
		if item.ScopeType == ScopeOrg {
			excludeOrg = item.ScopeID
		}
		if item.ScopeType == ScopeRole {
			excludeRole = item.ScopeID
		}
		item.ShareOrgIDs, err = s.cleanOrgs(ctx, input.ShareOrgIDs, item.TenantIDs, excludeOrg)
		if err != nil {
			return Base{}, err
		}
		item.ShareRoleIDs, err = s.cleanRoles(ctx, input.ShareRoleIDs, item.TenantIDs, excludeRole)
		if err != nil {
			return Base{}, err
		}
		item.ShareUserIDs = []string{}
	default:
		return Base{}, invalid("岗位知识库暂不支持分享")
	}
	item.Updater = actor.Username
	item.UpdateTime = time.Now().UTC()
	err = s.store.Transaction(ctx, func(tx Store) error {
		if err := tx.UpdateBase(ctx, item); err != nil {
			return err
		}
		if item.ScopeType == ScopePersonal {
			if err := tx.ReplaceShares(ctx, item.ID, item.ShareUserIDs); err != nil {
				return err
			}
			return tx.ReplaceShareScopes(ctx, item.ID, nil, nil)
		}
		if err := tx.ReplaceShares(ctx, item.ID, nil); err != nil {
			return err
		}
		return tx.ReplaceShareScopes(ctx, item.ID, item.ShareOrgIDs, item.ShareRoleIDs)
	})
	item.RemoteID = ""
	return item, err
}

func (s *Service) Delete(ctx context.Context, actor Actor, id string) error {
	item, _, err := s.writable(ctx, actor, id)
	if err != nil {
		return err
	}
	if s.remote != nil && item.RemoteID != "" {
		if err := s.remote.DeleteKnowledgeBase(ctx, item.RemoteID); err != nil && !errors.Is(err, weknora.ErrNotFound) {
			return remoteError(err)
		}
	}
	return s.store.DeleteBase(ctx, id)
}

func (s *Service) WikiStatus(ctx context.Context, actor Actor, id string) (WikiStatus, error) {
	item, _, err := s.visible(ctx, actor, id)
	if err != nil {
		return WikiStatus{}, err
	}
	files, err := s.store.ListFiles(ctx, id)
	if err != nil {
		return WikiStatus{}, err
	}
	total, terminal := docCounts(files)
	if !item.WikiEnabled {
		return WikiStatus{Status: WikiDisabled, DocTotal: total, DocTerminal: terminal}, nil
	}
	stats := weknora.WikiStats{}
	if s.remote != nil && item.RemoteID != "" && total > 0 {
		stats, err = s.remote.WikiStats(ctx, item.RemoteID)
		if err != nil {
			return WikiStatus{}, remoteError(err)
		}
	}
	elapsed := time.Duration(0)
	now := time.Now().UTC()
	if item.WikiWaitStart != nil {
		elapsed = now.Sub(*item.WikiWaitStart)
	}
	status := ComputeWikiStatus(WikiStatusInput{
		Enabled: true, TotalPages: stats.TotalPages, PendingTasks: stats.PendingTasks, IsActive: stats.IsActive,
		DocTotal: total, DocTerminal: terminal, Elapsed: elapsed, Timeout: s.cfg.WikiStatusTimeout,
	})
	wait := item.WikiWaitStart
	if status == WikiGenerating && terminal == total && stats.TotalPages == 0 && !stats.IsActive && stats.PendingTasks == 0 {
		if wait == nil {
			wait = &now
		}
	} else if status == WikiReady || status == WikiEmpty || terminal < total {
		wait = nil
	}
	if (wait == nil) != (item.WikiWaitStart == nil) || (wait != nil && item.WikiWaitStart != nil && !wait.Equal(*item.WikiWaitStart)) {
		item.WikiWaitStart = wait
		item.UpdateTime = now
		_ = s.store.UpdateBase(ctx, item)
	}
	return WikiStatus{Status: status, PageCount: stats.TotalPages, DocTotal: total, DocTerminal: terminal, UpdatedAt: &now}, nil
}

func (s *Service) WikiGraph(ctx context.Context, actor Actor, id string, query url.Values) (weknora.WikiGraph, error) {
	item, err := s.wikiReadyBase(ctx, actor, id)
	if err != nil {
		return weknora.WikiGraph{}, err
	}
	return s.remote.WikiGraph(ctx, item.RemoteID, query)
}

func (s *Service) wikiReadyBase(ctx context.Context, actor Actor, id string) (Base, error) {
	item, _, err := s.visible(ctx, actor, id)
	if err != nil {
		return Base{}, err
	}
	if !item.WikiEnabled {
		return Base{}, invalid("这个知识库没有启用 Wiki")
	}
	if !s.remoteReady() {
		return Base{}, notReady("请先在系统配置中填写 WeKnora 地址和 API Key")
	}
	return item, nil
}

func (s *Service) visible(ctx context.Context, actor Actor, id string) (Base, ScopeFacts, error) {
	item, err := s.store.FindBase(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return Base{}, ScopeFacts{}, notFound("知识库不存在")
	}
	if err != nil {
		return Base{}, ScopeFacts{}, err
	}
	facts, err := s.facts(ctx, actor)
	if err != nil {
		return Base{}, ScopeFacts{}, err
	}
	if !CanView(item, item.Shares(), actor, facts) {
		return Base{}, ScopeFacts{}, forbidden("无权查看这个知识库")
	}
	return item, facts, nil
}

func (s *Service) writable(ctx context.Context, actor Actor, id string) (Base, ScopeFacts, error) {
	item, facts, err := s.visible(ctx, actor, id)
	if err != nil {
		return Base{}, ScopeFacts{}, err
	}
	if !CanWrite(item, actor) {
		return Base{}, ScopeFacts{}, forbidden("无权修改这个知识库")
	}
	return item, facts, nil
}

func (s *Service) facts(ctx context.Context, actor Actor) (ScopeFacts, error) {
	facts := ScopeFacts{OrgIDs: map[string]struct{}{}, PostIDs: map[string]struct{}{}, RoleIDs: map[string]struct{}{}}
	if s.identity == nil {
		return facts, nil
	}
	for _, tenantID := range actor.TenantIDs {
		assignments, err := s.identity.ListAssignmentRecords(ctx, tenantID, actor.UserID)
		if err != nil {
			return ScopeFacts{}, err
		}
		orgs, err := s.identity.ListOrganizationRecords(ctx, tenantID)
		if err != nil {
			return ScopeFacts{}, err
		}
		parent := map[string]string{}
		for _, org := range orgs {
			parent[org.ID] = org.ParentID
		}
		assigned := make([]string, 0, len(assignments))
		for _, item := range assignments {
			assigned = append(assigned, item.OrganizationID)
		}
		for id := range VisibleOrgIDs(assigned, parent) {
			facts.OrgIDs[id] = struct{}{}
		}
		posts, err := s.identity.ListUserPosts(ctx, tenantID, actor.UserID)
		if err == nil {
			for _, post := range posts {
				facts.PostIDs[post.ID] = struct{}{}
			}
		} else if !errors.Is(err, identity.ErrNotFound) {
			return ScopeFacts{}, err
		}
		roleIDs, err := s.identity.ListUserRoleIDs(ctx, tenantID, actor.UserID)
		if err != nil {
			return ScopeFacts{}, err
		}
		for _, id := range roleIDs {
			facts.RoleIDs[id] = struct{}{}
		}
	}
	return facts, nil
}

func (s *Service) allowedTenants(ctx context.Context, actor Actor, requested []string) ([]string, error) {
	if len(requested) == 0 {
		return nil, invalid("至少选择一个租户")
	}
	allowed := map[string]struct{}{}
	for _, id := range actor.TenantIDs {
		allowed[id] = struct{}{}
	}
	if actor.CanQueryAll && s.identity != nil {
		page, err := s.identity.ListTenants(ctx, identity.ListQuery{Page: 1, PageSize: 100})
		if err != nil {
			return nil, err
		}
		for _, tenant := range page.Items {
			allowed[tenant.ID] = struct{}{}
		}
	}
	out := make([]string, 0, len(requested))
	seen := map[string]struct{}{}
	for _, id := range requested {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		if _, ok := allowed[id]; !ok {
			return nil, forbidden("不能选择未加入的租户")
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil, invalid("至少选择一个租户")
	}
	return out, nil
}

func (s *Service) validateScope(ctx context.Context, actor Actor, scopeType, scopeID string, tenants []string) (string, error) {
	switch scopeType {
	case ScopePersonal:
		if scopeID != "" && scopeID != actor.UserID {
			return "", invalid("个人知识库只能属于当前用户")
		}
		return actor.UserID, nil
	case ScopeOrg, ScopePost, ScopeRole:
		if !actor.CanManage {
			return "", forbidden("组织、岗位和角色知识库只能由管理员创建")
		}
		if scopeID == "" {
			return "", invalid("请选择归属对象")
		}
	default:
		return "", invalid("归属类型不正确")
	}
	if s.identity == nil {
		return "", notReady("身份服务不可用")
	}
	tenantSet := map[string]struct{}{}
	for _, id := range tenants {
		tenantSet[id] = struct{}{}
	}
	switch scopeType {
	case ScopeOrg:
		org, err := s.identity.FindOrganizationByID(ctx, scopeID)
		if errors.Is(err, identity.ErrNotFound) {
			return "", invalid("组织不存在")
		}
		if err != nil {
			return "", err
		}
		if _, ok := tenantSet[org.TenantID]; !ok {
			return "", invalid("所选组织不属于已选租户")
		}
	case ScopePost:
		found := false
		for tenantID := range tenantSet {
			if _, err := s.identity.GetPost(ctx, tenantID, scopeID); err == nil {
				found = true
				break
			} else if !errors.Is(err, identity.ErrNotFound) {
				return "", err
			}
		}
		if !found {
			return "", invalid("岗位不存在或不属于已选租户")
		}
	case ScopeRole:
		found := false
		for tenantID := range tenantSet {
			if _, err := s.identity.GetRole(ctx, tenantID, scopeID); err == nil {
				found = true
				break
			} else if !errors.Is(err, identity.ErrNotFound) {
				return "", err
			}
		}
		if !found {
			return "", invalid("角色不存在或不属于已选租户")
		}
	}
	return scopeID, nil
}

func (s *Service) cleanUsers(ctx context.Context, userIDs []string, owner string) ([]string, error) {
	out := make([]string, 0, len(userIDs))
	seen := map[string]struct{}{}
	for _, id := range userIDs {
		id = strings.TrimSpace(id)
		if id == "" || id == owner {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		if s.identity != nil {
			if _, err := s.identity.GetUser(ctx, id); errors.Is(err, identity.ErrNotFound) {
				return nil, invalid("分享对象不存在")
			} else if err != nil {
				return nil, err
			}
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, nil
}

func (s *Service) cleanOrgs(ctx context.Context, orgIDs, tenants []string, exclude string) ([]string, error) {
	tenantSet := map[string]struct{}{}
	for _, id := range tenants {
		tenantSet[id] = struct{}{}
	}
	out := make([]string, 0, len(orgIDs))
	seen := map[string]struct{}{}
	for _, id := range orgIDs {
		id = strings.TrimSpace(id)
		if id == "" || id == exclude {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		if s.identity == nil {
			return nil, notReady("身份服务不可用")
		}
		org, err := s.identity.FindOrganizationByID(ctx, id)
		if errors.Is(err, identity.ErrNotFound) {
			return nil, invalid("分享组织不存在")
		}
		if err != nil {
			return nil, err
		}
		if _, ok := tenantSet[org.TenantID]; !ok {
			return nil, invalid("分享组织必须属于知识库已发布的租户")
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, nil
}

func (s *Service) cleanRoles(ctx context.Context, roleIDs, tenants []string, exclude string) ([]string, error) {
	out := make([]string, 0, len(roleIDs))
	seen := map[string]struct{}{}
	for _, id := range roleIDs {
		id = strings.TrimSpace(id)
		if id == "" || id == exclude {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		if s.identity == nil {
			return nil, notReady("身份服务不可用")
		}
		found := false
		for _, tenantID := range tenants {
			if _, err := s.identity.GetRole(ctx, tenantID, id); err == nil {
				found = true
				break
			} else if !errors.Is(err, identity.ErrNotFound) {
				return nil, err
			}
		}
		if !found {
			return nil, invalid("分享角色不存在或不属于知识库已发布的租户")
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out, nil
}

func docCounts(files []File) (int, int) {
	total, terminal := 0, 0
	for _, file := range files {
		if file.ParseStatus == "skipped" {
			continue
		}
		total++
		if file.ParseStatus == "completed" || file.ParseStatus == "failed" || file.ParseStatus == "duplicate" {
			terminal++
		}
	}
	return total, terminal
}

func splitFolder(custom string) (string, string) {
	custom = path.Clean(strings.ReplaceAll(custom, "\\", "/"))
	if custom == "." {
		return "", ""
	}
	return path.Dir(custom), path.Base(custom)
}

func remoteError(err error) error {
	var apiErr *weknora.APIError
	if errors.As(err, &apiErr) {
		text := strings.ToLower(apiErr.Body)
		if strings.Contains(text, "quota") || strings.Contains(apiErr.Body, "配额") || strings.Contains(apiErr.Body, "存储") {
			return invalid("知识库存储空间不足")
		}
		if apiErr.StatusCode == 409 {
			return conflict("文件已存在")
		}
		return invalid("知识库服务暂时无法完成这个操作")
	}
	return err
}
