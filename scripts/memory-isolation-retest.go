//go:build ignore

// One-off live retest: shared WeKnora API Key + two X-External-User-Token subjects.
// Prints only verdicts (no secrets, no memory content dumps).
//
// Usage (from repo root):
//   go run ./scripts/memory-isolation-retest.go
//
// Loads ./.env or ../.env when present; existing process env wins.
//
// Added: Step 0 sets the tenant's api-principal-config to signed_token with
// WEKNORA_EXTERNAL_USER_HMAC before the isolation checks run.
// Optional env:
//   WEKNORA_ADMIN_TOKEN             tenant-admin JWT (NOT the API key). Prefer login below.
//   WEKNORA_ADMIN_EMAIL             WeKnora admin email for POST /api/v1/auth/login
//   WEKNORA_ADMIN_PASSWORD          WeKnora admin password (used when ADMIN_TOKEN empty)
//   WEKNORA_SKIP_PRINCIPAL_CONFIG=1 skip Step 0
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
	"path/filepath"
	"strings"
	"time"
)

func main() {
	loadDotEnv()
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("WEKNORA_BASE_URL")), "/")
	apiKey := strings.TrimSpace(os.Getenv("WEKNORA_API_KEY"))
	hmacSecret := strings.TrimSpace(os.Getenv("WEKNORA_EXTERNAL_USER_HMAC"))
	adminToken := strings.TrimSpace(os.Getenv("WEKNORA_ADMIN_TOKEN"))
	adminEmail := strings.TrimSpace(os.Getenv("WEKNORA_ADMIN_EMAIL"))
	adminPassword := strings.TrimSpace(os.Getenv("WEKNORA_ADMIN_PASSWORD"))
	tenantID := strings.TrimSpace(os.Getenv("WEKNORA_TENANT_ID"))
	if tenantID == "" {
		tenantID = "10000"
	}
	if base == "" || apiKey == "" || hmacSecret == "" {
		fmt.Println("RESULT=SKIP reason=missing WEKNORA_BASE_URL or WEKNORA_API_KEY or WEKNORA_EXTERNAL_USER_HMAC")
		fmt.Println("HINT: run from repo root so .env can be loaded, or export those variables first")
		os.Exit(2)
	}

	tokenA, err := signToken(hmacSecret, "retest-user-a", tenantID)
	if err != nil {
		fmt.Println("RESULT=SKIP reason=sign_token_a_failed")
		os.Exit(2)
	}
	tokenB, err := signToken(hmacSecret, "retest-user-b", tenantID)
	if err != nil {
		fmt.Println("RESULT=SKIP reason=sign_token_b_failed")
		os.Exit(2)
	}

	client := &http.Client{Timeout: 20 * time.Second}
	fail := false

	// 0) Configure principal mode = signed_token on the WeKnora side (new)
	if os.Getenv("WEKNORA_SKIP_PRINCIPAL_CONFIG") != "1" {
		if adminToken == "" && adminEmail != "" && adminPassword != "" {
			tok, loginErr := loginAdmin(client, base, adminEmail, adminPassword)
			if loginErr != nil {
				fmt.Printf("STEP0 admin_login err=%s\n", loginErr.Error())
				fmt.Println("RESULT=SKIP reason=admin_login_failed")
				os.Exit(2)
			}
			adminToken = tok
			fmt.Println("STEP0 admin_login ok (token from email/password; not API key)")
		}
		if !configurePrincipal(client, base, apiKey, adminToken, tenantID, hmacSecret) {
			os.Exit(2)
		}
	} else {
		fmt.Println("STEP0 skipped (WEKNORA_SKIP_PRINCIPAL_CONFIG=1)")
	}

	// 1) No external token
	statusNo, bodyNo, err := doJSON(client, http.MethodGet, base+"/api/v1/memory/settings", apiKey, "", nil)
	printCheck("no_token_get_settings", statusNo, err, fmt.Sprintf("http=%d body_keys=%v", statusNo, topKeys(bodyNo)))
	if statusNo == 200 {
		fmt.Println("VERDICT_DETAIL no_token_accepted=true (falls back to API Key principal — isolation risk)")
		fail = true
	}

	// 2) Personal settings isolation: A=false, B=true; each GET must keep own value.
	statusPutA, bodyPutA, err := doJSON(client, http.MethodPut, base+"/api/v1/memory/settings", apiKey, tokenA, map[string]any{
		"enabled": false, "user_enabled": false,
	})
	printCheck("user_a_put_settings_false", statusPutA, err, fmt.Sprintf("http=%d user_enabled=%v", statusPutA, field(bodyPutA, "user_enabled")))

	statusPutB, bodyPutB, err := doJSON(client, http.MethodPut, base+"/api/v1/memory/settings", apiKey, tokenB, map[string]any{
		"enabled": true, "user_enabled": true,
	})
	printCheck("user_b_put_settings_true", statusPutB, err, fmt.Sprintf("http=%d user_enabled=%v", statusPutB, field(bodyPutB, "user_enabled")))

	statusGetA, bodyGetA, err := doJSON(client, http.MethodGet, base+"/api/v1/memory/settings", apiKey, tokenA, nil)
	printCheck("user_a_get_settings", statusGetA, err, fmt.Sprintf("http=%d user_enabled=%v", statusGetA, field(bodyGetA, "user_enabled")))
	statusGetB, bodyGetB, err := doJSON(client, http.MethodGet, base+"/api/v1/memory/settings", apiKey, tokenB, nil)
	printCheck("user_b_get_settings", statusGetB, err, fmt.Sprintf("http=%d user_enabled=%v", statusGetB, field(bodyGetB, "user_enabled")))
	if statusPutA == 200 && statusPutB == 200 && statusGetA == 200 && statusGetB == 200 {
		aEnabled := boolField(bodyGetA, "user_enabled")
		bEnabled := boolField(bodyGetB, "user_enabled")
		if aEnabled == bEnabled {
			fmt.Printf("VERDICT_DETAIL settings_shared_across_users=true a=%v b=%v\n", aEnabled, bEnabled)
			fail = true
		} else if !aEnabled && bEnabled {
			fmt.Println("VERDICT_DETAIL settings_shared_across_users=false")
		} else {
			fmt.Printf("VERDICT_DETAIL settings_unexpected a=%v b=%v\n", aEnabled, bEnabled)
			fail = true
		}
	} else {
		fmt.Println("VERDICT_DETAIL settings_check_incomplete")
		fail = true
	}

	// 3) A creates item; B lists (re-enable A after the settings toggle above)
	statusReA, _, err := doJSON(client, http.MethodPut, base+"/api/v1/memory/settings", apiKey, tokenA, map[string]any{
		"enabled": true, "user_enabled": true,
	})
	printCheck("user_a_reenable_before_create", statusReA, err, fmt.Sprintf("http=%d", statusReA))
	marker := fmt.Sprintf("isolation-probe-%d", time.Now().Unix())
	createBody := map[string]any{
		"type":     "fact",
		"content":  marker,
		"priority": 1,
	}
	statusCreate, bodyCreate, err := doJSON(client, http.MethodPost, base+"/api/v1/memory/items", apiKey, tokenA, createBody)
	printCheck("user_a_create_item", statusCreate, err, fmt.Sprintf("http=%d keys=%v subject=%v", statusCreate, topKeys(bodyCreate), field(bodyCreate, "subject_id")))
	itemID := stringField(bodyCreate, "id")
	if itemID == "" {
		itemID = stringField(bodyCreate, "memory_id")
	}

	statusListB, bodyListB, err := doJSON(client, http.MethodGet, base+"/api/v1/memory/items?page=1&page_size=50", apiKey, tokenB, nil)
	printCheck("user_b_list_items", statusListB, err, fmt.Sprintf("http=%d keys=%v", statusListB, topKeys(bodyListB)))
	if statusCreate == 200 && statusListB == 200 {
		if bodyContains(bodyListB, marker) || (itemID != "" && bodyContains(bodyListB, itemID)) {
			fmt.Println("VERDICT_DETAIL items_visible_across_users=true")
			subj := stringField(bodyCreate, "subject_id")
			fmt.Printf("VERDICT_DETAIL create_subject_id=%s\n", sanitize(subj))
			fail = true
		} else {
			fmt.Println("VERDICT_DETAIL items_visible_across_users=false")
		}
	}

	// 4) B tries delete A's item
	if itemID != "" {
		statusDel, _, err := doJSON(client, http.MethodDelete, base+"/api/v1/memory/items/"+itemID, apiKey, tokenB, nil)
		printCheck("user_b_delete_a_item", statusDel, err, fmt.Sprintf("http=%d", statusDel))
		if statusDel == 200 || statusDel == 204 {
			fmt.Println("VERDICT_DETAIL cross_user_delete_allowed=true")
			fail = true
		} else {
			fmt.Println("VERDICT_DETAIL cross_user_delete_allowed=false")
		}
		// cleanup as A
		_, _, _ = doJSON(client, http.MethodDelete, base+"/api/v1/memory/items/"+itemID, apiKey, tokenA, nil)
	}

	if fail {
		fmt.Println("RESULT=FAIL isolation_not_proven")
		os.Exit(1)
	}
	fmt.Println("RESULT=PASS isolation_ok")
}

