package sso

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
	"qingzhi/enterprise-api/internal/identity"
)

// IdentityBridge is the minimal identity surface used by federation login.
type IdentityBridge interface {
	LoginFederated(ctx context.Context, userID, tenantID, loginIP string) (identity.LoginResult, error)
	CreateFederatedUser(ctx context.Context, username, nickname, email, tenantID string) (identity.User, error)
}

// RuntimeConfig holds process-level SSO login settings.
type RuntimeConfig struct {
	FrontendBaseURL string
	DefaultTenantID string
	HTTPClient      *http.Client
}

// SetIdentity attaches the local identity service used after external verification.
func (s *Service) SetIdentity(bridge IdentityBridge) {
	if s != nil {
		s.identity = bridge
	}
}

// SetRuntime configures frontend redirect and default tenant for auto-provisioning.
func (s *Service) SetRuntime(cfg RuntimeConfig) {
	if s != nil {
		s.runtime = cfg
		if cfg.HTTPClient != nil {
			s.httpClient = cfg.HTTPClient
		}
	}
}

// LoginOptions returns enabled providers for the login page.
func (s *Service) LoginOptions(ctx context.Context) ([]LoginOption, Settings, error) {
	settings, err := s.store.Settings(ctx)
	if err != nil {
		return nil, Settings{}, err
	}
	items, err := s.store.ListLoginProviders(ctx)
	if err != nil {
		return nil, Settings{}, err
	}
	return items, settings, nil
}

