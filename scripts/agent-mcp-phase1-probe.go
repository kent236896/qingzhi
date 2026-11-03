package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func env(k, d string) string {
	v := strings.TrimSpace(os.Getenv(k))
	if v == "" {
		return d
	}
	return v
}

func mustLoadDotEnv(path string) {
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		if os.Getenv(k) == "" {
			_ = os.Setenv(k, v)
		}
	}
}

func issueToken(hmacSecret, sub, tenantID string, ttl time.Duration) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	now := time.Now().UTC()
	claims, _ := json.Marshal(map[string]any{
		"sub": sub, "tenant_id": tenantID, "aud": "weknora",
		"iat": now.Unix(), "exp": now.Add(ttl).Unix(),
	})
	payload := base64.RawURLEncoding.EncodeToString(claims)
	input := header + "." + payload
	mac := hmac.New(sha256.New, []byte(hmacSecret))
	_, _ = mac.Write([]byte(input))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return input + "." + sig
}

func do(method, url, apiKey, adminBearer, extToken string, body any) (int, string) {
	var rdr io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		rdr = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, url, rdr)
	if err != nil {
		return 0, err.Error()
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	if adminBearer != "" {
		req.Header.Set("Authorization", "Bearer "+adminBearer)
	}
	if extToken != "" {
		req.Header.Set("X-External-User-Token", extToken)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err.Error()
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return resp.StatusCode, string(raw)
}

func adminLogin(base, email, password string) string {
	status, raw := do(http.MethodPost, base+"/api/v1/auth/login", "", "", "", map[string]any{
		"email": email, "password": password,
	})
	fmt.Printf("ADMIN_LOGIN status=%d\n", status)
	var env struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
		Token string `json:"token"`
	}
	_ = json.Unmarshal([]byte(raw), &env)
	if env.Data.Token != "" {
		return env.Data.Token
	}
	return env.Token
}

