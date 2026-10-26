package sso

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"qingzhi/enterprise-api/internal/identity"
)

type stubIdentity struct {
	loginFn   func(ctx context.Context, userID, tenantID, loginIP string) (identity.LoginResult, error)
	createFn  func(ctx context.Context, username, nickname, email, tenantID string) (identity.User, error)
	created   []identity.User
	loginCalls int
}

func (s *stubIdentity) LoginFederated(ctx context.Context, userID, tenantID, loginIP string) (identity.LoginResult, error) {
	s.loginCalls++
	if s.loginFn != nil {
		return s.loginFn(ctx, userID, tenantID, loginIP)
	}
	return identity.LoginResult{
		User:     identity.User{ID: userID, Username: "sso_user", Nickname: "SSO", Status: identity.StatusNormal},
		TenantID: tenantID,
		Access: identity.SessionAccess{
			Permissions: []string{},
			Access:      identity.DataAccess{Scope: identity.DataScopeSelf},
		},
	}, nil
}

func (s *stubIdentity) CreateFederatedUser(ctx context.Context, username, nickname, email, tenantID string) (identity.User, error) {
	if s.createFn != nil {
		return s.createFn(ctx, username, nickname, email, tenantID)
	}
	user := identity.User{ID: newID(), Username: username, Nickname: nickname, Email: email, Status: identity.StatusNormal}
	s.created = append(s.created, user)
	return user, nil
}

func TestSanitizeRedirectPath(t *testing.T) {
	cases := map[string]string{
		"":                    "/",
		"/dashboard":          "/dashboard",
		"//evil.com":          "/",
		"https://evil.com":    "/",
		"/ok?x=1":             "/ok?x=1",
		"dashboard":           "/",
		"/path\nLocation: x":  "/",
	}
	for in, want := range cases {
		if got := sanitizeRedirectPath(in); got != want {
			t.Fatalf("sanitizeRedirectPath(%q)=%q want %q", in, got, want)
		}
	}
}

func TestFederatedUsernameStableAndSafe(t *testing.T) {
	a := federatedUsername("provider-1", "subject-a")
	b := federatedUsername("provider-1", "subject-a")
	c := federatedUsername("provider-2", "subject-a")
	if a != b {
		t.Fatal("username should be stable")
	}
	if a == c {
		t.Fatal("different providers must not collide")
	}
	if !strings.HasPrefix(a, "sso_") || len(a) > 30 || len(a) < 3 {
		t.Fatalf("username out of range: %q", a)
	}
}

func TestResolveMappedUserRequiresLinkWithoutAutoCreate(t *testing.T) {
	store := NewMemoryStore()
	vault, _ := NewVault("test-sso-secrets-key-32chars-min!", "")
	svc := NewService(store, vault)
	svc.SetIdentity(&stubIdentity{})
	ctx := context.Background()
	provider, err := svc.CreateProvider(ctx, ProviderInput{
		Name: "Keycloak", Protocol: ProtocolOIDC, Status: StatusEnabled,
		Issuer: "https://idp.example.com", ClientID: "web",
		ClientSecret: "secret-value-1234567890", RedirectURI: "http://127.0.0.1:8080/api/v1/sso/oidc/callback",
		AutoCreateUser: false, DisplayOnLogin: true,
	}, "admin")
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.resolveMappedUser(ctx, provider, "sub-1", "a@example.com", "Ada", "")
	if err == nil {
		t.Fatal("expected unbound error")
	}
}

func TestResolveMappedUserAutoCreateDoesNotMergeByEmail(t *testing.T) {
	store := NewMemoryStore()
	vault, _ := NewVault("test-sso-secrets-key-32chars-min!", "")
	svc := NewService(store, vault)
	bridge := &stubIdentity{}
	svc.SetIdentity(bridge)
	svc.SetRuntime(RuntimeConfig{DefaultTenantID: "tenant-1"})
	ctx := context.Background()
	provider, err := svc.CreateProvider(ctx, ProviderInput{
		Name: "Keycloak", Protocol: ProtocolOIDC, Status: StatusEnabled,
		Issuer: "https://idp.example.com", ClientID: "web",
		ClientSecret: "secret-value-1234567890", RedirectURI: "http://127.0.0.1:8080/api/v1/sso/oidc/callback",
		AutoCreateUser: true, DisplayOnLogin: true,
	}, "admin")
	if err != nil {
		t.Fatal(err)
	}
	userID, err := svc.resolveMappedUser(ctx, provider, "sub-unique", "shared@example.com", "Ada", "tenant-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(bridge.created) != 1 {
		t.Fatalf("created=%d", len(bridge.created))
	}
	again, err := svc.resolveMappedUser(ctx, provider, "sub-unique", "shared@example.com", "Ada", "tenant-1")
	if err != nil {
		t.Fatal(err)
	}
	if again != userID {
		t.Fatal("second resolve should reuse link")
	}
	if len(bridge.created) != 1 {
		t.Fatal("must not create another user for same subject")
	}
	other, err := svc.resolveMappedUser(ctx, provider, "sub-other", "shared@example.com", "Bob", "tenant-1")
	if err != nil {
		t.Fatal(err)
	}
	if other == userID {
		t.Fatal("same email must not merge different subjects")
	}
	if len(bridge.created) != 2 {
		t.Fatalf("expected second user, got %d", len(bridge.created))
	}
}

func TestExchangeLoginTicket(t *testing.T) {
	store := NewMemoryStore()
	svc := NewService(store, nil)
	bridge := &stubIdentity{}
	svc.SetIdentity(bridge)
	ctx := context.Background()
	_, err := store.CreateLoginTicket(ctx, LoginTicket{
		Ticket: "ticket-1", UserID: "user-1", TenantID: "tenant-1", ProviderID: "p1",
		ExpiresAt: time.Now().UTC().Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := svc.ExchangeLoginTicket(ctx, "ticket-1", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if result.User.ID != "user-1" || result.TenantID != "tenant-1" {
		t.Fatalf("result=%#v", result)
	}
	_, err = svc.ExchangeLoginTicket(ctx, "ticket-1", "127.0.0.1")
	if err == nil {
		t.Fatal("ticket must be one-time")
	}
}

func TestDisabledUserCannotLoginFederated(t *testing.T) {
	store := NewMemoryStore()
	svc := NewService(store, nil)
	svc.SetIdentity(&stubIdentity{
		loginFn: func(context.Context, string, string, string) (identity.LoginResult, error) {
			return identity.LoginResult{}, &identity.Error{Kind: identity.ErrUnauthorized, Message: "账号已停用"}
		},
	})
	ctx := context.Background()
	_, err := store.CreateLoginTicket(ctx, LoginTicket{
		Ticket: "t2", UserID: "disabled", TenantID: "tenant-1", ProviderID: "p1",
		ExpiresAt: time.Now().UTC().Add(time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.ExchangeLoginTicket(ctx, "t2", "127.0.0.1")
	if err == nil {
		t.Fatal("expected failure")
	}
	var domain *Error
	if !errors.As(err, &domain) || domain.Message != "账号已停用" {
		t.Fatalf("err=%v", err)
	}
}