// StartExternalLogin begins an OIDC or OAuth2 authorization-code flow with PKCE.
func (s *Service) StartExternalLogin(ctx context.Context, providerID, redirectPath, tenantID string) (authURL string, err error) {
	provider, err := s.store.GetProvider(ctx, providerID)
	if err != nil {
		return "", err
	}
	if provider.Status != StatusEnabled {
		return "", invalid("身份提供方未启用")
	}
	if provider.Protocol != ProtocolOIDC && provider.Protocol != ProtocolOAuth2 {
		return "", invalid("该提供方不支持此登录方式")
	}
	redirectPath = sanitizeRedirectPath(redirectPath)
	tenantID = strings.TrimSpace(tenantID)

	state, err := randomURLString(32)
	if err != nil {
		return "", err
	}
	nonce, err := randomURLString(32)
	if err != nil {
		return "", err
	}
	verifier, challenge, err := newPKCE()
	if err != nil {
		return "", err
	}

	req := AuthRequest{
		State:        state,
		Nonce:        nonce,
		CodeVerifier: verifier,
		ProviderID:   provider.ID,
		Protocol:     provider.Protocol,
		RedirectPath: redirectPath,
		TenantID:     tenantID,
		ExpiresAt:    s.now().UTC().Add(10 * time.Minute),
	}
	if _, err := s.store.CreateAuthRequest(ctx, req); err != nil {
		return "", err
	}

	oauthCfg, err := s.oauthConfig(ctx, provider)
	if err != nil {
		return "", err
	}
	opts := []oauth2.AuthCodeOption{
		oauth2.SetAuthURLParam("code_challenge", challenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	}
	if provider.Protocol == ProtocolOIDC {
		opts = append(opts, oidc.Nonce(nonce))
	}
	return oauthCfg.AuthCodeURL(state, opts...), nil
}

// CompleteExternalLogin finishes the IdP callback, maps identity, and returns a one-time ticket.
func (s *Service) CompleteExternalLogin(ctx context.Context, code, state, loginIP string) (ticket string, redirectPath string, err error) {
	if strings.TrimSpace(code) == "" || strings.TrimSpace(state) == "" {
		return "", "/", invalid("回调参数不完整")
	}
	authReq, err := s.store.ConsumeAuthRequest(ctx, state)
	if err != nil {
		return "", "/", err
	}
	provider, err := s.store.GetProvider(ctx, authReq.ProviderID)
	if err != nil {
		return "", authReq.RedirectPath, err
	}
	if provider.Status != StatusEnabled {
		return "", authReq.RedirectPath, invalid("身份提供方未启用")
	}

	var subject string
	var email, name string
	switch provider.Protocol {
	case ProtocolOIDC:
		subject, email, name, err = s.completeOIDC(ctx, provider, authReq, code)
	case ProtocolOAuth2:
		subject, email, name, err = s.completeOAuth2(ctx, provider, authReq, code)
	default:
		err = invalid("不支持的协议")
	}
	if err != nil {
		_ = s.store.MarkProviderAuthResult(ctx, provider.ID, false, safeErr(err))
		_ = s.RecordEvent(ctx, AuthEvent{
			EventType:  "login.callback",
			Result:     ResultFailure,
			Protocol:   provider.Protocol,
			ProviderID: provider.ID,
			Message:    safeErr(err),
			IP:         loginIP,
		})
		return "", authReq.RedirectPath, err
	}

	userID, err := s.resolveMappedUser(ctx, provider, subject, email, name, authReq.TenantID)
	if err != nil {
		_ = s.store.MarkProviderAuthResult(ctx, provider.ID, false, safeErr(err))
		_ = s.RecordEvent(ctx, AuthEvent{
			EventType:  "login.map",
			Result:     ResultFailure,
			Protocol:   provider.Protocol,
			ProviderID: provider.ID,
			Message:    safeErr(err),
			IP:         loginIP,
		})
		return "", authReq.RedirectPath, err
	}

	if s.identity == nil {
		return "", authReq.RedirectPath, notReady("身份服务未就绪")
	}
	tenantID := authReq.TenantID
	if tenantID == "" {
		tenantID = strings.TrimSpace(s.runtime.DefaultTenantID)
	}
	login, err := s.identity.LoginFederated(ctx, userID, tenantID, loginIP)
	if err != nil {
		_ = s.store.MarkProviderAuthResult(ctx, provider.ID, false, "本地会话建立失败")
		_ = s.RecordEvent(ctx, AuthEvent{
			EventType:  "login.session",
			Result:     ResultFailure,
			Protocol:   provider.Protocol,
			ProviderID: provider.ID,
			UserID:     userID,
			Message:    "本地会话建立失败",
			IP:         loginIP,
		})
		return "", authReq.RedirectPath, mapIdentityErr(err)
	}

	ticketValue, err := randomURLString(32)
	if err != nil {
		return "", authReq.RedirectPath, err
	}
	_, err = s.store.CreateLoginTicket(ctx, LoginTicket{
		Ticket:     ticketValue,
		UserID:     login.User.ID,
		TenantID:   login.TenantID,
		ProviderID: provider.ID,
		ExpiresAt:  s.now().UTC().Add(60 * time.Second),
	})
	if err != nil {
		return "", authReq.RedirectPath, err
	}

	_ = s.store.MarkProviderAuthResult(ctx, provider.ID, true, "")
	_ = s.RecordEvent(ctx, AuthEvent{
		EventType:  "login.success",
		Result:     ResultSuccess,
		Protocol:   provider.Protocol,
		ProviderID: provider.ID,
		UserID:     login.User.ID,
		Username:   login.User.Username,
		Message:    "外部身份登录成功",
		IP:         loginIP,
	})
	return ticketValue, authReq.RedirectPath, nil
}

// ExchangeLoginTicket consumes a one-time ticket and returns the mapped local login.
func (s *Service) ExchangeLoginTicket(ctx context.Context, ticket, loginIP string) (identity.LoginResult, error) {
	if s.identity == nil {
		return identity.LoginResult{}, notReady("身份服务未就绪")
	}
	item, err := s.store.ConsumeLoginTicket(ctx, strings.TrimSpace(ticket))
	if err != nil {
		return identity.LoginResult{}, err
	}
	result, err := s.identity.LoginFederated(ctx, item.UserID, item.TenantID, loginIP)
	if err != nil {
		return identity.LoginResult{}, mapIdentityErr(err)
	}
	_ = s.RecordEvent(ctx, AuthEvent{
		EventType:  "login.exchange",
		Result:     ResultSuccess,
		ProviderID: item.ProviderID,
		UserID:     result.User.ID,
		Username:   result.User.Username,
		Message:    "登录票据兑换成功",
		IP:         loginIP,
	})
	return result, nil
}

// FrontendRedirectURL builds a safe browser redirect after IdP callback.
func (s *Service) FrontendRedirectURL(ctx context.Context, path, ticket, errMsg string) (string, error) {
	settings, err := s.store.Settings(ctx)
	if err != nil {
		return "", err
	}
	base := strings.TrimSpace(settings.FrontendBaseURL)
	if base == "" {
		base = strings.TrimSpace(s.runtime.FrontendBaseURL)
	}
	if base == "" {
		return "", notReady("未配置前端回调地址 SSO frontend_base_url")
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", invalid("前端回调地址不合法")
	}
	path = sanitizeRedirectPath(path)
	target := strings.TrimRight(base, "/") + "/login/sso/callback"
	q := url.Values{}
	if ticket != "" {
		q.Set("ticket", ticket)
		q.Set("redirect", path)
	}
	if errMsg != "" {
		q.Set("error", clip(errMsg, 120))
	}
	u, _ := url.Parse(target)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (s *Service) completeOIDC(ctx context.Context, provider Provider, authReq AuthRequest, code string) (subject, email, name string, err error) {
	oauthCfg, err := s.oauthConfig(ctx, provider)
	if err != nil {
		return "", "", "", err
	}
	ctx = oidc.ClientContext(ctx, s.client())
	providerOIDC, err := oidc.NewProvider(ctx, provider.Issuer)
	if err != nil {
		return "", "", "", invalid("无法加载 OIDC Discovery")
	}
	verifier := providerOIDC.Verifier(&oidc.Config{ClientID: provider.ClientID})
	token, err := oauthCfg.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", authReq.CodeVerifier))
	if err != nil {
		return "", "", "", invalid("授权码交换失败")
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return "", "", "", invalid("响应缺少 ID Token")
	}
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return "", "", "", invalid("ID Token 校验失败")
	}
	if idToken.Nonce != authReq.Nonce {
		return "", "", "", invalid("nonce 校验失败")
	}
	var claims map[string]any
	if err := idToken.Claims(&claims); err != nil {
		return "", "", "", invalid("无法解析 ID Token Claims")
	}
	subject = claimString(claims, "sub")
	if subject == "" {
		return "", "", "", invalid("ID Token 缺少 sub")
	}
	email = mappedClaim(claims, provider.ClaimMapping, "email", "email")
	name = mappedClaim(claims, provider.ClaimMapping, "name", "name", "preferred_username")
	return subject, email, name, nil
}

func (s *Service) completeOAuth2(ctx context.Context, provider Provider, authReq AuthRequest, code string) (subject, email, name string, err error) {
	if strings.TrimSpace(provider.UserinfoURL) == "" {
		return "", "", "", invalid("通用 OAuth2 必须配置可信用户信息端点")
	}
	oauthCfg, err := s.oauthConfig(ctx, provider)
	if err != nil {
		return "", "", "", err
	}
	token, err := oauthCfg.Exchange(ctx, code, oauth2.SetAuthURLParam("code_verifier", authReq.CodeVerifier))
	if err != nil {
		return "", "", "", invalid("授权码交换失败")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, provider.UserinfoURL, nil)
	if err != nil {
		return "", "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	resp, err := s.client().Do(req)
	if err != nil {
		return "", "", "", invalid("用户信息端点请求失败")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", "", invalid("用户信息端点返回异常")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", "", "", err
	}
	var claims map[string]any
	if err := json.Unmarshal(body, &claims); err != nil {
		return "", "", "", invalid("用户信息响应不是合法 JSON")
	}
	subject = mappedClaim(claims, provider.ClaimMapping, "sub", "sub", "id", "user_id")
	if subject == "" {
		return "", "", "", invalid("用户信息缺少稳定 Subject")
	}
	email = mappedClaim(claims, provider.ClaimMapping, "email", "email")
	name = mappedClaim(claims, provider.ClaimMapping, "name", "name", "display_name")
	return subject, email, name, nil
}

func (s *Service) resolveMappedUser(ctx context.Context, provider Provider, subject, email, name, tenantHint string) (string, error) {
	link, err := s.store.FindIdentityLink(ctx, provider.ID, subject)
	if err == nil {
		return link.UserID, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return "", err
	}
	if !provider.AutoCreateUser {
		return "", invalid("外部身份尚未绑定本平台账号")
	}
	if s.identity == nil {
		return "", notReady("身份服务未就绪")
	}
	tenantID := strings.TrimSpace(tenantHint)
	if tenantID == "" {
		settings, err := s.store.Settings(ctx)
		if err != nil {
			return "", err
		}
		tenantID = strings.TrimSpace(settings.DefaultTenantID)
		if tenantID == "" {
			tenantID = strings.TrimSpace(s.runtime.DefaultTenantID)
		}
	}
	if tenantID == "" {
		return "", invalid("自动开户需要配置默认租户")
	}
	username := federatedUsername(provider.ID, subject)
	nickname := strings.TrimSpace(name)
	if nickname == "" {
		nickname = username
	}
	// Never auto-merge by email: always create a new local user for an unbound subject.
	user, err := s.identity.CreateFederatedUser(ctx, username, nickname, strings.TrimSpace(email), tenantID)
	if err != nil {
		return "", mapIdentityErr(err)
	}
	_, err = s.store.CreateIdentityLink(ctx, IdentityLink{
		ProviderID:      provider.ID,
		ExternalSubject: subject,
		UserID:          user.ID,
	})
	if err != nil {
		return "", err
	}
	_ = s.RecordEvent(ctx, AuthEvent{
		EventType:  "identity_link.create",
		Result:     ResultSuccess,
		Protocol:   provider.Protocol,
		ProviderID: provider.ID,
		UserID:     user.ID,
		Username:   user.Username,
		Message:    "自动创建用户并建立身份映射",
	})
	return user.ID, nil
}

func (s *Service) oauthConfig(ctx context.Context, provider Provider) (*oauth2.Config, error) {
	secret, err := s.providerClientSecret(ctx, provider.ID)
	if err != nil {
		return nil, err
	}
	redirectURI := strings.TrimSpace(provider.RedirectURI)
	if redirectURI == "" {
		return nil, invalid("提供方未配置 Redirect URI")
	}
	scopes := splitScopes(provider.Scopes)
	if provider.Protocol == ProtocolOIDC {
		hasOpenID := false
		for _, scope := range scopes {
			if scope == "openid" {
				hasOpenID = true
				break
			}
		}
		if !hasOpenID {
			scopes = append([]string{"openid"}, scopes...)
		}
	}
	endpoint := oauth2.Endpoint{
		AuthURL:  provider.AuthorizationURL,
		TokenURL: provider.TokenURL,
	}
	if provider.Protocol == ProtocolOIDC {
		ctx = oidc.ClientContext(ctx, s.client())
		p, err := oidc.NewProvider(ctx, provider.Issuer)
		if err != nil {
			return nil, invalid("无法加载 OIDC Discovery")
		}
		endpoint = p.Endpoint()
	} else {
		if endpoint.AuthURL == "" || endpoint.TokenURL == "" {
			return nil, invalid("OAuth2 缺少 Authorization/Token URL")
		}
	}
	return &oauth2.Config{
		ClientID:     provider.ClientID,
		ClientSecret: secret,
		RedirectURL:  redirectURI,
		Endpoint:     endpoint,
		Scopes:       scopes,
	}, nil
}

func (s *Service) providerClientSecret(ctx context.Context, providerID string) (string, error) {
	if s.vault == nil {
		return "", notReady("联邦密钥服务未配置")
	}
	secretID, err := s.store.GetProviderSecretID(ctx, providerID)
	if err != nil {
		return "", err
	}
	ciphertext, err := s.store.GetSecretCiphertext(ctx, secretID)
	if err != nil {
		return "", err
	}
	plain, err := s.vault.Decrypt(ciphertext)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func (s *Service) client() *http.Client {
	if s.httpClient != nil {
		return s.httpClient
	}
	return http.DefaultClient
}

func sanitizeRedirectPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") {
		return "/"
	}
	if strings.ContainsAny(path, "\r\n\\") {
		return "/"
	}
	u, err := url.Parse(path)
	if err != nil || u.IsAbs() || u.Host != "" {
		return "/"
	}
	return path
}

func splitScopes(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return unicode.IsSpace(r) || r == ','
	})
	out := make([]string, 0, len(fields))
	seen := map[string]struct{}{}
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		if _, ok := seen[f]; ok {
			continue
		}
		seen[f] = struct{}{}
		out = append(out, f)
	}
	return out
}

