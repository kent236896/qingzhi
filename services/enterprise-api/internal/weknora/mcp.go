package weknora

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// MCPAuthModeShared is space-level static credentials (admin plane only to write).
const MCPAuthModeShared = "shared"

// MCPAuthModeUserOAuth is per External Principal OAuth (user plane for authorize/status/revoke).
const MCPAuthModeUserOAuth = "user_oauth"

// MCPOAuthStatus is the per-principal OAuth state for an MCP service.
type MCPOAuthStatus struct {
	Authorized       bool   `json:"authorized"`
	State            string `json:"state"`
	RefreshAvailable bool   `json:"refresh_available"`
}

// MCPAuthorizeURLResult is returned by POST .../oauth/authorize-url.
type MCPAuthorizeURLResult struct {
	AuthorizationURL     string `json:"authorization_url"`
	AuthorizationAttempt string `json:"authorization_attempt"`
}

// MCPCredentialsStatus reports which shared credential fields are configured (never secret values).
type MCPCredentialsStatus struct {
	Fields map[string]MCPCredentialField `json:"fields"`
}

// MCPCredentialField is one credential slot.
type MCPCredentialField struct {
	Configured bool `json:"configured"`
}

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
	// AuthType maps to WeKnora auth_config.auth_type: none / api_key / bearer / oauth.
	AuthType string `json:"-"`
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
	if authType := strings.TrimSpace(input.AuthType); authType != "" {
		body["auth_config"] = map[string]any{"auth_type": authType}
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

// GetRemoteMCPService loads one MCP service (admin plane).
func (c *HTTPClient) GetRemoteMCPService(ctx context.Context, id string) (RemoteMCPService, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return RemoteMCPService{}, fmt.Errorf("缺少 MCP 标识")
	}
	status, raw, err := c.ExchangeAdmin(ctx, http.MethodGet, "/api/v1/mcp-services/"+id, nil)
	if err != nil {
		return RemoteMCPService{}, err
	}
	if status == http.StatusNotFound {
		return RemoteMCPService{}, ErrNotFound
	}
	if status != http.StatusOK {
		return RemoteMCPService{}, fmt.Errorf("读取 MCP 失败（HTTP %d）", status)
	}
	return parseRemoteMCP(raw)
}

// RefreshRemoteMCPMetadata asks WeKnora to refresh tool metadata (admin plane).
func (c *HTTPClient) RefreshRemoteMCPMetadata(ctx context.Context, id string) (map[string]any, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("缺少 MCP 标识")
	}
	status, raw, err := c.ExchangeAdmin(ctx, http.MethodPost, "/api/v1/mcp-services/"+id+"/metadata/refresh", map[string]any{})
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("刷新 MCP 元数据失败（HTTP %d）", status)
	}
	var envelope map[string]any
	_ = json.Unmarshal(raw, &envelope)
	if data, ok := envelope["data"].(map[string]any); ok {
		return data, nil
	}
	return envelope, nil
}

// GetRemoteMCPMetadata reads cached tool metadata (admin plane).
func (c *HTTPClient) GetRemoteMCPMetadata(ctx context.Context, id string) (any, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, fmt.Errorf("缺少 MCP 标识")
	}
	status, raw, err := c.ExchangeAdmin(ctx, http.MethodGet, "/api/v1/mcp-services/"+id+"/metadata", nil)
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("读取 MCP 元数据失败（HTTP %d）", status)
	}
	var envelope map[string]any
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, err
	}
	if data, ok := envelope["data"]; ok {
		return data, nil
	}
	return envelope, nil
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

// PutRemoteMCPCredentials writes shared static credentials (admin plane only).
// values keys are field names such as "api_key" or "token"; secrets must not be logged.
func (c *HTTPClient) PutRemoteMCPCredentials(ctx context.Context, id string, values map[string]string) (MCPCredentialsStatus, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return MCPCredentialsStatus{}, fmt.Errorf("缺少 MCP 标识")
	}
	if len(values) == 0 {
		return MCPCredentialsStatus{}, fmt.Errorf("请提供要写入的凭证字段")
	}
	body := make(map[string]any, len(values))
	for k, v := range values {
		body[k] = v
	}
	status, raw, err := c.ExchangeAdmin(ctx, http.MethodPut, "/api/v1/mcp-services/"+id+"/credentials", body)
	if err != nil {
		return MCPCredentialsStatus{}, err
	}
	if status == http.StatusNotFound {
		return MCPCredentialsStatus{}, ErrNotFound
	}
	if status != http.StatusOK {
		return MCPCredentialsStatus{}, fmt.Errorf("写入 MCP 凭证失败（HTTP %d）", status)
	}
	return parseMCPCredentials(raw)
}

