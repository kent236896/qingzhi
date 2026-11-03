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

func loadEnv() {
	b, _ := os.ReadFile(`d:\go_code\qinzhi\.env`)
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		p := strings.SplitN(line, "=", 2)
		if os.Getenv(strings.TrimSpace(p[0])) == "" {
			_ = os.Setenv(strings.TrimSpace(p[0]), strings.TrimSpace(p[1]))
		}
	}
}

func tok(secret, sub, tid string) string {
	h := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	now := time.Now().UTC()
	c, _ := json.Marshal(map[string]any{
		"sub": sub, "tenant_id": tid, "aud": "weknora",
		"iat": now.Unix(), "exp": now.Add(10 * time.Minute).Unix(),
	})
	p := base64.RawURLEncoding.EncodeToString(c)
	in := h + "." + p
	m := hmac.New(sha256.New, []byte(secret))
	_, _ = m.Write([]byte(in))
	return in + "." + base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

func do(method, url, apiKey, bearer, ext string, body any) (int, string) {
	var rdr io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		rdr = bytes.NewReader(raw)
	}
	req, _ := http.NewRequest(method, url, rdr)
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	if ext != "" {
		req.Header.Set("X-External-User-Token", ext)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err.Error()
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return resp.StatusCode, string(raw)
}

func clip(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > n {
		return s[:n] + "..."
	}
	return s
}

func main() {
	loadEnv()
	base := strings.TrimRight(os.Getenv("WEKNORA_BASE_URL"), "/")
	api := os.Getenv("WEKNORA_API_KEY")
	hmacSecret := os.Getenv("WEKNORA_EXTERNAL_USER_HMAC")
	tid := os.Getenv("WEKNORA_TENANT_ID")

	st, raw := do("POST", base+"/api/v1/auth/login", "", "", "", map[string]any{
		"email": os.Getenv("WEKNORA_ADMIN_USER"), "password": os.Getenv("WEKNORA_ADMIN_PASSWORD"),
	})
	var login struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal([]byte(raw), &login)
	admin := login.Data.Token
	fmt.Println("login", st)

	extA := tok(hmacSecret, "probe-user-a", tid)
	body := map[string]any{
		"name": "qz-probe-user-agent", "description": "user create",
		"config": map[string]any{"agent_mode": "quick-answer", "kb_selection_mode": "none", "mcp_selection_mode": "none"},
	}
	st, raw = do("POST", base+"/api/v1/agents", api, "", extA, body)
	fmt.Printf("AGENT_CREATE_USER status=%d body=%s\n", st, clip(raw, 700))
	var cr struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal([]byte(raw), &cr)
	id, _ := cr.Data["id"].(string)
	fmt.Println("created_by", cr.Data["created_by"], "id", id)

	st, raw = do("POST", base+"/api/v1/agents", api, admin, extA, body)
	fmt.Printf("AGENT_CREATE_ADMIN_PLUS_EXT status=%d body=%s\n", st, clip(raw, 400))
	var cr2 struct {
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal([]byte(raw), &cr2)
	id2, _ := cr2.Data["id"].(string)
	fmt.Println("created_by2", cr2.Data["created_by"])

	if id != "" {
		extB := tok(hmacSecret, "probe-user-b", tid)
		st, raw = do("DELETE", base+"/api/v1/agents/"+id, api, "", extB, nil)
		fmt.Printf("AGENT_DELETE_BY_B status=%d body=%s\n", st, clip(raw, 200))
		st, raw = do("DELETE", base+"/api/v1/agents/"+id, api, "", extA, nil)
		fmt.Printf("AGENT_DELETE_BY_A status=%d body=%s\n", st, clip(raw, 200))
	}
	if id2 != "" {
		st, _ = do("DELETE", base+"/api/v1/agents/"+id2, api, admin, "", nil)
		fmt.Printf("AGENT_DELETE_ADMIN2 status=%d\n", st)
	}

	mcpID := "c28e6504-94fc-417a-877f-aca03803d713"
	paths := []string{
		"/api/v1/mcp-services/" + mcpID + "/oauth/authorize",
		"/api/v1/mcp-services/" + mcpID + "/oauth/authorization",
		"/api/v1/mcp-services/" + mcpID + "/oauth/start",
		"/api/v1/mcp-services/" + mcpID + "/authorize",
		"/api/v1/mcp-oauth/" + mcpID + "/authorize",
		"/api/v1/mcp-services/" + mcpID + "/tools/approvals",
	}
	for _, p := range paths {
		st, raw = do("POST", base+p, api, "", extA, map[string]any{})
		fmt.Printf("PATH POST %s status=%d body=%s\n", p, st, clip(raw, 180))
		st, raw = do("GET", base+p, api, "", extA, nil)
		fmt.Printf("PATH GET %s status=%d body=%s\n", p, st, clip(raw, 180))
	}
	st, raw = do("GET", base+"/api/v1/mcp-services/"+mcpID+"/tools", api, admin, "", nil)
	fmt.Printf("MCP_TOOLS_EXISTING status=%d body=%s\n", st, clip(raw, 500))
}
