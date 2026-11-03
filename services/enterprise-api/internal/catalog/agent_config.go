package catalog

import (
	"context"
	"encoding/json"
	"strings"
	"unicode/utf8"

	"qingzhi/enterprise-api/internal/scoped"
)

// Platform agent config uses Qingzhi knowledge_base_ids / mcp_ids.
// WeKnora receives knowledge_bases / mcp_services (remote ids) via toRemoteAgentConfig.

const (
	AgentModeQuickAnswer    = "quick-answer"
	AgentModeSmartReasoning = "smart-reasoning"

	AgentTypeCustom        = "custom"
	AgentTypeRAGQA         = "rag-qa"
	AgentTypeWikiQA        = "wiki-qa"
	AgentTypeHybridRAGWiki = "hybrid-rag-wiki"

	SelectionNone     = "none"
	SelectionSelected = "selected"
	SelectionAll      = "all"
)

// KnowledgeRemoteMapper maps visible platform KB ids to WeKnora remote ids.
type KnowledgeRemoteMapper interface {
	MapVisibleRemoteIDs(ctx context.Context, userID, username string, tenantIDs []string, platformIDs []string) ([]string, error)
}

// PlatformAgentConfig is the catalog-facing agent config (no remote ids).
type PlatformAgentConfig struct {
	AgentMode                   string   `json:"agent_mode"`
	AgentType                   string   `json:"agent_type,omitempty"`
	SystemPrompt                string   `json:"system_prompt"`
	KBSelectionMode             string   `json:"kb_selection_mode"`
	KnowledgeBaseIDs            []string `json:"knowledge_base_ids"`
	MCPSelectionMode            string   `json:"mcp_selection_mode"`
	MCPIDs                      []string `json:"mcp_ids"`
	WebSearchEnabled            bool     `json:"web_search_enabled"`
	MultiTurnEnabled            bool     `json:"multi_turn_enabled"`
	MemoryEnabled               *bool    `json:"memory_enabled,omitempty"`
	RetrieveKBOnlyWhenMentioned bool     `json:"retrieve_kb_only_when_mentioned"`
	Temperature                 float64  `json:"temperature,omitempty"`
	MaxIterations               int      `json:"max_iterations,omitempty"`
}

func defaultPlatformAgentConfig() PlatformAgentConfig {
	mem := true
	return PlatformAgentConfig{
		AgentMode:        AgentModeQuickAnswer,
		KBSelectionMode:  SelectionNone,
		MCPSelectionMode: SelectionNone,
		MultiTurnEnabled: true,
		MemoryEnabled:    &mem,
		Temperature:      0.7,
	}
}

// NormalizePlatformAgentConfig validates and fills defaults.
// MCP "all" means all platform-visible MCP services (expanded to remote ids on sync).
// KB "all" remains rejected — knowledge isolation must stay explicitly bound.
func NormalizePlatformAgentConfig(raw map[string]any) (PlatformAgentConfig, error) {
	cfg := defaultPlatformAgentConfig()
	if raw == nil {
		return cfg, nil
	}
	blob, err := json.Marshal(raw)
	if err != nil {
		return PlatformAgentConfig{}, invalid("智能体配置不合法")
	}
	if err := json.Unmarshal(blob, &cfg); err != nil {
		return PlatformAgentConfig{}, invalid("智能体配置不合法")
	}

	cfg.AgentMode = strings.TrimSpace(cfg.AgentMode)
	switch cfg.AgentMode {
	case "", AgentModeQuickAnswer:
		cfg.AgentMode = AgentModeQuickAnswer
	case AgentModeSmartReasoning:
		// ok
	default:
		return PlatformAgentConfig{}, invalid("智能体模式仅支持 quick-answer 或 smart-reasoning")
	}

	cfg.AgentType = strings.TrimSpace(cfg.AgentType)
	if cfg.AgentMode == AgentModeQuickAnswer {
		cfg.AgentType = ""
	} else {
		switch cfg.AgentType {
		case "", AgentTypeCustom:
			cfg.AgentType = AgentTypeCustom
		case AgentTypeRAGQA, AgentTypeWikiQA, AgentTypeHybridRAGWiki:
			// ok
		default:
			return PlatformAgentConfig{}, invalid("智能体类型不合法")
		}
	}

	if utf8.RuneCountInString(cfg.SystemPrompt) > 20000 {
		return PlatformAgentConfig{}, invalid("系统提示词不能超过 20000 个字符")
	}

	cfg.KBSelectionMode = strings.TrimSpace(cfg.KBSelectionMode)
	switch cfg.KBSelectionMode {
	case "", SelectionNone:
		cfg.KBSelectionMode = SelectionNone
		cfg.KnowledgeBaseIDs = nil
	case SelectionSelected:
		cfg.KnowledgeBaseIDs = uniqueNonEmpty(cfg.KnowledgeBaseIDs)
		if len(cfg.KnowledgeBaseIDs) == 0 {
			return PlatformAgentConfig{}, invalid("已选择「指定知识库」时请至少勾选一个知识库")
		}
	case "all":
		return PlatformAgentConfig{}, invalid("平台不支持「全部知识库」，请显式勾选可见知识库以落实隔离")
	default:
		return PlatformAgentConfig{}, invalid("知识库选择模式不合法")
	}

	cfg.MCPSelectionMode = strings.TrimSpace(cfg.MCPSelectionMode)
	switch cfg.MCPSelectionMode {
	case "", SelectionNone:
		cfg.MCPSelectionMode = SelectionNone
		cfg.MCPIDs = nil
	case SelectionSelected:
		cfg.MCPIDs = uniqueNonEmpty(cfg.MCPIDs)
		if len(cfg.MCPIDs) == 0 {
			return PlatformAgentConfig{}, invalid("已选择「指定工具」时请至少勾选一个 MCP")
		}
		if cfg.AgentMode != AgentModeSmartReasoning {
			return PlatformAgentConfig{}, invalid("MCP 工具仅在智能推理模式下可用，请先切换模式")
		}
	case SelectionAll:
		cfg.MCPIDs = nil
		if cfg.AgentMode != AgentModeSmartReasoning {
			return PlatformAgentConfig{}, invalid("MCP 工具仅在智能推理模式下可用，请先切换模式")
		}
	default:
		return PlatformAgentConfig{}, invalid("MCP 选择模式不合法")
	}

	if cfg.Temperature < 0 {
		cfg.Temperature = 0
	}
	if cfg.Temperature > 2 {
		cfg.Temperature = 2
	}
	if cfg.MaxIterations < 0 {
		cfg.MaxIterations = 0
	}
	return cfg, nil
}

