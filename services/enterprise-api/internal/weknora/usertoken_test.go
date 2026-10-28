package weknora

import (
	"testing"
	"time"
)

func TestExternalUserTokenSamePlatformUserAcrossAuthMethods(t *testing.T) {
	issuer, err := NewExternalUserTokenIssuer("test-weknora-hmac-secret-32", 10*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	local, err := ResolvePlatformIdentity("user-1", "tenant-a", "admin", AuthMethodLocal, "10001")
	if err != nil {
		t.Fatal(err)
	}
	sso, err := ResolvePlatformIdentity("user-1", "tenant-a", "admin", AuthMethodSSO, "10001")
	if err != nil {
		t.Fatal(err)
	}
	tokLocal, _, err := issuer.Issue(local, now)
	if err != nil {
		t.Fatal(err)
	}
	tokSSO, _, err := issuer.Issue(sso, now)
	if err != nil {
		t.Fatal(err)
	}
	claimsLocal, err := issuer.Parse(tokLocal)
	if err != nil {
		t.Fatal(err)
	}
	claimsSSO, err := issuer.Parse(tokSSO)
	if err != nil {
		t.Fatal(err)
	}
	if claimsLocal.Subject != claimsSSO.Subject || claimsLocal.Subject != "user-1" {
		t.Fatalf("sub mismatch local=%q sso=%q", claimsLocal.Subject, claimsSSO.Subject)
	}
	if claimsLocal.TenantID != "10001" || claimsSSO.TenantID != "10001" {
		t.Fatalf("tenant_id mismatch: %#v %#v", claimsLocal, claimsSSO)
	}
	if claimsLocal.Audience != externalAudience {
		t.Fatalf("aud=%q", claimsLocal.Audience)
	}
	if local.AuthMethod == sso.AuthMethod {
		t.Fatal("auth methods should differ while sub stays the same")
	}
}

func TestExternalUserTokenDifferentUsersDoNotShareSubject(t *testing.T) {
	issuer, err := NewExternalUserTokenIssuer("test-weknora-hmac-secret-32", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := ResolvePlatformIdentity("user-a", "t1", "a", AuthMethodLocal, "10001")
	b, _ := ResolvePlatformIdentity("user-b", "t1", "b", AuthMethodLocal, "10001")
	tokA, _, err := issuer.Issue(a, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	tokB, _, err := issuer.Issue(b, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	ca, _ := issuer.Parse(tokA)
	cb, _ := issuer.Parse(tokB)
	if ca.Subject == cb.Subject {
		t.Fatal("different users must not share WeKnora subject")
	}
}

func TestResolvePlatformIdentityRequiresMapping(t *testing.T) {
	if _, err := ResolvePlatformIdentity("u1", "t1", "admin", AuthMethodLocal, ""); err == nil {
		t.Fatal("missing weknora tenant must fail")
	}
	if _, err := ResolvePlatformIdentity("", "t1", "admin", AuthMethodLocal, "1"); err == nil {
		t.Fatal("missing platform user must fail")
	}
}

func TestMergeMemoryConfigPreservesUnspecifiedDefaults(t *testing.T) {
	current := DefaultMemoryConfig()
	current.ExtractInstructions = "keep me"
	current.ExtractModelID = "model-a"
	current.Enabled = true
	patch := MemoryConfig{
		Enabled:                   false,
		WriteMode:                 "auto",
		MaxItems:                  50,
		ExtractDelaySeconds:       10,
		ExtractMinIntervalSeconds: 20,
		InterestThreshold:         5,
		VectorRecall:              false,
		RetrievalConditioning:     true,
		ExtractInstructions:       "keep me",
		ExtractModelID:            "model-a",
		EmbeddingModelID:          "emb-1",
	}
	got := MergeMemoryConfig(current, patch)
	if got.ExtractInstructions != "keep me" || got.ExtractModelID != "model-a" {
		t.Fatalf("lost fields: %#v", got)
	}
	if got.Enabled || got.WriteMode != "auto" || got.MaxItems != 50 {
		t.Fatalf("patch not applied: %#v", got)
	}
	if err := ValidateMemoryConfig(got); err != nil {
		t.Fatal(err)
	}
}