func claimString(claims map[string]any, key string) string {
	v, ok := claims[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case fmt.Stringer:
		return strings.TrimSpace(t.String())
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

func mappedClaim(claims map[string]any, mapping map[string]any, logical string, defaults ...string) string {
	if mapping != nil {
		if raw, ok := mapping[logical]; ok {
			if key, ok := raw.(string); ok && strings.TrimSpace(key) != "" {
				if v := claimString(claims, strings.TrimSpace(key)); v != "" {
					return v
				}
			}
		}
	}
	for _, key := range defaults {
		if v := claimString(claims, key); v != "" {
			return v
		}
	}
	return ""
}

func federatedUsername(providerID, subject string) string {
	sum := sha256.Sum256([]byte(providerID + "|" + subject))
	encoded := base64.RawURLEncoding.EncodeToString(sum[:])
	encoded = strings.ToLower(strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r
		case r >= '0' && r <= '9':
			return r
		default:
			return -1
		}
	}, encoded))
	if len(encoded) > 20 {
		encoded = encoded[:20]
	}
	return "sso_" + encoded
}

func newPKCE() (verifier, challenge string, err error) {
	verifier, err = randomURLString(32)
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

func randomURLString(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func safeErr(err error) string {
	if err == nil {
		return ""
	}
	var domain *Error
	if errors.As(err, &domain) {
		return domain.Message
	}
	return "认证失败"
}

func mapIdentityErr(err error) error {
	if err == nil {
		return nil
	}
	var domain *identity.Error
	if errors.As(err, &domain) {
		return &Error{Kind: ErrInvalid, Message: domain.Message}
	}
	return err
}
