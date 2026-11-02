//go:build ignore

// Live A/B isolation check against the platform enterprise-api (same WeKnora API Key underneath).
// Env:
//   API_BASE (default http://127.0.0.1:8080)
//   ISOLATION_USER_A / ISOLATION_PASS_A
//   ISOLATION_USER_B / ISOLATION_PASS_B
//
// Usage: go run ./scripts/chat-isolation-retest.go
package main

import (
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

	client := &http.Client{Timeout: 45 * time.Second}
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

	fail := false
	sessA, status, err := createSession(client, base, tokA, "iso-a-"+fmt.Sprint(time.Now().Unix()))
	printCheck("a_create_session", status, err, sessA)
	if status != 200 || sessA == "" {
		fmt.Println("RESULT=FAIL cannot_create_session_a")
		os.Exit(1)
	}
	sessB, status, err := createSession(client, base, tokB, "iso-b-"+fmt.Sprint(time.Now().Unix()))
	printCheck("b_create_session", status, err, sessB)
	if status != 200 || sessB == "" {
		fmt.Println("RESULT=FAIL cannot_create_session_b")
		os.Exit(1)
	}

	listA, status, err := listSessions(client, base, tokA)
	printCheck("a_list_sessions", status, err, fmt.Sprintf("count=%d", len(listA)))
	if containsID(listA, sessB) {
		fmt.Println("VERDICT_DETAIL a_sees_b_session=true")
		fail = true
	} else {
		fmt.Println("VERDICT_DETAIL a_sees_b_session=false")
	}
	listB, status, err := listSessions(client, base, tokB)
	printCheck("b_list_sessions", status, err, fmt.Sprintf("count=%d", len(listB)))
	if containsID(listB, sessA) {
		fmt.Println("VERDICT_DETAIL b_sees_a_session=true")
		fail = true
	} else {
		fmt.Println("VERDICT_DETAIL b_sees_a_session=false")
	}

	for _, name := range []string{"messages", "rename", "delete", "stop", "ask"} {
		status, err := crossAccess(client, base, tokA, sessB, name)
		printCheck("a_cross_"+name, status, err, fmt.Sprintf("http=%d", status))
		if status == 200 {
			fmt.Printf("VERDICT_DETAIL a_cross_%s_allowed=true\n", name)
			fail = true
		} else if status == 404 || status == 403 {
			fmt.Printf("VERDICT_DETAIL a_cross_%s_denied=true http=%d\n", name, status)
		} else if status == 503 {
			fmt.Printf("VERDICT_DETAIL a_cross_%s_unavailable http=%d (WeKnora/HMAC?)\n", name, status)
		} else {
			fmt.Printf("VERDICT_DETAIL a_cross_%s_unexpected http=%d\n", name, status)
			fail = true
		}
	}

	// Knowledge: A must not see B personal KB if create succeeds.
	kbB, status, err := createPersonalKB(client, base, tokB, "iso-kb-b-"+fmt.Sprint(time.Now().Unix()))
	printCheck("b_create_personal_kb", status, err, kbB)
	if status == 200 && kbB != "" {
		status, body, err := getKB(client, base, tokA, kbB)
		printCheck("a_get_b_kb", status, err, fmt.Sprintf("http=%d", status))
		if status == 200 {
			fmt.Println("VERDICT_DETAIL a_sees_b_kb=true")
			fail = true
		} else if status == 404 {
			fmt.Println("VERDICT_DETAIL a_sees_b_kb=false")
		} else {
			fmt.Printf("VERDICT_DETAIL a_get_b_kb_unexpected http=%d body_keys=%v\n", status, topKeys(body))
			fail = true
		}
		_, _, _ = doJSON(client, http.MethodDelete, base+"/api/v1/knowledge-bases/"+kbB, tokB, nil)
	} else {
		fmt.Println("VERDICT_DETAIL kb_create_skipped_or_failed")
	}

	_, _, _ = doJSON(client, http.MethodDelete, base+"/api/v1/chat/sessions/"+sessA, tokA, nil)
	_, _, _ = doJSON(client, http.MethodDelete, base+"/api/v1/chat/sessions/"+sessB, tokB, nil)

	if fail {
		fmt.Println("RESULT=FAIL isolation_not_proven")
		os.Exit(1)
	}
	fmt.Println("RESULT=PASS platform_chat_kb_isolation")
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

func createSession(client *http.Client, base, token, title string) (string, int, error) {
	status, body, err := doJSON(client, http.MethodPost, base+"/api/v1/chat/sessions", token, map[string]any{"title": title})
	if err != nil {
		return "", status, err
	}
	data, _ := body["data"].(map[string]any)
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
		id, _ := m["id"].(string)
		if id != "" {
			ids = append(ids, id)
		}
	}
	return ids, status, nil
}

func crossAccess(client *http.Client, base, token, sessionID, kind string) (int, error) {
	switch kind {
	case "messages":
		status, _, err := doJSON(client, http.MethodGet, base+"/api/v1/chat/sessions/"+sessionID+"/messages?limit=10", token, nil)
		return status, err
	case "rename":
		status, _, err := doJSON(client, http.MethodPut, base+"/api/v1/chat/sessions/"+sessionID, token, map[string]any{"title": "hijack"})
		return status, err
	case "delete":
		status, _, err := doJSON(client, http.MethodDelete, base+"/api/v1/chat/sessions/"+sessionID, token, nil)
		return status, err
	case "stop":
		status, _, err := doJSON(client, http.MethodPost, base+"/api/v1/chat/sessions/"+sessionID+"/stop", token, map[string]any{"message_id": "fake"})
		return status, err
	case "ask":
		// Prefer empty KB list → 400; if somehow passes ownership, still must fail.
		req, err := http.NewRequest(http.MethodPost, base+"/api/v1/chat/sessions/"+sessionID+"/ask", bytes.NewReader(mustJSON(map[string]any{
			"query": "probe", "knowledge_base_ids": []string{"00000000-0000-0000-0000-000000000000"},
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
		_, _ = io.Copy(io.Discard, resp.Body)
		return resp.StatusCode, nil
	default:
		return 0, fmt.Errorf("unknown kind")
	}
}

func createPersonalKB(client *http.Client, base, token, name string) (string, int, error) {
	status, body, err := doJSON(client, http.MethodPost, base+"/api/v1/knowledge-bases", token, map[string]any{
		"name": name, "description": "isolation probe", "scope_type": "personal",
	})
	if err != nil {
		return "", status, err
	}
	data, _ := body["data"].(map[string]any)
	id, _ := data["id"].(string)
	return id, status, nil
}

func getKB(client *http.Client, base, token, id string) (int, map[string]any, error) {
	return doJSON(client, http.MethodGet, base+"/api/v1/knowledge-bases/"+id, token, nil)
}

func containsID(ids []string, id string) bool {
	for _, item := range ids {
		if item == id {
			return true
		}
	}
	return false
}

func printCheck(name string, status int, err error, detail string) {
	if err != nil {
		fmt.Printf("CHECK %s err=%v detail=%s\n", name, err, detail)
		return
	}
	fmt.Printf("CHECK %s http=%d detail=%s\n", name, status, detail)
}

func doJSON(client *http.Client, method, url, token string, payload any) (int, map[string]any, error) {
	var body io.Reader
	if payload != nil {
		body = bytes.NewReader(mustJSON(payload))
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return 0, nil, err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if payload != nil {
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
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func topKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