func clip(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

func main() {
	root := `d:\go_code\qinzhi`
	mustLoadDotEnv(root + `\.env`)
	base := strings.TrimRight(env("WEKNORA_BASE_URL", ""), "/")
	apiKey := env("WEKNORA_API_KEY", "")
	hmacSecret := env("WEKNORA_EXTERNAL_USER_HMAC", "")
	tenantID := env("WEKNORA_TENANT_ID", "10000")
	email := env("WEKNORA_ADMIN_USER", "")
	password := env("WEKNORA_ADMIN_PASSWORD", "")
	if base == "" {
		fmt.Println("missing WEKNORA_BASE_URL")
		os.Exit(1)
	}
	fmt.Println("BASE", base)
	fmt.Println("TENANT", tenantID)

	admin := adminLogin(base, email, password)
	if admin == "" {
		fmt.Println("admin login failed")
		os.Exit(1)
	}
	fmt.Println("ADMIN_TOKEN_OK")

	probe := func(label, method, path, bearer, ext string, body any) {
		status, raw := do(method, base+path, apiKey, bearer, ext, body)
		fmt.Printf("%s method=%s path=%s status=%d body=%s\n", label, method, path, status, clip(raw, 500))
	}

	// Agent list / create / get / patch / copy
	probe("AGENT_LIST_ADMIN", http.MethodGet, "/api/v1/agents", admin, "", nil)
	probe("AGENT_LIST_ADMIN_MINE", http.MethodGet, "/api/v1/agents?creator=mine", admin, "", nil)
	probe("AGENT_LIST_USER_EXT", http.MethodGet, "/api/v1/agents", "", issueToken(hmacSecret, "probe-user-a", tenantID, 10*time.Minute), nil)

	createBody := map[string]any{
		"name":        "qz-probe-agent-" + time.Now().Format("150405"),
		"description": "phase1 probe, delete me",
		"config": map[string]any{
			"agent_mode": "quick-answer", "kb_selection_mode": "none", "mcp_selection_mode": "none",
		},
	}
	status, raw := do(http.MethodPost, base+"/api/v1/agents", apiKey, admin, "", createBody)
	fmt.Printf("AGENT_CREATE_ADMIN status=%d body=%s\n", status, clip(raw, 800))
	var created struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal([]byte(raw), &created)
	agentID, _ := created.Data["id"].(string)
	fmt.Println("AGENT_ID", agentID)
	if agentID != "" {
		for _, k := range []string{"created_by", "creator", "creator_id", "user_id", "owner_id"} {
			if v, ok := created.Data[k]; ok {
				fmt.Printf("AGENT_CREATE_FIELD %s=%v\n", k, v)
			}
		}
		probe("AGENT_GET_ADMIN", http.MethodGet, "/api/v1/agents/"+agentID, admin, "", nil)
		probe("AGENT_PATCH_ADMIN", http.MethodPatch, "/api/v1/agents/"+agentID, admin, "", map[string]any{"description": "patched"})
		probe("AGENT_PUT_ADMIN", http.MethodPut, "/api/v1/agents/"+agentID, admin, "", map[string]any{
			"name": created.Data["name"], "description": "put-update",
			"config": createBody["config"],
		})
		probe("AGENT_COPY_ADMIN", http.MethodPost, "/api/v1/agents/"+agentID+"/copy", admin, "", map[string]any{})
		extA := issueToken(hmacSecret, "probe-user-a", tenantID, 10*time.Minute)
		extB := issueToken(hmacSecret, "probe-user-b", tenantID, 10*time.Minute)
		probe("AGENT_GET_USER_A", http.MethodGet, "/api/v1/agents/"+agentID, "", extA, nil)
		probe("AGENT_DELETE_USER_A", http.MethodDelete, "/api/v1/agents/"+agentID, "", extA, nil)
		probe("AGENT_DELETE_ADMIN", http.MethodDelete, "/api/v1/agents/"+agentID, admin, "", nil)
		_ = extB
	}

	// MCP services
	probe("MCP_LIST_ADMIN", http.MethodGet, "/api/v1/mcp-services", admin, "", nil)
	probe("MCP_LIST_USER", http.MethodGet, "/api/v1/mcp-services", "", issueToken(hmacSecret, "probe-user-a", tenantID, 10*time.Minute), nil)

	mcpBody := map[string]any{
		"name": "qz-probe-mcp-" + time.Now().Format("150405"),
		"description": "phase1 probe",
		"enabled": true,
		"transport_type": "sse",
		"url": "https://example.com/mcp",
		"auth_type": "none",
	}
	status, raw = do(http.MethodPost, base+"/api/v1/mcp-services", apiKey, admin, "", mcpBody)
	fmt.Printf("MCP_CREATE_ADMIN status=%d body=%s\n", status, clip(raw, 800))
	var mcpCreated struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal([]byte(raw), &mcpCreated)
	mcpID, _ := mcpCreated.Data["id"].(string)
	fmt.Println("MCP_ID", mcpID)
	if mcpID != "" {
		probe("MCP_GET_ADMIN", http.MethodGet, "/api/v1/mcp-services/"+mcpID, admin, "", nil)
		probe("MCP_TEST_ADMIN", http.MethodPost, "/api/v1/mcp-services/"+mcpID+"/test", admin, "", map[string]any{})
		for _, p := range []string{
			"/api/v1/mcp-services/" + mcpID + "/tools",
			"/api/v1/mcp-services/" + mcpID + "/metadata",
			"/api/v1/mcp-services/" + mcpID + "/metadata/refresh",
			"/api/v1/mcp-services/" + mcpID + "/sync",
			"/api/v1/mcp-services/" + mcpID + "/oauth/authorize",
			"/api/v1/mcp-services/" + mcpID + "/oauth/status",
			"/api/v1/mcp-services/" + mcpID + "/oauth/token",
		} {
			method := http.MethodGet
			if strings.HasSuffix(p, "/refresh") || strings.HasSuffix(p, "/sync") || strings.HasSuffix(p, "/authorize") {
				method = http.MethodPost
			}
			probe("MCP_PATH", method, p, admin, "", map[string]any{})
		}
		extA := issueToken(hmacSecret, "probe-user-a", tenantID, 10*time.Minute)
		probe("MCP_OAUTH_STATUS_USER", http.MethodGet, "/api/v1/mcp-services/"+mcpID+"/oauth/status", "", extA, nil)
		probe("MCP_OAUTH_AUTH_USER", http.MethodPost, "/api/v1/mcp-services/"+mcpID+"/oauth/authorize", "", extA, map[string]any{})
		probe("MCP_DELETE_ADMIN", http.MethodDelete, "/api/v1/mcp-services/"+mcpID, admin, "", nil)
	}

	// Chat / agent-chat paths
	extA := issueToken(hmacSecret, "probe-user-a", tenantID, 10*time.Minute)
	status, raw = do(http.MethodPost, base+"/api/v1/sessions", apiKey, "", extA, map[string]any{"title": "probe-a"})
	fmt.Printf("SESSION_CREATE_A status=%d body=%s\n", status, clip(raw, 500))
	var sess struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
		ID string `json:"id"`
	}
	_ = json.Unmarshal([]byte(raw), &sess)
	sid := sess.Data.ID
	if sid == "" {
		sid = sess.ID
	}
	fmt.Println("SESSION_A", sid)
	if sid != "" {
		probe("AGENT_CHAT_PATH", http.MethodPost, "/api/v1/agent-chat/"+sid, "", extA, map[string]any{"query": "hi"})
		probe("KNOWLEDGE_CHAT_AGENT_TRUE", http.MethodPost, "/api/v1/knowledge-chat/"+sid, "", extA, map[string]any{
			"query": "hi", "agent_enabled": true, "knowledge_base_ids": []string{},
		})
		extB := issueToken(hmacSecret, "probe-user-b", tenantID, 10*time.Minute)
		probe("SESSION_GET_B", http.MethodGet, "/api/v1/sessions/"+sid, "", extB, nil)
		probe("SESSION_DELETE_A", http.MethodDelete, "/api/v1/sessions/"+sid, "", extA, nil)
	}

	// principal config get
	probe("PRINCIPAL_CONFIG", http.MethodGet, "/api/v1/tenants/"+tenantID+"/api-principal-config", admin, "", nil)
}
