package auth

import (
	"testing"
	"time"
)

func TestTokenIssuerRoundTrip(t *testing.T) {
	issuer, err := NewTokenIssuer("test-jwt-secret-01", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	token, exp, err := issuer.Issue("user-1", "admin", "tenant-1", now)
	if err != nil {
		t.Fatal(err)
	}
	if !exp.After(now) {
		t.Fatalf("exp = %s", exp)
	}
	claims, err := issuer.Parse(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != "user-1" || claims.Username != "admin" || claims.TenantID != "tenant-1" {
		t.Fatalf("claims = %#v", claims)
	}
}

func TestTokenIssuerRejectsTamperedToken(t *testing.T) {
	issuer, err := NewTokenIssuer("test-jwt-secret-01", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := issuer.Issue("user-1", "admin", "tenant-1", time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	tampered := token[:len(token)-2] + "aa"
	if _, err := issuer.Parse(tampered); err == nil {
		t.Fatal("expected invalid token error")
	}
}
