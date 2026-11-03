package weknora

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestAssertUserPlaneAllowed_BlocksDangerousWrites(t *testing.T) {
	cases := []struct {
		method string
		path   string
	}{
		{http.MethodDelete, "/api/v1/agents/abc"},
		{http.MethodPost, "/api/v1/agents"},
		{http.MethodPost, "/api/v1/agents/abc/copy"},
		{http.MethodPut, "/api/v1/agents/abc"},
		{http.MethodDelete, "/api/v1/mcp-services/abc"},
		{http.MethodPost, "/api/v1/mcp-services"},
		{http.MethodPut, "/api/v1/mcp-services/abc"},
		{http.MethodPost, "/api/v1/mcp-services/abc/test"},
		{http.MethodPut, "/api/v1/mcp-services/abc/credentials"},
		{http.MethodDelete, "/api/v1/mcp-services/abc/credentials/api_key"},
		{http.MethodGet, "/api/v1/mcp-services/abc/tool-approvals"},
		{http.MethodPost, "/api/v1/mcp-services/abc/tool-approvals"},
		{http.MethodPut, "/api/v1/mcp-services/abc/tools/approvals"},
	}
	for _, tc := range cases {
		err := AssertUserPlaneAllowed(tc.method, tc.path)
		if !errors.Is(err, ErrUserPlaneForbidden) {
			t.Fatalf("%s %s: want ErrUserPlaneForbidden, got %v", tc.method, tc.path, err)
		}
	}
}

func TestAssertUserPlaneAllowed_AllowsUserOAuthAndReads(t *testing.T) {
	cases := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/v1/agents"},
		{http.MethodGet, "/api/v1/mcp-services/abc/oauth/status"},
		{http.MethodPost, "/api/v1/mcp-services/abc/oauth/authorize-url"},
		{http.MethodDelete, "/api/v1/mcp-services/abc/oauth/token"},
		{http.MethodPost, "/api/v1/knowledge-chat/sid"},
		{http.MethodPost, "/api/v1/agent-chat/sid"},
		{http.MethodGet, "/api/v1/memory/settings"},
	}
	for _, tc := range cases {
		if err := AssertUserPlaneAllowed(tc.method, tc.path); err != nil {
			t.Fatalf("%s %s: unexpected %v", tc.method, tc.path, err)
		}
	}
}

func TestRequireExternalUserToken(t *testing.T) {
	if err := RequireExternalUserToken(context.Background()); !errors.Is(err, ErrExternalUserTokenRequired) {
		t.Fatalf("got %v", err)
	}
	ctx := WithExternalUserToken(context.Background(), "tok")
	if err := RequireExternalUserToken(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestExchange_RejectsAdminMutationWithoutNetwork(t *testing.T) {
	c := NewHTTPClient("http://127.0.0.1:9", "key", 0)
	_, _, err := c.Exchange(context.Background(), http.MethodDelete, "/api/v1/agents/x", nil)
	if !errors.Is(err, ErrUserPlaneForbidden) {
		t.Fatalf("got %v", err)
	}
}

func TestExchangeUser_RequiresToken(t *testing.T) {
	c := NewHTTPClient("http://127.0.0.1:9", "key", 0)
	_, _, err := c.ExchangeUser(context.Background(), http.MethodGet, "/api/v1/mcp-services/x/oauth/status", nil)
	if !errors.Is(err, ErrExternalUserTokenRequired) {
		t.Fatalf("got %v", err)
	}
}

func TestMCPCallbackRedirectURI(t *testing.T) {
	if got := MCPCallbackRedirectURI("http://wk:8083/"); got != "http://wk:8083/api/v1/mcp-oauth/callback" {
		t.Fatalf("got %q", got)
	}
	if got := MCPCallbackRedirectURI(""); got != "" {
		t.Fatalf("empty base got %q", got)
	}
}
