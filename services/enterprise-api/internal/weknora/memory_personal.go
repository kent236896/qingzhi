package weknora

// PersonalMemoryIsolationStatus documents whether personal memory APIs are
// safe to expose through the shared workspace API key + signed_token.
type PersonalMemoryIsolationStatus struct {
	Verified   bool   `json:"verified"`
	Ready      bool   `json:"ready"`
	Reason     string `json:"reason"`
	RetestedAt string `json:"retested_at,omitempty"`
}

// PersonalMemoryIsolation reports the current integration posture after live retests.
// 2026-10-30: after setting api-principal-config.mode=signed_token (via admin JWT),
// scripts/memory-isolation-retest.go returned RESULT=PASS isolation_ok.
func PersonalMemoryIsolation() PersonalMemoryIsolationStatus {
	return PersonalMemoryIsolationStatus{
		Verified:   true,
		Ready:      true,
		RetestedAt: "2026-10-30T10:05+08:00",
		Reason: "实机复测通过（2026-10-30）：api-principal-config=signed_token 后，" +
			"缺 Token 访问 /memory/settings 返回 401；A/B 个人开关互不影响；" +
			"条目 subject_id=api_external_user:<空间>:<sub>；跨用户列表不可见且删除返回 404。" +
			"开放个人记忆代理时必须注入 X-External-User-Token，禁止回退管理员主体。",
	}
}
