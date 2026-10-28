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
		Name:        "Corp OIDC",
		Protocol:    ProtocolOIDC,
		Status:      StatusEnabled,
		Issuer:      "https://idp.example.com",
		ClientID:    "web",
		RedirectURI: "https://app.example.com/callback",
	}, "admin")
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.CreateProvider(ctx, ProviderInput{
		Name:        "Corp SAML",
		Protocol:    ProtocolSAML,
		Status:      StatusEnabled,
		EntityID:    "https://idp.example.com/entity",
		SSOURL:      "https://idp.example.com/sso",
		IDPCertificatePEM: "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----",
	}, "admin")
	if err != nil {
		t.Fatal(err)
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
	if overview.SAMLProviders.Total != 1 || overview.SAMLProviders.Enabled != 1 {
		t.Fatalf("saml providers = %#v", overview.SAMLProviders)
	}
	if overview.RecentLoginSuccess != 1 || overview.RecentLoginFailure != 1 {
		t.Fatalf("login counts success=%d failure=%d", overview.RecentLoginSuccess, overview.RecentLoginFailure)
	}
	if len(overview.RecentProtocolFailures) != 1 {
		t.Fatalf("failures = %#v", overview.RecentProtocolFailures)
	}
}

func TestRejectOAuth2Provider(t *testing.T) {
	store := NewMemoryStore()
	svc := NewService(store, nil)
	_, err := svc.CreateProvider(context.Background(), ProviderInput{
		Name:     "Legacy OAuth2",
		Protocol: "oauth2",
		Status:   StatusDisabled,
	}, "admin")
	if err == nil || !strings.Contains(err.Error(), "协议") {
		t.Fatalf("expected protocol rejection, got %v", err)
	}
}

func TestVaultRoundTrip(t *testing.T) {
	vault, err := NewVault("test-sso-secrets-key-32chars-min!", "")
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
	ctx := context.Background()
	expired := time.Now().UTC().Add(-24 * time.Hour)
	_, err := store.CreateProvider(ctx, Provider{
		Name:                "Expiring",
		Protocol:            ProtocolOIDC,
		Status:              StatusEnabled,
		Issuer:              "https://idp.example.com",
		ClientID:            "web",
		CertificateNotAfter: &expired,
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	overview, err := svc.Overview(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(overview.CertificateWarnings) != 1 || !overview.CertificateWarnings[0].Expired {
		t.Fatalf("warnings = %#v", overview.CertificateWarnings)
	}
}
