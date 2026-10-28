package weknora

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	AuthMethodLocal = "local"
	AuthMethodSSO   = "sso"

	externalAudience = "weknora"
	maxTokenTTL      = 24 * time.Hour
	defaultTokenTTL  = 10 * time.Minute
)

// PlatformIdentity is the unified authenticated subject used for WeKnora calls.
// PlatformUserID is stable across local and SSO login for the same sys_users row.
type PlatformIdentity struct {
	PlatformUserID   string
	PlatformTenantID string
	WeKnoraTenantID  string
	AuthMethod       string
	Username         string
}

// ExternalUserClaims are the short-lived HS256 claims for X-External-User-Token.
type ExternalUserClaims struct {
	Subject  string `json:"sub"`
	TenantID string `json:"tenant_id"`
	Audience string `json:"aud"`
	IssuedAt int64  `json:"iat"`
	Expires  int64  `json:"exp"`
}

// ExternalUserTokenIssuer signs terminal-user JWTs for WeKnora signed_token mode.
type ExternalUserTokenIssuer struct {
	secret []byte
	ttl    time.Duration
}

// NewExternalUserTokenIssuer builds an issuer. hmacSecret must match WeKnora
// api-principal-config hmac_secret. Empty secret disables issuance.
func NewExternalUserTokenIssuer(hmacSecret string, ttl time.Duration) (*ExternalUserTokenIssuer, error) {
	hmacSecret = strings.TrimSpace(hmacSecret)
	if hmacSecret == "" {
		return nil, fmt.Errorf("WEKNORA_EXTERNAL_USER_HMAC is required for signed_token issuance")
	}
	if len(hmacSecret) < 16 {
		return nil, fmt.Errorf("WEKNORA_EXTERNAL_USER_HMAC must be at least 16 characters")
	}
	if ttl <= 0 {
		ttl = defaultTokenTTL
	}
	if ttl > maxTokenTTL {
		ttl = maxTokenTTL
	}
	return &ExternalUserTokenIssuer{secret: []byte(hmacSecret), ttl: ttl}, nil
}

// Enabled reports whether the issuer can sign tokens.
func (i *ExternalUserTokenIssuer) Enabled() bool {
	return i != nil && len(i.secret) >= 16
}

// Issue signs a short-lived token for the given platform identity.
// sub is always PlatformUserID so local and SSO logins of the same user share memory identity.
func (i *ExternalUserTokenIssuer) Issue(identity PlatformIdentity, now time.Time) (string, time.Time, error) {
	if !i.Enabled() {
		return "", time.Time{}, fmt.Errorf("WeKnora external user token issuer is not configured")
	}
	if strings.TrimSpace(identity.PlatformUserID) == "" {
		return "", time.Time{}, fmt.Errorf("platform user id is required")
	}
	if strings.TrimSpace(identity.WeKnoraTenantID) == "" {
		return "", time.Time{}, fmt.Errorf("weknora tenant id is required")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	exp := now.Add(i.ttl)
	claims := ExternalUserClaims{
		Subject:  identity.PlatformUserID,
		TenantID: identity.WeKnoraTenantID,
		Audience: externalAudience,
		IssuedAt: now.Unix(),
		Expires:  exp.Unix(),
	}
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		return "", time.Time{}, err
	}
	payload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := header + "." + payload
	mac := hmac.New(sha256.New, i.secret)
	_, _ = mac.Write([]byte(signingInput))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return signingInput + "." + sig, exp, nil
}

// Parse validates a token signed by this issuer (tests / diagnostics only).
func (i *ExternalUserTokenIssuer) Parse(token string) (ExternalUserClaims, error) {
	if !i.Enabled() {
		return ExternalUserClaims{}, fmt.Errorf("issuer not configured")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return ExternalUserClaims{}, fmt.Errorf("token malformed")
	}
	signingInput := parts[0] + "." + parts[1]
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return ExternalUserClaims{}, fmt.Errorf("token malformed")
	}
	mac := hmac.New(sha256.New, i.secret)
	_, _ = mac.Write([]byte(signingInput))
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return ExternalUserClaims{}, fmt.Errorf("token invalid")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ExternalUserClaims{}, fmt.Errorf("token malformed")
	}
	var claims ExternalUserClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ExternalUserClaims{}, fmt.Errorf("token malformed")
	}
	if claims.Subject == "" || claims.TenantID == "" || claims.Audience != externalAudience || claims.Expires <= 0 {
		return ExternalUserClaims{}, fmt.Errorf("token malformed")
	}
	if time.Now().UTC().Unix() >= claims.Expires {
		return ExternalUserClaims{}, fmt.Errorf("token expired")
	}
	return claims, nil
}

// ResolvePlatformIdentity builds a WeKnora-facing identity from the platform principal.
// weknoraTenantID must come from verified server-side mapping, never from the client.
func ResolvePlatformIdentity(platformUserID, platformTenantID, username, authMethod, weknoraTenantID string) (PlatformIdentity, error) {
	platformUserID = strings.TrimSpace(platformUserID)
	platformTenantID = strings.TrimSpace(platformTenantID)
	weknoraTenantID = strings.TrimSpace(weknoraTenantID)
	if platformUserID == "" {
		return PlatformIdentity{}, fmt.Errorf("未认证用户")
	}
	if platformTenantID == "" {
		return PlatformIdentity{}, fmt.Errorf("缺少平台租户")
	}
	if weknoraTenantID == "" {
		return PlatformIdentity{}, fmt.Errorf("未配置 WeKnora 空间 ID")
	}
	method := strings.TrimSpace(authMethod)
	if method == "" {
		method = AuthMethodLocal
	}
	return PlatformIdentity{
		PlatformUserID:   platformUserID,
		PlatformTenantID: platformTenantID,
		WeKnoraTenantID:  weknoraTenantID,
		AuthMethod:       method,
		Username:         strings.TrimSpace(username),
	}, nil
}
