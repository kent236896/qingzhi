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
		if !CanView(item, item.ShareUserIDs, actor, facts) {
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
			return Base{}, invalid("只有个人知识库可以分享给他人")
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
	if !CanView(item, item.ShareUserIDs, actor, facts) {
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
