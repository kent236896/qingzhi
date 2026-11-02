package weknora

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// SetAdminCredentials stores WeKnora tenant-admin email/password used to obtain a short-lived JWT.
// This is distinct from WEKNORA_API_KEY: API Key is the workspace key; admin JWT comes from /auth/login.
func (c *HTTPClient) SetAdminCredentials(email, password string) {
	if c == nil {
		return
	}
	email = strings.TrimSpace(email)
	password = strings.TrimSpace(password)
	c.mu.Lock()
	defer c.mu.Unlock()
	changed := c.adminEmail != email || c.adminPassword != password
	c.adminEmail = email
	c.adminPassword = password
	if changed {
		c.adminToken = ""
		c.adminTokenExp = time.Time{}
		c.rebuildLocked()
	}
}

// AdminBearer returns the cached tenant-admin JWT, if any.
func (c *HTTPClient) AdminBearer() string {
	if c == nil {
		return ""
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.adminToken
}

// EnsureAdminAuth logs in with stored admin credentials when no usable admin JWT is cached.
// Required for tenant admin APIs once api-principal-config.mode=signed_token (API Key alone returns 401).
func (c *HTTPClient) EnsureAdminAuth(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("WeKnora 未配置")
	}
	c.mu.RLock()
	token := c.adminToken
	exp := c.adminTokenExp
	email := c.adminEmail
	password := c.adminPassword
	c.mu.RUnlock()
	if token != "" && time.Now().Before(exp.Add(-30*time.Second)) {
		return nil
	}
	if email == "" || password == "" {
		return fmt.Errorf("请填写并保存 WeKnora 管理员邮箱与密码（用于登录换 JWT，不是 API Key）")
	}
	next, loginErr := c.AdminLogin(ctx, email, password)
	if loginErr != nil {
		return loginErr
	}
	c.mu.Lock()
	c.adminToken = next
	// WeKnora access tokens are typically long-lived; refresh at least hourly.
	c.adminTokenExp = time.Now().Add(55 * time.Minute)
	c.rebuildLocked()
	c.mu.Unlock()
	return nil
}

// AdminLogin exchanges WeKnora email/password for a tenant-admin JWT.
// Used only server-side; credentials never go to the browser.
// Does not send X-API-Key — login is account auth, separate from API Key.
func (c *HTTPClient) AdminLogin(ctx context.Context, email, password string) (string, error) {
	email = strings.TrimSpace(email)
	password = strings.TrimSpace(password)
	if email == "" || password == "" {
		return "", fmt.Errorf("请填写 WeKnora 管理员邮箱与密码")
	}
	_, baseURL, _ := c.session()
	if baseURL == "" {
		return "", fmt.Errorf("WeKnora 未配置")
	}
	body, err := json.Marshal(map[string]any{"email": email, "password": password})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/auth/login", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("WeKnora 管理员登录失败（HTTP %d）", resp.StatusCode)
	}
	token := extractAdminToken(raw)
	if token == "" {
		return "", fmt.Errorf("WeKnora 登录响应缺少 token")
	}
	return token, nil
}

func extractAdminToken(raw []byte) string {
	var envelope map[string]any
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return ""
	}
	if data, ok := envelope["data"].(map[string]any); ok {
		if tok := tokenFromMap(data); tok != "" {
			return tok
		}
	}
	return tokenFromMap(envelope)
}

func tokenFromMap(m map[string]any) string {
	for _, key := range []string{"token", "access_token", "jwt"} {
		if v, ok := m[key].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// EnsureSignedTokenPrincipal sets the space api-principal-config to signed_token.
func (c *HTTPClient) EnsureSignedTokenPrincipal(ctx context.Context, tenantID, adminToken, hmacSecret string) (string, error) {
	tenantID = strings.TrimSpace(tenantID)
	adminToken = strings.TrimSpace(adminToken)
	hmacSecret = strings.TrimSpace(hmacSecret)
	if tenantID == "" {
		return "", fmt.Errorf("缺少 WeKnora 空间 ID")
	}
	if adminToken == "" {
		return "", fmt.Errorf("缺少 WeKnora 管理员令牌")
	}
	if len(hmacSecret) < 16 {
		return "", fmt.Errorf("未配置 WEKNORA_EXTERNAL_USER_HMAC（至少 16 字符）")
	}
	path := "/api/v1/tenants/" + tenantID + "/api-principal-config"
	payload := map[string]any{
		"mode":        "signed_token",
		"hmac_secret": hmacSecret,
	}
	status, raw, err := c.exchangeWithBearer(ctx, http.MethodPut, path, adminToken, payload)
	if err != nil {
		return "", err
	}
	if status != http.StatusOK && status != http.StatusCreated && status != http.StatusNoContent {
		msg := strings.TrimSpace(string(raw))
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", status)
		}
		return "", fmt.Errorf("写入 api-principal-config 失败（%s）", msg)
	}
	status, raw, err = c.exchangeWithBearer(ctx, http.MethodGet, path, adminToken, nil)
	if err != nil {
		return "signed_token", nil
	}
	if status != http.StatusOK {
		return "signed_token", nil
	}
	var envelope map[string]any
	_ = json.Unmarshal(raw, &envelope)
	if data, ok := envelope["data"].(map[string]any); ok {
		envelope = data
	}
	mode, _ := envelope["mode"].(string)
	mode = strings.TrimSpace(mode)
	if mode == "" {
		mode = "signed_token"
	}
	if mode != "signed_token" {
		return mode, fmt.Errorf("api-principal-config 未生效，当前 mode=%s", mode)
	}
	return mode, nil
}

func (c *HTTPClient) exchangeWithBearer(ctx context.Context, method, path, bearer string, body any) (int, []byte, error) {
	_, baseURL, apiKey := c.session()
	if baseURL == "" {
		return 0, nil, fmt.Errorf("WeKnora 未配置")
	}
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, reader)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	// Prefer admin Bearer for principal-config; API Key alone is rejected under signed_token.
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, raw, nil
}
