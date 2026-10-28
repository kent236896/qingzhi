package weknora

import (
	"context"
	"fmt"
)

// PersonalMemoryIsolationStatus documents whether personal memory APIs are
// safe to expose through the shared workspace API key + signed_token.
type PersonalMemoryIsolationStatus struct {
	Verified bool   `json:"verified"`
	Ready    bool   `json:"ready"`
	Reason   string `json:"reason"`
}

// PersonalMemoryIsolation reports the current integration posture.
// Official docs state API Key Principal currently isolates Session/MCP, not
// general RBAC or an explicit guarantee for /memory/* subject isolation under
// signed_token. Until a live deployment proves isolation, personal APIs stay pending.
func PersonalMemoryIsolation() PersonalMemoryIsolationStatus {
	return PersonalMemoryIsolationStatus{
		Verified: false,
		Ready:    false,
		Reason:   "未验证：公开文档未保证 signed_token 对 /memory/* 按终端用户隔离；禁止用超级管理员 API Key 冒充个人记忆授权",
	}
}

// GetMemorySettingsForCaller is intentionally unavailable until isolation is verified.
func (c *HTTPClient) GetMemorySettingsForCaller(ctx context.Context, externalUserToken string) error {
	_ = ctx
	_ = externalUserToken
	status := PersonalMemoryIsolation()
	return fmt.Errorf("%s", status.Reason)
}
