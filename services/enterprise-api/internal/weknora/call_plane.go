package weknora

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Call plane: Admin uses ExchangeAdmin (tenant-admin JWT); User uses Exchange /
// ExchangeUser (API Key + X-External-User-Token). Never mix Admin Bearer into
// session/chat/personal-memory/oauth-user calls.
//
// See docs/weknora-call-planes.md.

var (
	// ErrUserPlaneForbidden is returned when Exchange would hit an admin-only mutation.
	ErrUserPlaneForbidden = errors.New("weknora: 该操作仅允许管理面（Admin JWT）调用")
	// ErrExternalUserTokenRequired is returned when a user-plane call lacks a signed_token.
	ErrExternalUserTokenRequired = errors.New("weknora: 缺少 X-External-User-Token")
)

// RequireExternalUserToken ensures ctx carries a terminal-user token.
func RequireExternalUserToken(ctx context.Context) error {
	if strings.TrimSpace(ExternalUserTokenFrom(ctx)) == "" {
		return ErrExternalUserTokenRequired
	}
	return nil
}

// AssertUserPlaneAllowed rejects WeKnora paths that must not be issued with only
// API Key + External Token from Qingzhi (space-wide Agent/MCP definition writes
// and shared credentials). GET listing remains allowed for diagnostics but ACL
// truth stays in Qingzhi catalog tables.
func AssertUserPlaneAllowed(method, path string) error {
	method = strings.ToUpper(strings.TrimSpace(method))
	path = normalizeWeKnoraPath(path)
	if method == "" || path == "" {
		return fmt.Errorf("weknora: 缺少 method 或 path")
	}
	// Tool approvals are out of Qingzhi scope: never proxy from either plane via Exchange.
	if isMCPToolApprovalPath(path) {
		return fmt.Errorf("%w: %s %s", ErrUserPlaneForbidden, method, path)
	}
	if method == http.MethodGet || method == http.MethodHead {
		return nil
	}
	if isAgentDefinitionMutation(method, path) {
		return fmt.Errorf("%w: %s %s", ErrUserPlaneForbidden, method, path)
	}
	if isMCPDefinitionMutation(method, path) {
		return fmt.Errorf("%w: %s %s", ErrUserPlaneForbidden, method, path)
	}
	if isMCPSharedCredentialMutation(method, path) {
		return fmt.Errorf("%w: %s %s", ErrUserPlaneForbidden, method, path)
	}
	return nil
}

func normalizeWeKnoraPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if i := strings.IndexByte(path, '?'); i >= 0 {
		path = path[:i]
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if len(path) > 1 {
		path = strings.TrimRight(path, "/")
	}
	return path
}

func isAgentDefinitionMutation(method, path string) bool {
	if path == "/api/v1/agents" {
		return method == http.MethodPost
	}
	if !strings.HasPrefix(path, "/api/v1/agents/") {
		return false
	}
	rest := strings.TrimPrefix(path, "/api/v1/agents/")
	// /agents/{id} or /agents/{id}/copy
	switch method {
	case http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	case http.MethodPost:
		return strings.HasSuffix(rest, "/copy") || !strings.Contains(rest, "/")
	default:
		return false
	}
}

func isMCPDefinitionMutation(method, path string) bool {
	if path == "/api/v1/mcp-services" {
		return method == http.MethodPost
	}
	if !strings.HasPrefix(path, "/api/v1/mcp-services/") {
		return false
	}
	rest := strings.TrimPrefix(path, "/api/v1/mcp-services/")
	// Allow user OAuth under a service id. Tool approvals are rejected earlier.
	if strings.Contains(rest, "/oauth/") || strings.HasSuffix(rest, "/oauth") {
		return false
	}
	if strings.Contains(rest, "/credentials") {
		return false // handled by isMCPSharedCredentialMutation
	}
	switch method {
	case http.MethodPut, http.MethodPatch, http.MethodDelete:
		// /mcp-services/{id} definition replace/delete
		return !strings.Contains(rest, "/")
	case http.MethodPost:
		// create already handled; block test / metadata refresh / sync on user plane
		return strings.HasSuffix(rest, "/test") ||
			strings.HasSuffix(rest, "/metadata/refresh") ||
			strings.HasSuffix(rest, "/sync") ||
			!strings.Contains(rest, "/")
	default:
		return false
	}
}

func isMCPToolApprovalPath(path string) bool {
	return strings.Contains(path, "/tool-approvals") || strings.Contains(path, "/tools/approvals")
}

func isMCPSharedCredentialMutation(method, path string) bool {
	if !strings.Contains(path, "/mcp-services/") || !strings.Contains(path, "/credentials") {
		return false
	}
	switch method {
	case http.MethodPut, http.MethodPatch, http.MethodPost, http.MethodDelete:
		return true
	default:
		return false
	}
}
