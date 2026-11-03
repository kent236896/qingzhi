package catalog

import "testing"

func TestNormalizeMCPAuthMode(t *testing.T) {
	if got := NormalizeMCPAuthMode(""); got != MCPAuthModeShared {
		t.Fatalf("empty = %q", got)
	}
	if got := NormalizeMCPAuthMode("SHARED"); got != MCPAuthModeShared {
		t.Fatalf("SHARED = %q", got)
	}
	if got := NormalizeMCPAuthMode("user_oauth"); got != MCPAuthModeUserOAuth {
		t.Fatalf("user_oauth = %q", got)
	}
	if got := NormalizeMCPAuthMode("other"); got != MCPAuthModeShared {
		t.Fatalf("other = %q", got)
	}
}

func TestResolveMCPAuthMode(t *testing.T) {
	svc := &Service{}
	if got := svc.ResolveMCPAuthMode(Item{}); got != MCPAuthModeShared {
		t.Fatalf("got %q", got)
	}
	if got := svc.ResolveMCPAuthMode(Item{AuthMode: MCPAuthModeUserOAuth}); got != MCPAuthModeUserOAuth {
		t.Fatalf("got %q", got)
	}
}