func loginAdmin(client *http.Client, base, email, password string) (string, error) {
	raw, _ := json.Marshal(map[string]any{"email": email, "password": password})
	req, err := http.NewRequest(http.MethodPost, base+"/api/v1/auth/login", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("http=%d", resp.StatusCode)
	}
	out := map[string]any{}
	_ = json.Unmarshal(body, &out)
	if data, ok := out["data"].(map[string]any); ok {
		if tok, _ := data["token"].(string); strings.TrimSpace(tok) != "" {
			return strings.TrimSpace(tok), nil
		}
	}
	if tok, _ := out["token"].(string); strings.TrimSpace(tok) != "" {
		return strings.TrimSpace(tok), nil
	}
	return "", fmt.Errorf("missing token in login response")
}

// configurePrincipal sets the tenant API principal mode to signed_token and reads it back.
// Returns false when the config could not be applied (the retest would be meaningless).
func configurePrincipal(client *http.Client, base, apiKey, adminToken, tenantID, hmacSecret string) bool {
	url := base + "/api/v1/tenants/" + tenantID + "/api-principal-config"

	st, body, err := doAdmin(client, http.MethodGet, url, apiKey, adminToken, nil)
	if err != nil {
		fmt.Printf("STEP0 get_principal_config err=%s\n", err.Error())
	} else {
		fmt.Printf("STEP0 get_principal_config http=%d mode=%s\n", st, sanitize(stringField(body, "mode")))
	}

	payload := map[string]any{"mode": "signed_token", "hmac_secret": hmacSecret}
	st, body, err = doAdmin(client, http.MethodPut, url, apiKey, adminToken, payload)
	if err != nil {
		fmt.Printf("STEP0 put_principal_config err=%s\n", err.Error())
		fmt.Println("RESULT=SKIP reason=principal_config_request_failed")
		return false
	}
	fmt.Printf("STEP0 put_principal_config http=%d keys=%v\n", st, topKeys(body))
	if st != 200 && st != 201 && st != 204 {
		fmt.Println("RESULT=SKIP reason=principal_config_rejected")
		fmt.Println("HINT: 401/403 -> set WEKNORA_ADMIN_EMAIL/PASSWORD (login) or WEKNORA_ADMIN_TOKEN (JWT, not API key); 404 -> principal config API unsupported")
		return false
	}

	st, body, err = doAdmin(client, http.MethodGet, url, apiKey, adminToken, nil)
	if err == nil && st == 200 {
		mode := stringField(body, "mode")
		fmt.Printf("STEP0 verify_principal_config http=%d mode=%s\n", st, sanitize(mode))
		if mode != "" && mode != "signed_token" {
			fmt.Println("RESULT=SKIP reason=principal_mode_not_applied")
			return false
		}
	} else {
		fmt.Printf("STEP0 verify_principal_config http=%d (could not read back; continuing)\n", st)
	}
	return true
}

