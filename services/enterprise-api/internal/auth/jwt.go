package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Claims are the JWT payload fields used by the admin API.
type Claims struct {
	UserID     string `json:"uid"`
	Username   string `json:"uname"`
	TenantID   string `json:"tid"`
	AuthMethod string `json:"amethod,omitempty"`
	IssuedAt   int64  `json:"iat"`
	Expires    int64  `json:"exp"`
}

const (
	AuthMethodLocal = "local"
	AuthMethodSSO   = "sso"
)

var (
	errTokenMalformed = errors.New("token malformed")
	errTokenExpired   = errors.New("token expired")
	errTokenInvalid   = errors.New("token invalid")
)

// TokenIssuer creates and validates HS256 JWTs without an external JWT library.
type TokenIssuer struct {
	secret []byte
	ttl    time.Duration
}

// NewTokenIssuer returns an HS256 issuer. JWT is implemented with the standard
// library so the service does not need an extra dependency for a single token format.
func NewTokenIssuer(secret string, ttl time.Duration) (*TokenIssuer, error) {
	secret = strings.TrimSpace(secret)
	if len(secret) < 16 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 16 characters")
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &TokenIssuer{secret: []byte(secret), ttl: ttl}, nil
}

// Issue returns a signed access token and its expiry time.
func (i *TokenIssuer) Issue(userID, username, tenantID string, now time.Time) (string, time.Time, error) {
	return i.IssueWithMethod(userID, username, tenantID, AuthMethodLocal, now)
}

// IssueWithMethod records how the platform session was established (local or sso).
func (i *TokenIssuer) IssueWithMethod(userID, username, tenantID, authMethod string, now time.Time) (string, time.Time, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	exp := now.Add(i.ttl)
	method := strings.TrimSpace(authMethod)
	if method == "" {
		method = AuthMethodLocal
	}
	claims := Claims{
		UserID:     userID,
		Username:   username,
		TenantID:   tenantID,
		AuthMethod: method,
		IssuedAt:   now.Unix(),
		Expires:    exp.Unix(),
	}
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		return "", time.Time{}, err
	}
	payload := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := header + "." + payload
	sig := signHS256(i.secret, signingInput)
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig), exp, nil
}

// Parse validates the token signature and expiry.
func (i *TokenIssuer) Parse(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, errTokenMalformed
	}
	signingInput := parts[0] + "." + parts[1]
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return Claims{}, errTokenMalformed
	}
	expected := signHS256(i.secret, signingInput)
	if !hmac.Equal(sig, expected) {
		return Claims{}, errTokenInvalid
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, errTokenMalformed
	}
	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return Claims{}, errTokenMalformed
	}
	if claims.UserID == "" || claims.TenantID == "" || claims.Expires <= 0 {
		return Claims{}, errTokenMalformed
	}
	if time.Now().UTC().Unix() >= claims.Expires {
		return Claims{}, errTokenExpired
	}
	return claims, nil
}

func signHS256(secret []byte, signingInput string) []byte {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(signingInput))
	return mac.Sum(nil)
}
