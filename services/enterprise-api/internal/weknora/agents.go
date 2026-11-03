package weknora

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// RemoteAgent is a WeKnora custom agent (admin plane).
type RemoteAgent struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Avatar      string          `json:"avatar"`
	IsBuiltin   bool            `json:"is_builtin"`
	Config      json.RawMessage `json:"config,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// CreateRemoteAgentInput is the minimal create body.
type CreateRemoteAgentInput struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Avatar      string         `json:"avatar,omitempty"`
	Config      map[string]any `json:"config,omitempty"`
}

// ListRemoteAgents lists agents in the WeKnora workspace (admin plane).
// creatorFilter may be "" or "mine" (WeKnora admin user, not External Principal).
func (c *HTTPClient) ListRemoteAgents(ctx context.Context, creatorFilter string) ([]RemoteAgent, error) {
	path := "/api/v1/agents"
	if strings.TrimSpace(creatorFilter) != "" {
		path = path + "?creator=" + strings.TrimSpace(creatorFilter)
	}
	status, raw, err := c.ExchangeAdmin(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("列出智能体失败（HTTP %d）", status)
	}
	var envelope struct {
		Data []RemoteAgent `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, err
	}
	if envelope.Data == nil {
		return []RemoteAgent{}, nil
	}
	return envelope.Data, nil
}

// CopyRemoteAgent duplicates an agent via admin JWT.
func (c *HTTPClient) CopyRemoteAgent(ctx context.Context, id string) (RemoteAgent, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return RemoteAgent{}, fmt.Errorf("缺少智能体标识")
	}
	status, raw, err := c.ExchangeAdmin(ctx, http.MethodPost, "/api/v1/agents/"+id+"/copy", map[string]any{})
	if err != nil {
		return RemoteAgent{}, err
	}
	if status == http.StatusNotFound {
		return RemoteAgent{}, ErrNotFound
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return RemoteAgent{}, fmt.Errorf("复制智能体失败（HTTP %d）", status)
	}
	return parseRemoteAgent(raw)
}

// CreateRemoteAgent creates an agent via admin JWT.
func (c *HTTPClient) CreateRemoteAgent(ctx context.Context, input CreateRemoteAgentInput) (RemoteAgent, error) {
	if strings.TrimSpace(input.Name) == "" {
		return RemoteAgent{}, fmt.Errorf("请填写智能体名称")
	}
	if input.Config == nil {
		input.Config = map[string]any{
			"agent_mode":         "quick-answer",
			"kb_selection_mode":  "none",
			"mcp_selection_mode": "none",
		}
	}
	status, raw, err := c.ExchangeAdmin(ctx, http.MethodPost, "/api/v1/agents", input)
	if err != nil {
		return RemoteAgent{}, err
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return RemoteAgent{}, fmt.Errorf("创建智能体失败（HTTP %d）", status)
	}
	return parseRemoteAgent(raw)
}

// GetRemoteAgent loads one agent.
func (c *HTTPClient) GetRemoteAgent(ctx context.Context, id string) (RemoteAgent, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return RemoteAgent{}, fmt.Errorf("缺少智能体标识")
	}
	status, raw, err := c.ExchangeAdmin(ctx, http.MethodGet, "/api/v1/agents/"+id, nil)
	if err != nil {
		return RemoteAgent{}, err
	}
	if status == http.StatusNotFound {
		return RemoteAgent{}, ErrNotFound
	}
	if status != http.StatusOK {
		return RemoteAgent{}, fmt.Errorf("读取智能体失败（HTTP %d）", status)
	}
	return parseRemoteAgent(raw)
}

// UpdateRemoteAgent replaces name/description/config.
func (c *HTTPClient) UpdateRemoteAgent(ctx context.Context, id string, input CreateRemoteAgentInput) (RemoteAgent, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return RemoteAgent{}, fmt.Errorf("缺少智能体标识")
	}
	status, raw, err := c.ExchangeAdmin(ctx, http.MethodPut, "/api/v1/agents/"+id, input)
	if err != nil {
		return RemoteAgent{}, err
	}
	if status == http.StatusNotFound {
		return RemoteAgent{}, ErrNotFound
	}
	if status != http.StatusOK {
		return RemoteAgent{}, fmt.Errorf("更新智能体失败（HTTP %d）", status)
	}
	return parseRemoteAgent(raw)
}

// DeleteRemoteAgent deletes a WeKnora agent.
func (c *HTTPClient) DeleteRemoteAgent(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("缺少智能体标识")
	}
	status, _, err := c.ExchangeAdmin(ctx, http.MethodDelete, "/api/v1/agents/"+id, nil)
	if err != nil {
		return err
	}
	if status == http.StatusNotFound {
		return ErrNotFound
	}
	if status != http.StatusOK && status != http.StatusNoContent {
		return fmt.Errorf("删除智能体失败（HTTP %d）", status)
	}
	return nil
}

func parseRemoteAgent(raw []byte) (RemoteAgent, error) {
	var envelope struct {
		Data    RemoteAgent `json:"data"`
		Success bool        `json:"success"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return RemoteAgent{}, err
	}
	if envelope.Data.ID != "" {
		return envelope.Data, nil
	}
	var direct RemoteAgent
	if err := json.Unmarshal(raw, &direct); err != nil {
		return RemoteAgent{}, err
	}
	return direct, nil
}
