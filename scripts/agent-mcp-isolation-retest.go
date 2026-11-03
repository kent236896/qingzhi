//go:build ignore

// Live isolation check for agents / toolbox MCP / chat between two platform users.
// Also seeds a few personal MCPs + agents for each user.
//
// Env:
//   API_BASE (default http://127.0.0.1:8080)
//   ISOLATION_USER_A / ISOLATION_PASS_A  (e.g. kun.tang)
//   ISOLATION_USER_B / ISOLATION_PASS_B  (e.g. kent)
//
// Usage: go run ./scripts/agent-mcp-isolation-retest.go
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	base := strings.TrimRight(env("API_BASE", "http://127.0.0.1:8080"), "/")
	userA := strings.TrimSpace(os.Getenv("ISOLATION_USER_A"))
	passA := strings.TrimSpace(os.Getenv("ISOLATION_PASS_A"))
	userB := strings.TrimSpace(os.Getenv("ISOLATION_USER_B"))
	passB := strings.TrimSpace(os.Getenv("ISOLATION_PASS_B"))
	if userA == "" || passA == "" || userB == "" || passB == "" {
		fmt.Println("RESULT=SKIP reason=missing ISOLATION_USER_A/B or ISOLATION_PASS_A/B")
		os.Exit(2)
	}

	client := &http.Client{Timeout: 90 * time.Second}
	tokA, err := login(client, base, userA, passA)
	if err != nil {
		fmt.Printf("RESULT=SKIP reason=login_a_failed err=%v\n", err)
		os.Exit(2)
	}
	tokB, err := login(client, base, userB, passB)
	if err != nil {
		fmt.Printf("RESULT=SKIP reason=login_b_failed err=%v\n", err)
		os.Exit(2)
	}
	fmt.Printf("LOGIN_OK a=%s b=%s\n", userA, userB)

	fail := false
	stamp := time.Now().Unix()

	// --- seed personal MCPs (WeKnora remote create) ---
	mcpA := []string{}
	for i, name := range []string{
		fmt.Sprintf("iso-mcp-a-search-%d", stamp),
		fmt.Sprintf("iso-mcp-a-calc-%d", stamp),
	} {
		id, status, err := createMCP(client, base, tokA, name, fmt.Sprintf("https://example.com/mcp/a/%d", i))
		printCheck("a_create_mcp_"+name, status, err, id)
		if status == 200 && id != "" {
			mcpA = append(mcpA, id)
		} else {
			fail = true
		}
	}
	mcpB := []string{}
	for i, name := range []string{
		fmt.Sprintf("iso-mcp-b-mail-%d", stamp),
		fmt.Sprintf("iso-mcp-b-docs-%d", stamp),
	} {
		id, status, err := createMCP(client, base, tokB, name, fmt.Sprintf("https://example.com/mcp/b/%d", i))
		printCheck("b_create_mcp_"+name, status, err, id)
		if status == 200 && id != "" {
			mcpB = append(mcpB, id)
		} else {
			fail = true
		}
	}

	listA, status, err := listMCP(client, base, tokA)
	printCheck("a_list_mcp", status, err, fmt.Sprintf("count=%d", len(listA)))
	listB, status, err := listMCP(client, base, tokB)
	printCheck("b_list_mcp", status, err, fmt.Sprintf("count=%d", len(listB)))
	if overlaps(listA, mcpB) {
		fmt.Println("VERDICT_DETAIL a_sees_b_mcp=true")
		fail = true
	} else {
		fmt.Println("VERDICT_DETAIL a_sees_b_mcp=false")
	}
	if overlaps(listB, mcpA) {
		fmt.Println("VERDICT_DETAIL b_sees_a_mcp=true")
		fail = true
	} else {
		fmt.Println("VERDICT_DETAIL b_sees_a_mcp=false")
	}
	if len(mcpB) > 0 {
		status, err := getOrDeleteMCP(client, base, tokA, mcpB[0], false)
		printCheck("a_get_b_mcp", status, err, fmt.Sprintf("http=%d", status))
		// list-only API; cross delete should 404
		status, err = getOrDeleteMCP(client, base, tokA, mcpB[0], true)
		printCheck("a_delete_b_mcp", status, err, fmt.Sprintf("http=%d", status))
		if status == 200 {
			fmt.Println("VERDICT_DETAIL a_delete_b_mcp_allowed=true")
			fail = true
		} else if status == 404 || status == 403 {
			fmt.Println("VERDICT_DETAIL a_delete_b_mcp_denied=true")
		} else {
			fail = true
		}
	}

	// --- seed personal agents with MCP binding ---
	agentA, status, err := createAgent(client, base, tokA, fmt.Sprintf("iso-agent-a-%d", stamp), map[string]any{
		"agent_mode": "smart-reasoning", "agent_type": "custom",
		"system_prompt": "仅回答用户 A 可见内容", "mcp_selection_mode": "selected", "mcp_ids": mcpA,
		"kb_selection_mode": "none", "multi_turn_enabled": true,
	})
	printCheck("a_create_agent", status, err, agentA)
	if status != 200 || agentA == "" {
		fail = true
	}
	agentB, status, err := createAgent(client, base, tokB, fmt.Sprintf("iso-agent-b-%d", stamp), map[string]any{
		"agent_mode": "smart-reasoning", "agent_type": "custom",
		"system_prompt": "仅回答用户 B 可见内容", "mcp_selection_mode": "selected", "mcp_ids": mcpB,
		"kb_selection_mode": "none", "multi_turn_enabled": true,
	})
	printCheck("b_create_agent", status, err, agentB)
	if status != 200 || agentB == "" {
		fail = true
	}

	agentsA, status, err := listAgents(client, base, tokA)
	printCheck("a_list_agents", status, err, fmt.Sprintf("count=%d", len(agentsA)))
	agentsB, status, err := listAgents(client, base, tokB)
	printCheck("b_list_agents", status, err, fmt.Sprintf("count=%d", len(agentsB)))
	if contains(agentsA, agentB) {
		fmt.Println("VERDICT_DETAIL a_sees_b_agent=true")
		fail = true
	} else {
		fmt.Println("VERDICT_DETAIL a_sees_b_agent=false")
	}
	if contains(agentsB, agentA) {
		fmt.Println("VERDICT_DETAIL b_sees_a_agent=true")
		fail = true
	} else {
		fmt.Println("VERDICT_DETAIL b_sees_a_agent=false")
	}
	if agentB != "" {
		status, err := deleteAgent(client, base, tokA, agentB)
		printCheck("a_delete_b_agent", status, err, fmt.Sprintf("http=%d", status))
		if status == 200 {
			fmt.Println("VERDICT_DETAIL a_delete_b_agent_allowed=true")
			fail = true
		} else if status == 404 || status == 403 {
			fmt.Println("VERDICT_DETAIL a_delete_b_agent_denied=true")
		} else {
			fail = true
		}
	}

	// A must not create session bound to B's agent
	if agentB != "" {
		sess, status, err := createSession(client, base, tokA, "hijack-agent", agentB)
		printCheck("a_create_session_with_b_agent", status, err, sess)
		if status == 200 {
			fmt.Println("VERDICT_DETAIL a_bind_b_agent_allowed=true")
			fail = true
			if sess != "" {
				_, _, _ = doJSON(client, http.MethodDelete, base+"/api/v1/chat/sessions/"+sess, tokA, nil)
			}
		} else if status == 404 || status == 403 {
			fmt.Println("VERDICT_DETAIL a_bind_b_agent_denied=true")
		} else {
			fmt.Printf("VERDICT_DETAIL a_bind_b_agent_unexpected http=%d\n", status)
			fail = true
		}
	}

	// --- chat session isolation + ask with own agent ---
	sessA, status, err := createSession(client, base, tokA, fmt.Sprintf("iso-chat-a-%d", stamp), agentA)
	printCheck("a_create_session", status, err, sessA)
	sessB, status, err := createSession(client, base, tokB, fmt.Sprintf("iso-chat-b-%d", stamp), agentB)
	printCheck("b_create_session", status, err, sessB)
	if sessA == "" || sessB == "" {
		fail = true
	} else {
		listSA, status, err := listSessions(client, base, tokA)
		printCheck("a_list_sessions", status, err, fmt.Sprintf("count=%d", len(listSA)))
		listSB, status, err := listSessions(client, base, tokB)
		printCheck("b_list_sessions", status, err, fmt.Sprintf("count=%d", len(listSB)))
		if contains(listSA, sessB) {
			fmt.Println("VERDICT_DETAIL a_sees_b_session=true")
			fail = true
		} else {
			fmt.Println("VERDICT_DETAIL a_sees_b_session=false")
		}
		if contains(listSB, sessA) {
			fmt.Println("VERDICT_DETAIL b_sees_a_session=true")
			fail = true
		} else {
			fmt.Println("VERDICT_DETAIL b_sees_a_session=false")
		}

		for _, kind := range []string{"messages", "ask"} {
			status, err := crossSession(client, base, tokA, sessB, kind)
			printCheck("a_cross_"+kind, status, err, fmt.Sprintf("http=%d", status))
			if status == 200 {
				fmt.Printf("VERDICT_DETAIL a_cross_%s_allowed=true\n", kind)
				fail = true
			} else if status == 404 || status == 403 {
				fmt.Printf("VERDICT_DETAIL a_cross_%s_denied=true\n", kind)
			} else if status == 503 {
				fmt.Printf("VERDICT_DETAIL a_cross_%s_unavailable http=503\n", kind)
			} else {
				fmt.Printf("VERDICT_DETAIL a_cross_%s_unexpected http=%d\n", kind, status)
				fail = true
			}
		}

		// own ask (agent-chat path); may fail upstream if MCP URL unreachable — still must not leak
		askStatus, askSnippet, err := askOnce(client, base, tokA, sessA, "用一句话介绍你自己")
		printCheck("a_ask_own_agent", askStatus, err, askSnippet)
		if askStatus != 200 && askStatus != 0 {
			fmt.Printf("VERDICT_DETAIL a_ask_own_http=%d (upstream may fail; isolation still checked)\n", askStatus)
		}
	}

	// cleanup
	if sessA != "" {
		_, _, _ = doJSON(client, http.MethodDelete, base+"/api/v1/chat/sessions/"+sessA, tokA, nil)
	}
	if sessB != "" {
		_, _, _ = doJSON(client, http.MethodDelete, base+"/api/v1/chat/sessions/"+sessB, tokB, nil)
	}
	if agentA != "" {
		_, _ = deleteAgent(client, base, tokA, agentA)
	}
	if agentB != "" {
		_, _ = deleteAgent(client, base, tokB, agentB)
	}
	for _, id := range mcpA {
		_, _ = getOrDeleteMCP(client, base, tokA, id, true)
	}
	for _, id := range mcpB {
		_, _ = getOrDeleteMCP(client, base, tokB, id, true)
	}

	if fail {
		fmt.Println("RESULT=FAIL isolation_not_proven")
		os.Exit(1)
	}
	fmt.Println("RESULT=PASS agent_mcp_chat_isolation")
}

