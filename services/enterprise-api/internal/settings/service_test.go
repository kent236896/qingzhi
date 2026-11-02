package settings

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

type memoryStore struct{ item Record }

func (s *memoryStore) Get(context.Context) (Record, error) { return s.item, nil }

func (s *memoryStore) Save(_ context.Context, item Record) error {
	s.item = item
	return nil
}

func TestEnsureDoesNotOverridePageSettings(t *testing.T) {
	store := &memoryStore{item: Record{BaseURL: "http://saved.example", APIKey: "saved-key"}}
	svc := NewService(store, nil)
	got, err := svc.Ensure(context.Background(), "http://env.example", "env-key", "10001")
	if err != nil {
		t.Fatal(err)
	}
	if got.BaseURL != "http://saved.example" || got.APIKey != "saved-key" {
		t.Fatalf("stored settings were replaced: %#v", got)
	}
}

func TestEnsureSeedsEmptyRowFromEnv(t *testing.T) {
	store := &memoryStore{}
	svc := NewService(store, nil)
	got, err := svc.Ensure(context.Background(), "http://env.example/", "env-key", "10000")
	if err != nil {
		t.Fatal(err)
	}
	if got.BaseURL != "http://env.example" || got.APIKey != "env-key" || got.UpdatedBy != "环境变量" {
		t.Fatalf("seed = %#v", got)
	}
	if got.WeKnoraTenantID != "10000" {
		t.Fatalf("tenant id = %q", got.WeKnoraTenantID)
	}
}

func TestEnsureSeedsDefaultTenantIDWhenBlank(t *testing.T) {
	store := &memoryStore{item: Record{BaseURL: "http://saved.example", APIKey: "saved-key"}}
	svc := NewService(store, nil)
	got, err := svc.Ensure(context.Background(), "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if got.WeKnoraTenantID != DefaultWeKnoraTenantID {
		t.Fatalf("tenant id = %q", got.WeKnoraTenantID)
	}
}

func TestUpdateKeepsKeyAndHidesIt(t *testing.T) {
	const secret = "sk-page-secret-value"
	const adminPass = "admin-secret-pass"
	store := &memoryStore{item: Record{
		BaseURL: "http://old.example", APIKey: secret,
		AdminEmail: "admin@example.com", AdminPassword: adminPass,
	}}
	var applied string
	svc := NewService(store, nil)
	svc.now = func() time.Time { return time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC) }
	svc.SetHooks(Hooks{
		Apply: func(baseURL, apiKey, adminEmail, adminPassword string) {
			applied = baseURL + "|" + apiKey + "|" + adminEmail + "|" + adminPassword
		},
		Probe: func(context.Context) (string, error) { return "0.8.2", nil },
	})
	view, err := svc.Update(context.Background(), Update{BaseURL: "http://new.example", Updater: "admin"})
	if err != nil {
		t.Fatal(err)
	}
	if view.BaseURL != "http://new.example" || !view.APIKeyConfigured || !view.Reachable || view.Version != "0.8.2" {
		t.Fatalf("view = %#v", view)
	}
	if view.AdminEmail != "admin@example.com" || !view.AdminPasswordConfigured {
		t.Fatalf("admin view = %#v", view)
	}
	raw, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), secret) || strings.Contains(string(raw), adminPass) {
		t.Fatalf("response leaked secrets: %s", raw)
	}
	if applied != "http://new.example|"+secret+"|admin@example.com|"+adminPass {
		t.Fatalf("apply = %s", applied)
	}
}

func TestEnsurePrincipalRequiresAdminCredentials(t *testing.T) {
	svc := NewService(&memoryStore{item: Record{BaseURL: "http://w.example", APIKey: "k"}}, nil)
	svc.SetHooks(Hooks{
		EnsurePrincipal: func(context.Context, string, string, string, string, string) (string, error) {
			return "signed_token", nil
		},
	})
	_, err := svc.EnsureSignedTokenPrincipal(context.Background())
	if err == nil || !strings.Contains(err.Error(), "管理员") {
		t.Fatalf("err = %v", err)
	}
}

func TestUpdateRejectsAPIPath(t *testing.T) {
	svc := NewService(&memoryStore{}, nil)
	_, err := svc.Update(context.Background(), Update{BaseURL: "http://example.test/api/v1", APIKey: "key"})
	if err == nil || !strings.Contains(err.Error(), "/api/v1") {
		t.Fatalf("err = %v", err)
	}
}

func TestUpdateRequiresKeyWhenMissing(t *testing.T) {
	svc := NewService(&memoryStore{}, nil)
	_, err := svc.Update(context.Background(), Update{BaseURL: "http://example.test"})
	if err == nil || !strings.Contains(err.Error(), "API Key") {
		t.Fatalf("err = %v", err)
	}
}
