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

func main() {
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
	base := strings.TrimRight(os.Getenv("WEKNORA_BASE_URL"), "/")
	api := os.Getenv("WEKNORA_API_KEY")
	secret := os.Getenv("WEKNORA_EXTERNAL_USER_HMAC")
	tid := os.Getenv("WEKNORA_TENANT_ID")
	h := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	now := time.Now().UTC()
	c, _ := json.Marshal(map[string]any{
		"sub": "probe-user-a", "tenant_id": tid, "aud": "weknora",
		"iat": now.Unix(), "exp": now.Add(10 * time.Minute).Unix(),
	})
	p := base64.RawURLEncoding.EncodeToString(c)
	in := h + "." + p
	m := hmac.New(sha256.New, []byte(secret))
	_, _ = m.Write([]byte(in))
	ext := in + "." + base64.RawURLEncoding.EncodeToString(m.Sum(nil))
	mcp := "c28e6504-94fc-417a-877f-aca03803d713"
	call := func(method, path string, body any) {
		var rdr io.Reader
		if body != nil {
			raw, _ := json.Marshal(body)
			rdr = bytes.NewReader(raw)
		}
		req, _ := http.NewRequest(method, base+path, rdr)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", api)
		req.Header.Set("X-External-User-Token", ext)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Println(method, path, err)
			return
		}
		defer resp.Body.Close()
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 800))
		fmt.Printf("%s %s %d %s\n", method, path, resp.StatusCode, strings.ReplaceAll(string(raw), "\n", " "))
	}
	call("POST", "/api/v1/mcp-services/"+mcp+"/oauth/authorize-url", map[string]any{
		"redirect_uri": base + "/api/v1/mcp-oauth/callback",
	})
	call("DELETE", "/api/v1/mcp-services/"+mcp+"/oauth/token", nil)
	call("PUT", "/api/v1/mcp-services/"+mcp+"/credentials", map[string]any{"api_key": "dummy"})
	call("GET", "/api/v1/mcp-services/"+mcp+"/tool-approvals", nil)
	call("GET", "/api/v1/mcp-services/"+mcp+"/tools", nil)
	call("POST", "/api/v1/mcp-services/"+mcp+"/metadata/refresh", map[string]any{})
}
