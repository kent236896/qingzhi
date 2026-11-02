package weknora

import "testing"

func TestPersonalMemoryIsolationGateOpen(t *testing.T) {
	status := PersonalMemoryIsolation()
	if !status.Ready {
		t.Fatal("personal memory must be open after 2026-10-30 isolation PASS")
	}
	if !status.Verified {
		t.Fatal("expected verified=true")
	}
	if status.RetestedAt == "" {
		t.Fatal("expected retested_at")
	}
	if status.Reason == "" {
		t.Fatal("expected reason")
	}
}