// DeleteRemoteMCPCredential clears one shared credential field (admin plane).
func (c *HTTPClient) DeleteRemoteMCPCredential(ctx context.Context, id, field string) error {
	id = strings.TrimSpace(id)
	field = strings.TrimSpace(field)
	if id == "" || field == "" {
		return fmt.Errorf("缺少 MCP 标识或凭证字段")
	}
	status, _, err := c.ExchangeAdmin(ctx, http.MethodDelete, "/api/v1/mcp-services/"+id+"/credentials/"+field, nil)
	if err != nil {
		return err
	}
	if status == http.StatusNotFound {
		return ErrNotFound
	}
	if status != http.StatusOK && status != http.StatusNoContent {
		return fmt.Errorf("删除 MCP 凭证失败（HTTP %d）", status)
	}
	return nil
}

// GetRemoteMCPOAuthStatus reads per-principal OAuth state (user plane; requires External Token).
// When authorizationAttempt is set, WeKnora only treats that attempt as complete (avoids stale tokens).
func (c *HTTPClient) GetRemoteMCPOAuthStatus(ctx context.Context, id, authorizationAttempt string) (MCPOAuthStatus, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return MCPOAuthStatus{}, fmt.Errorf("缺少 MCP 标识")
	}
	path := "/api/v1/mcp-services/" + id + "/oauth/status"
	if attempt := strings.TrimSpace(authorizationAttempt); attempt != "" {
		path += "?authorization_attempt=" + url.QueryEscape(attempt)
	}
	status, raw, err := c.ExchangeUser(ctx, http.MethodGet, path, nil)
	if err != nil {
		return MCPOAuthStatus{}, err
	}
	if status == http.StatusNotFound {
		return MCPOAuthStatus{}, ErrNotFound
	}
	if status != http.StatusOK {
		return MCPOAuthStatus{}, fmt.Errorf("读取 MCP OAuth 状态失败（HTTP %d）", status)
	}
	var envelope struct {
		Data MCPOAuthStatus `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return MCPOAuthStatus{}, err
	}
	return envelope.Data, nil
}

// MCPCallbackRedirectURI builds WeKnora's public OAuth callback absolute URL.
func MCPCallbackRedirectURI(weknoraBaseURL string) string {
	base := strings.TrimRight(strings.TrimSpace(weknoraBaseURL), "/")
	if base == "" {
		return ""
	}
	return base + "/api/v1/mcp-oauth/callback"
}

// CreateRemoteMCPAuthorizeURL starts user OAuth (user plane; requires External Token).
func (c *HTTPClient) CreateRemoteMCPAuthorizeURL(ctx context.Context, id, redirectURI, frontendRedirect string) (MCPAuthorizeURLResult, error) {
	id = strings.TrimSpace(id)
	redirectURI = strings.TrimSpace(redirectURI)
	if id == "" {
		return MCPAuthorizeURLResult{}, fmt.Errorf("缺少 MCP 标识")
	}
	if redirectURI == "" {
		return MCPAuthorizeURLResult{}, fmt.Errorf("请填写 OAuth 回调地址")
	}
	body := map[string]any{"redirect_uri": redirectURI}
	if fr := strings.TrimSpace(frontendRedirect); fr != "" {
		body["frontend_redirect"] = fr
	}
	status, raw, err := c.ExchangeUser(ctx, http.MethodPost, "/api/v1/mcp-services/"+id+"/oauth/authorize-url", body)
	if err != nil {
		return MCPAuthorizeURLResult{}, err
	}
	if status == http.StatusNotFound {
		return MCPAuthorizeURLResult{}, ErrNotFound
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return MCPAuthorizeURLResult{}, fmt.Errorf("获取 MCP OAuth 授权地址失败（HTTP %d）", status)
	}
	var envelope struct {
		Data MCPAuthorizeURLResult `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return MCPAuthorizeURLResult{}, err
	}
	return envelope.Data, nil
}

// RevokeRemoteMCPOAuthToken drops the current principal's OAuth token (user plane).
func (c *HTTPClient) RevokeRemoteMCPOAuthToken(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("缺少 MCP 标识")
	}
	status, _, err := c.ExchangeUser(ctx, http.MethodDelete, "/api/v1/mcp-services/"+id+"/oauth/token", nil)
	if err != nil {
		return err
	}
	if status == http.StatusNotFound {
		return ErrNotFound
	}
	if status != http.StatusOK && status != http.StatusNoContent {
		return fmt.Errorf("撤销 MCP OAuth 令牌失败（HTTP %d）", status)
	}
	return nil
}

func parseMCPCredentials(raw []byte) (MCPCredentialsStatus, error) {
	var envelope struct {
		Data MCPCredentialsStatus `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return MCPCredentialsStatus{}, err
	}
	if envelope.Data.Fields != nil {
		return envelope.Data, nil
	}
	var direct MCPCredentialsStatus
	if err := json.Unmarshal(raw, &direct); err != nil {
		return MCPCredentialsStatus{}, err
	}
	return direct, nil
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
