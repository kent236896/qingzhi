package security

import "testing"

func TestModuleAndActionOfCatalogChat(t *testing.T) {
	cases := []struct {
		method, path, module, action string
	}{
		{"POST", "/api/v1/agents", "智能体", "新增"},
		{"POST", "/api/v1/agents/a1/copy", "智能体", "复制"},
		{"PUT", "/api/v1/toolbox/mcp/m1/credentials", "MCP工具箱", "密钥"},
		{"POST", "/api/v1/toolbox/mcp/m1/oauth/authorize-url", "MCP工具箱", "OAuth授权"},
		{"DELETE", "/api/v1/toolbox/mcp/m1/oauth/token", "MCP工具箱", "OAuth撤销"},
		{"POST", "/api/v1/toolbox/mcp/m1/test", "MCP工具箱", "测试"},
		{"POST", "/api/v1/toolbox/mcp/m1/metadata/refresh", "MCP工具箱", "刷新元数据"},
		{"POST", "/api/v1/chat/sessions", "对话", "新增"},
		{"POST", "/api/v1/chat/sessions/s1/ask", "对话", "问答"},
		{"POST", "/api/v1/chat/sessions/s1/stop", "对话", "停止生成"},
		{"POST", "/api/v1/knowledge-bases", "知识库", "新增"},
	}
	for _, tc := range cases {
		if got := moduleOf(tc.path); got != tc.module {
			t.Fatalf("%s %s module=%q want %q", tc.method, tc.path, got, tc.module)
		}
		if got := actionOf(tc.method, tc.path); got != tc.action {
			t.Fatalf("%s %s action=%q want %q", tc.method, tc.path, got, tc.action)
		}
	}
}