func (c PlatformAgentConfig) AsMap() map[string]any {
	raw, _ := json.Marshal(c)
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	return out
}

func (s *Service) toRemoteAgentConfig(ctx context.Context, actor Actor, cfg PlatformAgentConfig) (map[string]any, error) {
	remoteKBs := []string{}
	if cfg.KBSelectionMode == SelectionSelected {
		ids, err := s.resolveKnowledgeRemoteIDs(ctx, actor, cfg.KnowledgeBaseIDs)
		if err != nil {
			return nil, err
		}
		remoteKBs = ids
	}
	remoteMCPs := []string{}
	mcpMode := cfg.MCPSelectionMode
	switch cfg.MCPSelectionMode {
	case SelectionSelected:
		ids, err := s.resolveMCPRemoteIDs(ctx, actor, cfg.MCPIDs)
		if err != nil {
			return nil, err
		}
		remoteMCPs = ids
	case SelectionAll:
		// Expand to currently visible platform MCPs as "selected" so WeKnora
		// cannot pull workspace-wide services outside Qingzhi ACL.
		ids, err := s.resolveAllVisibleMCPRemoteIDs(ctx, actor)
		if err != nil {
			return nil, err
		}
		remoteMCPs = ids
		mcpMode = SelectionSelected
	}

	out := map[string]any{
		"agent_mode":                      cfg.AgentMode,
		"system_prompt":                   cfg.SystemPrompt,
		"kb_selection_mode":               cfg.KBSelectionMode,
		"knowledge_bases":                 remoteKBs,
		"mcp_selection_mode":              mcpMode,
		"mcp_services":                    remoteMCPs,
		"web_search_enabled":              cfg.WebSearchEnabled,
		"multi_turn_enabled":              cfg.MultiTurnEnabled,
		"retrieve_kb_only_when_mentioned": cfg.RetrieveKBOnlyWhenMentioned,
		"temperature":                     cfg.Temperature,
	}
	if cfg.AgentType != "" {
		out["agent_type"] = cfg.AgentType
	}
	if cfg.MemoryEnabled != nil {
		out["memory_enabled"] = *cfg.MemoryEnabled
	}
	if cfg.MaxIterations > 0 {
		out["max_iterations"] = cfg.MaxIterations
	}
	return out, nil
}

func (s *Service) resolveKnowledgeRemoteIDs(ctx context.Context, actor Actor, platformIDs []string) ([]string, error) {
	if s.knowledge == nil {
		return nil, notReady("知识库服务不可用，无法绑定智能体知识库")
	}
	ids, err := s.knowledge.MapVisibleRemoteIDs(ctx, actor.UserID, actor.Username, actor.TenantIDs, platformIDs)
	if err != nil {
		return nil, mapKnowledgeErr(err)
	}
	return ids, nil
}

func (s *Service) resolveMCPRemoteIDs(ctx context.Context, actor Actor, platformIDs []string) ([]string, error) {
	out := make([]string, 0, len(platformIDs))
	seen := map[string]struct{}{}
	for _, id := range platformIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		item, err := s.visible(ctx, s.mcp, actor, id)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(item.RemoteID) == "" {
			return nil, invalid("MCP「" + item.Name + "」尚未同步到 WeKnora")
		}
		out = append(out, item.RemoteID)
	}
	if len(out) == 0 {
		return nil, invalid("请至少选择一个 MCP")
	}
	return out, nil
}

func (s *Service) resolveAllVisibleMCPRemoteIDs(ctx context.Context, actor Actor) ([]string, error) {
	if s.mcp == nil {
		return nil, notReady("MCP 目录不可用")
	}
	facts, err := s.facts(ctx, actor)
	if err != nil {
		return nil, err
	}
	all, err := s.mcp.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(all))
	for _, item := range all {
		if !scoped.CanView(item.Resource(), item.Shares(), actor, facts) {
			continue
		}
		if strings.TrimSpace(item.RemoteID) == "" {
			continue
		}
		out = append(out, item.RemoteID)
	}
	if len(out) == 0 {
		return nil, invalid("当前没有已同步的可见 MCP 工具，无法使用「全部工具」")
	}
	return out, nil
}

func uniqueNonEmpty(ids []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