func env(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

func login(client *http.Client, base, user, pass string) (string, error) {
	status, body, err := doJSON(client, http.MethodPost, base+"/api/v1/auth/login", "", map[string]any{
		"username": user, "password": pass,
	})
	if err != nil {
		return "", err
	}
	if status != 200 {
		return "", fmt.Errorf("http=%d", status)
	}
	data, _ := body["data"].(map[string]any)
	tok, _ := data["access_token"].(string)
	if tok == "" {
		return "", fmt.Errorf("missing access_token")
	}
	return tok, nil
}

func createMCP(client *http.Client, base, token, name, url string) (string, int, error) {
	status, body, err := doJSON(client, http.MethodPost, base+"/api/v1/toolbox/mcp", token, map[string]any{
		"name": name, "description": "isolation probe mcp", "scope_type": "personal",
		"tenant_ids": []string{}, "transport_type": "sse", "url": url, "auth_mode": "shared",
	})
	if err != nil {
		return "", status, err
	}
	data, _ := body["data"].(map[string]any)
	id, _ := data["id"].(string)
	return id, status, nil
}

func listMCP(client *http.Client, base, token string) ([]string, int, error) {
	status, body, err := doJSON(client, http.MethodGet, base+"/api/v1/toolbox/mcp", token, nil)
	if err != nil {
		return nil, status, err
	}
	items, _ := body["data"].([]any)
	ids := make([]string, 0, len(items))
	for _, item := range items {
		m, _ := item.(map[string]any)
		if id, _ := m["id"].(string); id != "" {
			ids = append(ids, id)
		}
	}
	return ids, status, nil
}

func getOrDeleteMCP(client *http.Client, base, token, id string, del bool) (int, error) {
	method := http.MethodGet
	path := base + "/api/v1/toolbox/mcp"
	if del {
		method = http.MethodDelete
		path = base + "/api/v1/toolbox/mcp/" + id
	} else {
		// no get-by-id; probe update as cross-access
		method = http.MethodPut
		path = base + "/api/v1/toolbox/mcp/" + id
		status, _, err := doJSON(client, method, path, token, map[string]any{
			"name": "hijack", "description": "x", "transport_type": "sse", "url": "https://example.com/x",
		})
		return status, err
	}
	status, _, err := doJSON(client, method, path, token, nil)
	return status, err
}

func createAgent(client *http.Client, base, token, name string, cfg map[string]any) (string, int, error) {
	status, body, err := doJSON(client, http.MethodPost, base+"/api/v1/agents", token, map[string]any{
		"name": name, "description": "isolation probe agent", "scope_type": "personal",
		"tenant_ids": []string{}, "config": cfg,
	})
	if err != nil {
		return "", status, err
	}
	data, _ := body["data"].(map[string]any)
	id, _ := data["id"].(string)
	return id, status, nil
}

func listAgents(client *http.Client, base, token string) ([]string, int, error) {
	status, body, err := doJSON(client, http.MethodGet, base+"/api/v1/agents", token, nil)
	if err != nil {
		return nil, status, err
	}
	items, _ := body["data"].([]any)
	ids := make([]string, 0, len(items))
	for _, item := range items {
		m, _ := item.(map[string]any)
		if id, _ := m["id"].(string); id != "" {
			ids = append(ids, id)
		}
	}
	return ids, status, nil
}

func deleteAgent(client *http.Client, base, token, id string) (int, error) {
	status, _, err := doJSON(client, http.MethodDelete, base+"/api/v1/agents/"+id, token, nil)
	return status, err
}

func createSession(client *http.Client, base, token, title, agentID string) (string, int, error) {
	body := map[string]any{"title": title}
	if agentID != "" {
		body["agent_id"] = agentID
	}
	status, resp, err := doJSON(client, http.MethodPost, base+"/api/v1/chat/sessions", token, body)
	if err != nil {
		return "", status, err
	}
	data, _ := resp["data"].(map[string]any)
	id, _ := data["id"].(string)
	return id, status, nil
}

func listSessions(client *http.Client, base, token string) ([]string, int, error) {
	status, body, err := doJSON(client, http.MethodGet, base+"/api/v1/chat/sessions?page=1&page_size=100", token, nil)
	if err != nil {
		return nil, status, err
	}
	data, _ := body["data"].(map[string]any)
	items, _ := data["items"].([]any)
	ids := make([]string, 0, len(items))
	for _, item := range items {
		m, _ := item.(map[string]any)
		if id, _ := m["id"].(string); id != "" {
			ids = append(ids, id)
		}
	}
	return ids, status, nil
}

func crossSession(client *http.Client, base, token, sessionID, kind string) (int, error) {
	switch kind {
	case "messages":
		status, _, err := doJSON(client, http.MethodGet, base+"/api/v1/chat/sessions/"+sessionID+"/messages?limit=5", token, nil)
		return status, err
	case "ask":
		req, err := http.NewRequest(http.MethodPost, base+"/api/v1/chat/sessions/"+sessionID+"/ask", bytes.NewReader(mustJSON(map[string]any{
			"query": "probe", "knowledge_base_ids": []string{},
		})))
		if err != nil {
			return 0, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return 0, err
		}
		defer resp.Body.Close()
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
		return resp.StatusCode, nil
	default:
		return 0, fmt.Errorf("unknown kind")
	}
}

func askOnce(client *http.Client, base, token, sessionID, query string) (int, string, error) {
	req, err := http.NewRequest(http.MethodPost, base+"/api/v1/chat/sessions/"+sessionID+"/ask", bytes.NewReader(mustJSON(map[string]any{
		"query": query, "knowledge_base_ids": []string{},
	})))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return resp.StatusCode, clip(string(raw), 120), nil
	}
	sc := bufio.NewScanner(resp.Body)
	var content strings.Builder
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "data:") {
			content.WriteString(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
			if content.Len() > 160 {
				break
			}
		}
	}
	return 200, clip(content.String(), 120), nil
}

func doJSON(client *http.Client, method, url, token string, body any) (int, map[string]any, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(mustJSON(body))
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return 0, nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	return resp.StatusCode, out, nil
}

func mustJSON(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}

func printCheck(name string, status int, err error, detail string) {
	if err != nil {
		fmt.Printf("CHECK %s err=%v detail=%s\n", name, err, detail)
		return
	}
	fmt.Printf("CHECK %s http=%d detail=%s\n", name, status, detail)
}

func contains(ids []string, id string) bool {
	if id == "" {
		return false
	}
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

func overlaps(have, want []string) bool {
	set := map[string]struct{}{}
	for _, id := range have {
		set[id] = struct{}{}
	}
	for _, id := range want {
		if _, ok := set[id]; ok {
			return true
		}
	}
	return false
}

func clip(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
