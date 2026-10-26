package sso

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestOverviewUsesRealCounts(t *testing.T) {
	store := NewMemoryStore()
	vault, err := NewVault("test-sso-secrets-key-32chars-min!", "")
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(store, vault)
	ctx := context.Background()

	_, err = svc.CreateProvider(ctx, ProviderInput{
		Name:       "Corp OIDC",
		Protocol:   ProtocolOIDC,
		Status:     StatusEnabled,
		Issuer:     "https://idp.example.com",
		ClientID:   "web",
		RedirectURI: "https://app.example.com/callback",
	}, "admin")
	if err != nil {
		t.Fatal(err)
	}
	created, err := svc.CreateOAuthClient(ctx, OAuthClientInput{
		Name:         "Partner App",
		ClientType:   ClientConfidential,
		Status:       StatusEnabled,
		RedirectURIs: []string{"https://partner.example.com/cb"},
		GrantTypes:   []string{"authorization_code"},
		Scopes:       []string{"openid"},
		PKCEPolicy:   PKCERequired,
	}, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if created.ClientSecret == "" || created.ClientSecretSet == false {
		t.Fatalf("expected one-time secret, got %#v", created)
	}
	if strings.Contains(created.ClientSecretHint, created.ClientSecret) {
		t.Fatal("hint must not contain full secret")
	}

	_ = svc.RecordEvent(ctx, AuthEvent{EventType: "login", Result: ResultSuccess, Protocol: ProtocolOIDC})
	_ = svc.RecordEvent(ctx, AuthEvent{EventType: "login", Result: ResultFailure, Protocol: ProtocolOIDC, Message: "nonce mismatch"})

	overview, err := svc.Overview(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if overview.OIDCProviders.Total != 1 || overview.OIDCProviders.Enabled != 1 {
		t.Fatalf("oidc providers = %#v", overview.OIDCProviders)
	}
	if overview.OAuthClients.Total != 1 || overview.OAuthClients.Enabled != 1 {
		t.Fatalf("oauth clients = %#v", overview.OAuthClients)
	}
	if overview.RecentLoginSuccess != 1 || overview.RecentLoginFailure != 1 {
		t.Fatalf("login counts success=%d failure=%d", overview.RecentLoginSuccess, overview.RecentLoginFailure)
	}
	if len(overview.RecentProtocolFailures) != 1 {
		t.Fatalf("failures = %#v", overview.RecentProtocolFailures)
	}
}

func TestOAuthClientListDoesNotExposeSecret(t *testing.T) {
	store := NewMemoryStore()
	vault, err := NewVault("test-sso-secrets-key-32chars-min!", "")
	if err != nil {
		t.Fatal(err)
	}
	svc := NewService(store, vault)
	ctx := context.Background()
	created, err := svc.CreateOAuthClient(ctx, OAuthClientInput{
		Name:         "App",
		ClientType:   ClientConfidential,
		Status:       StatusDisabled,
		RedirectURIs: []string{"https://app.example.com/cb"},
	}, "admin")
	if err != nil {
		t.Fatal(err)
	}
	listed, err := svc.ListOAuthClients(ctx, ListQuery{Page: 1, PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.Items) != 1 {
		t.Fatalf("items = %#v", listed.Items)
	}
	if !listed.Items[0].ClientSecretSet {
		t.Fatal("expected secret set flag")
	}
	if listed.Items[0].ClientSecretHint != "" && strings.Contains(created.ClientSecret, listed.Items[0].ClientSecretHint) {
		// hint is suffix only; full secret must not appear as the hint value
	}
	if listed.Items[0].ClientSecretHint == created.ClientSecret {
		t.Fatal("list leaked full secret into hint")
	}
}

func TestRejectWildcardRedirectURI(t *testing.T) {
	store := NewMemoryStore()
	vault, _ := NewVault("test-sso-secrets-key-32chars-min!", "")
	svc := NewService(store, vault)
	_, err := svc.CreateOAuthClient(context.Background(), OAuthClientInput{
		Name:         "Bad",
		ClientType:   ClientPublic,
		RedirectURIs: []string{"https://*.example.com/cb"},
		PKCEPolicy:   PKCERequired,
	}, "admin")
	if err == nil {
		t.Fatal("expected wildcard rejection")
	}
}

func TestOAuth2ProviderRequiresUserinfo(t *testing.T) {
	store := NewMemoryStore()
	svc := NewService(store, nil)
	_, err := svc.CreateProvider(context.Background(), ProviderInput{
		Name:             "Legacy OAuth",
		Protocol:         ProtocolOAuth2,
		AuthorizationURL: "https://auth.example.com/authorize",
		TokenURL:         "https://auth.example.com/token",
		ClientID:         "c1",
	}, "admin")
	if err == nil {
		t.Fatal("expected userinfo requirement")
	}
}

func TestVaultRoundTrip(t *testing.T) {
	vault, err := NewVault("abcdefghijklmnopqrstuvwxyz012345", "")
	if err != nil {
		t.Fatal(err)
	}
	cipher, err := vault.Encrypt([]byte("super-secret"))
	if err != nil {
		t.Fatal(err)
	}
	plain, err := vault.Decrypt(cipher)
	if err != nil {
		t.Fatal(err)
	}
	if string(plain) != "super-secret" {
		t.Fatalf("plain = %q", plain)
	}
}

func TestCertificateWarningAppearsInOverview(t *testing.T) {
	store := NewMemoryStore()
	svc := NewService(store, nil)
	expired := time.Now().UTC().Add(-24 * time.Hour)
	_, err := store.CreateSamlSP(context.Background(), SamlSP{
		Name:                "Old SP",
		EntityID:            "https://sp.example.com",
		Status:              StatusEnabled,
		ACSURLs:             []string{"https://sp.example.com/acs"},
		CertificateNotAfter: &expired,
	})
	if err != nil {
		t.Fatal(err)
	}
	overview, err := svc.Overview(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(overview.CertificateWarnings) != 1 || !overview.CertificateWarnings[0].Expired {
		t.Fatalf("warnings = %#v", overview.CertificateWarnings)
	}
}
