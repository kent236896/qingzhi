package catalog

import (
	"context"
	"encoding/json"
	"testing"

	"qingzhi/enterprise-api/internal/scoped"
)

type stubKBMapper struct {
	remoteByID map[string]string
}

func (s stubKBMapper) MapVisibleRemoteIDs(_ context.Context, _, _ string, _ []string, platformIDs []string) ([]string, error) {
	out := make([]string, 0, len(platformIDs))
	for _, id := range platformIDs {
		remote, ok := s.remoteByID[id]
		if !ok || remote == "" {
			return nil, notFound("知识库不存在或无权访问")
		}
		out = append(out, remote)
	}
	return out, nil
}

func TestNormalizePlatformAgentConfigRejectsKBAllAllowsMCPAll(t *testing.T) {
	_, err := NormalizePlatformAgentConfig(map[string]any{"kb_selection_mode": "all"})
	if err == nil {
		t.Fatal("expected kb all error")
	}
	cfg, err := NormalizePlatformAgentConfig(map[string]any{
		"agent_mode": "smart-reasoning", "mcp_selection_mode": "all",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MCPSelectionMode != SelectionAll {
		t.Fatalf("mcp mode = %s", cfg.MCPSelectionMode)
	}
}

func TestCreateAgentExpandsMCPAllToVisibleRemotes(t *testing.T) {
	agents := newMemStore()
	mcpStore := newMemStore()
	_ = mcpStore.Insert(context.Background(), Item{
		ID: "mcp-plat-1", Name: "工具A", RemoteID: "wk-mcp-1",
		ScopeType: scoped.ScopePersonal, CreatorID: "u1", TenantIDs: []string{"t1"},
	})
	_ = mcpStore.Insert(context.Background(), Item{
		ID: "mcp-plat-2", Name: "工具B", RemoteID: "wk-mcp-2",
		ScopeType: scoped.ScopePersonal, CreatorID: "other", TenantIDs: []string{"t1"},
	})
	remote := &stubAgents{}
	svc := &Service{
		agents: agents, mcp: mcpStore, identity: stubIdentity{},
		agentAPI: remote, mcpAPI: &stubMCP{},
	}
	actor := Actor{UserID: "u1", Username: "alice", TenantIDs: []string{"t1"}}
	_, err := svc.CreateAgent(context.Background(), actor, CreateInput{
		Name: "助手", ScopeType: scoped.ScopePersonal, TenantIDs: []string{"t1"},
		Config: map[string]any{
			"agent_mode": "smart-reasoning", "mcp_selection_mode": "all",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	services, _ := remote.lastConfig["mcp_services"].([]string)
	if remote.lastConfig["mcp_selection_mode"] != SelectionSelected || len(services) != 1 || services[0] != "wk-mcp-1" {
		t.Fatalf("remote mcp = %#v", remote.lastConfig)
	}
}

func TestCreateAgentMapsPlatformIDsToRemoteConfig(t *testing.T) {
	agents := newMemStore()
	mcpStore := newMemStore()
	_ = mcpStore.Insert(context.Background(), Item{
		ID: "mcp-plat-1", Name: "工具A", RemoteID: "wk-mcp-1",
		ScopeType: scoped.ScopePersonal, CreatorID: "u1", TenantIDs: []string{"t1"},
	})
	remote := &stubAgents{}
	svc := &Service{
		agents: agents, mcp: mcpStore, identity: stubIdentity{},
		agentAPI: remote, mcpAPI: &stubMCP{},
		knowledge: stubKBMapper{remoteByID: map[string]string{"kb-plat-1": "wk-kb-1"}},
	}
	actor := Actor{UserID: "u1", Username: "alice", TenantIDs: []string{"t1"}}
	item, err := svc.CreateAgent(context.Background(), actor, CreateInput{
		Name: "研究助手", ScopeType: scoped.ScopePersonal, TenantIDs: []string{"t1"},
		Config: map[string]any{
			"agent_mode":         "smart-reasoning",
			"agent_type":         "wiki-qa",
			"system_prompt":      "你是Wiki研究员",
			"kb_selection_mode":  "selected",
			"knowledge_base_ids": []string{"kb-plat-1"},
			"mcp_selection_mode": "selected",
			"mcp_ids":            []string{"mcp-plat-1"},
			"web_search_enabled": true,
			"multi_turn_enabled": true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var stored PlatformAgentConfig
	if err := json.Unmarshal(item.Config, &stored); err != nil {
		t.Fatal(err)
	}
	if stored.SystemPrompt != "你是Wiki研究员" || len(stored.KnowledgeBaseIDs) != 1 || stored.KnowledgeBaseIDs[0] != "kb-plat-1" {
		t.Fatalf("stored platform config = %+v", stored)
	}
	if remote.lastConfig["knowledge_bases"].([]string)[0] != "wk-kb-1" {
		t.Fatalf("remote kb = %#v", remote.lastConfig["knowledge_bases"])
	}
	if remote.lastConfig["mcp_services"].([]string)[0] != "wk-mcp-1" {
		t.Fatalf("remote mcp = %#v", remote.lastConfig["mcp_services"])
	}
	if remote.lastConfig["agent_type"] != "wiki-qa" {
		t.Fatalf("agent_type = %#v", remote.lastConfig["agent_type"])
	}
}

func TestCreateAgentRejectsMCPOutsideSmartReasoning(t *testing.T) {
	svc := &Service{
		agents: newMemStore(), mcp: newMemStore(), identity: stubIdentity{},
		agentAPI: &stubAgents{}, mcpAPI: &stubMCP{},
	}
	_, err := svc.CreateAgent(context.Background(), Actor{UserID: "u1", Username: "a", TenantIDs: []string{"t1"}}, CreateInput{
		Name: "x", ScopeType: scoped.ScopePersonal, TenantIDs: []string{"t1"},
		Config: map[string]any{
			"agent_mode": "quick-answer", "mcp_selection_mode": "selected", "mcp_ids": []string{"m1"},
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