func doAdmin(client *http.Client, method, url, apiKey, adminToken string, body any) (int, map[string]any, error) {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("Accept", "application/json")
	if adminToken != "" {
		req.Header.Set("Authorization", "Bearer "+adminToken)
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
	if data, ok := out["data"].(map[string]any); ok {
		out = data
	}
	return resp.StatusCode, out, nil
}

func printCheck(name string, status int, err error, detail string) {
	if err != nil {
		fmt.Printf("CHECK %s err=%s\n", name, err.Error())
		return
	}
	fmt.Printf("CHECK %s %s\n", name, detail)
}

func doJSON(client *http.Client, method, url, apiKey, externalToken string, body any) (int, map[string]any, error) {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if externalToken != "" {
		req.Header.Set("X-External-User-Token", externalToken)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	// Some APIs wrap in data
	if data, ok := out["data"].(map[string]any); ok {
		out = data
	}
	return resp.StatusCode, out, nil
}

func signToken(secret, sub, tenantID string) (string, error) {
	now := time.Now().UTC()
	claims := map[string]any{
		"sub":       sub,
		"tenant_id": tenantID,
		"aud":       "weknora",
		"iat":       now.Unix(),
		"exp":       now.Add(10 * time.Minute).Unix(),
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	body := base64.RawURLEncoding.EncodeToString(payload)
	signing := header + "." + body
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(signing))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return signing + "." + sig, nil
}

func topKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func field(m map[string]any, key string) any {
	if m == nil {
		return nil
	}
	return m[key]
}

func stringField(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func boolField(m map[string]any, key string) bool {
	v, _ := m[key].(bool)
	return v
}

func bodyContains(m map[string]any, needle string) bool {
	raw, _ := json.Marshal(m)
	return strings.Contains(string(raw), needle)
}

func sanitize(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "(empty)"
	}
	if len(s) > 64 {
		return s[:64] + "…"
	}
	return s
}

func loadDotEnv() {
	candidates := []string{".env", filepath.Join("..", ".env")}
	for _, path := range candidates {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(raw), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			key, val, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			key = strings.TrimSpace(key)
			val = strings.TrimSpace(val)
			if key == "" {
				continue
			}
			if len(val) >= 2 {
				if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
					val = val[1 : len(val)-1]
				}
			}
			if os.Getenv(key) == "" {
				_ = os.Setenv(key, val)
			}
		}
		return
	}
}