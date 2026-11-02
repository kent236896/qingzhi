package weknora

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// RemoteMCPService is a WeKnora MCP toolbox entry.
type RemoteMCPService struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	Enabled       bool   `json:"enabled"`
	TransportType string `json:"transport_type"`
	URL           string `json:"url,omitempty"`
	IsBuiltin     bool   `json:"is_builtin"`
}

// CreateRemoteMCPInput creates an MCP service (SSE / HTTP streamable only for now).
type CreateRemoteMCPInput struct {
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	Enabled       bool              `json:"enabled"`
	TransportType string            `json:"transport_type"`
	URL           string            `json:"url"`
	Headers       map[string]string `json:"headers,omitempty"`
}

// CreateRemoteMCPService registers an MCP service via admin JWT.
func (c *HTTPClient) CreateRemoteMCPService(ctx context.Context, input CreateRemoteMCPInput) (RemoteMCPService, error) {
	if strings.TrimSpace(input.Name) == "" {
		return RemoteMCPService{}, fmt.Errorf("请填写 MCP 名称")
	}
	transport := strings.TrimSpace(input.TransportType)
	if transport == "" {
		transport = "sse"
	}
	if transport != "sse" && transport != "http-streamable" {
		return RemoteMCPService{}, fmt.Errorf("工具箱仅支持 MCP（sse / http-streamable）")
	}
	if strings.TrimSpace(input.URL) == "" {
		return RemoteMCPService{}, fmt.Errorf("请填写 MCP 服务地址")
	}
	body := map[string]any{
		"name":           input.Name,
		"description":    input.Description,
		"enabled":        input.Enabled,
		"transport_type": transport,
		"url":            input.URL,
		"headers":        input.Headers,
	}
	status, raw, err := c.ExchangeAdmin(ctx, http.MethodPost, "/api/v1/mcp-services", body)
	if err != nil {
		return RemoteMCPService{}, err
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return RemoteMCPService{}, fmt.Errorf("创建 MCP 失败（HTTP %d）", status)
	}
	return parseRemoteMCP(raw)
}

// UpdateRemoteMCPService updates an MCP service.
func (c *HTTPClient) UpdateRemoteMCPService(ctx context.Context, id string, updates map[string]any) (RemoteMCPService, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return RemoteMCPService{}, fmt.Errorf("缺少 MCP 标识")
	}
	status, raw, err := c.ExchangeAdmin(ctx, http.MethodPut, "/api/v1/mcp-services/"+id, updates)
	if err != nil {
		return RemoteMCPService{}, err
	}
	if status == http.StatusNotFound {
		return RemoteMCPService{}, ErrNotFound
	}
	if status != http.StatusOK {
		return RemoteMCPService{}, fmt.Errorf("更新 MCP 失败（HTTP %d）", status)
	}
	return parseRemoteMCP(raw)
}

// DeleteRemoteMCPService deletes an MCP service.
func (c *HTTPClient) DeleteRemoteMCPService(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("缺少 MCP 标识")
	}
	status, _, err := c.ExchangeAdmin(ctx, http.MethodDelete, "/api/v1/mcp-services/"+id, nil)
	if err != nil {
		return err
	}
	if status == http.StatusNotFound {
		return ErrNotFound
	}
	if status != http.StatusOK && status != http.StatusNoContent {
		return fmt.Errorf("删除 MCP 失败（HTTP %d）", status)
	}
	return nil
}

// TestRemoteMCPService probes connectivity.
func (c *HTTPClient) TestRemoteMCPService(ctx context.Context, id string) (map[string]any, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("缺少 MCP 标识")
	}
	status, raw, err := c.ExchangeAdmin(ctx, http.MethodPost, "/api/v1/mcp-services/"+id+"/test", map[string]any{})
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("测试 MCP 失败（HTTP %d）", status)
	}
	var envelope map[string]any
	_ = json.Unmarshal(raw, &envelope)
	if data, ok := envelope["data"].(map[string]any); ok {
		return data, nil
	}
	return envelope, nil
}

func parseRemoteMCP(raw []byte) (RemoteMCPService, error) {
	var envelope struct {
		Data RemoteMCPService `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return RemoteMCPService{}, err
	}
	if envelope.Data.ID != "" {
		if envelope.Data.URL == "" {
			// url may be pointer in upstream; ignore
		}
		return envelope.Data, nil
	}
	var direct RemoteMCPService
	if err := json.Unmarshal(raw, &direct); err != nil {
		return RemoteMCPService{}, err
	}
	return direct, nil
}
