package sso

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

// Service implements SSO administration and audit recording.
type Service struct {
	store      Store
	vault      *Vault
	identity   IdentityBridge
	runtime    RuntimeConfig
	httpClient *http.Client
	now        func() time.Time
}

// NewService returns an SSO service.
func NewService(store Store, vault *Vault) *Service {
	return &Service{store: store, vault: vault, now: time.Now}
}

// Overview aggregates real counts from persistence.
func (s *Service) Overview(ctx context.Context) (Overview, error) {
	settings, err := s.store.Settings(ctx)
	if err != nil {
		return Overview{}, err
	}
	oidcTotal, oidcEnabled, err := s.store.CountProviders(ctx, ProtocolOIDC)
	if err != nil {
		return Overview{}, err
	}
	samlTotal, samlEnabled, err := s.store.CountProviders(ctx, ProtocolSAML)
	if err != nil {
		return Overview{}, err
	}
	links, err := s.store.CountIdentityLinks(ctx)
	if err != nil {
		return Overview{}, err
	}
	since := s.now().UTC().Add(-24 * time.Hour)
	success, err := s.store.CountAuthEventsSince(ctx, ResultSuccess, "login", since)
	if err != nil {
		return Overview{}, err
	}
	failure, err := s.store.CountAuthEventsSince(ctx, ResultFailure, "login", since)
	if err != nil {
		return Overview{}, err
	}
	failures, err := s.store.ListRecentAuthEvents(ctx, ResultFailure, 10)
	if err != nil {
		return Overview{}, err
	}
	if failures == nil {
		failures = []AuthEvent{}
	}
	certs, err := s.store.ListCertificateWarnings(ctx, 30)
	if err != nil {
		return Overview{}, err
	}
	if certs == nil {
		certs = []CertWarning{}
	}
	return Overview{
		LocalLoginEnabled:      settings.LocalLoginEnabled,
		OIDCProviders:          ProviderCount{Total: oidcTotal, Enabled: oidcEnabled},
		SAMLProviders:          ProviderCount{Total: samlTotal, Enabled: samlEnabled},
		IdentityLinks:          links,
		RecentLoginSuccess:     success,
		RecentLoginFailure:     failure,
		RecentProtocolFailures: failures,
		CertificateWarnings:    certs,
		CollectedAt:            s.now().UTC(),
	}, nil
}

// GetSettings returns global SSO settings.
func (s *Service) GetSettings(ctx context.Context) (Settings, error) {
	return s.store.Settings(ctx)
}

// UpdateSettings updates local login and client federation policy.
func (s *Service) UpdateSettings(ctx context.Context, item Settings, actor string) (Settings, error) {
	current, err := s.store.Settings(ctx)
	if err != nil {
		return Settings{}, err
	}
	item.FrontendBaseURL = strings.TrimSpace(item.FrontendBaseURL)
	if item.FrontendBaseURL != "" {
		base, err := canonicalizeFrontendBase(item.FrontendBaseURL)
		if err != nil {
			return Settings{}, err
		}
		item.FrontendBaseURL = base
	}
	item.DefaultTenantID = strings.TrimSpace(item.DefaultTenantID)
	item.Updater = clip(actor, 64)
	item.UpdatedAt = s.now().UTC()
	// Preserve SP signing material managed separately.
	item.SPCertificatePEM = current.SPCertificatePEM
	item.SPKeySecretID = current.SPKeySecretID
	if err := s.store.SaveSettings(ctx, item); err != nil {
		return Settings{}, err
	}
	_ = s.RecordEvent(ctx, AuthEvent{
		EventType: "settings.update",
		Result:    ResultSuccess,
		Message:   "更新 SSO 全局策略",
		Username:  actor,
	})
	return s.store.Settings(ctx)
}

func (s *Service) ListProviders(ctx context.Context, query ListQuery) (Page[Provider], error) {
	query = normalizeList(query)
	if query.Protocol != "" && !validProtocol(query.Protocol) {
		return Page[Provider]{}, invalid("协议类型不合法")
	}
	items, total, err := s.store.ListProviders(ctx, query)
	if err != nil {
		return Page[Provider]{}, err
	}
	return Page[Provider]{Items: items, Page: query.Page, PageSize: query.PageSize, Total: total}, nil
}

func (s *Service) GetProvider(ctx context.Context, id string) (Provider, error) {
	if strings.TrimSpace(id) == "" {
		return Provider{}, invalid("身份提供方不存在")
	}
	return s.store.GetProvider(ctx, id)
}

func (s *Service) CreateProvider(ctx context.Context, input ProviderInput, actor string) (Provider, error) {
	item, err := s.validateProviderInput(input, true)
	if err != nil {
		return Provider{}, err
	}
	var secretID *string
	if secret := strings.TrimSpace(input.ClientSecret); secret != "" {
		id, err := s.storeSecret(ctx, "provider_client_secret", secret)
		if err != nil {
			return Provider{}, err
		}
		secretID = &id
	}
	created, err := s.store.CreateProvider(ctx, item, secretID)
	if err != nil {
		return Provider{}, err
	}
	_ = s.RecordEvent(ctx, AuthEvent{
		EventType:  "provider.create",
		Result:     ResultSuccess,
		Protocol:   created.Protocol,
		ProviderID: created.ID,
		Message:    "创建外部身份提供方",
		Username:   actor,
	})
	return created, nil
}

func (s *Service) UpdateProvider(ctx context.Context, id string, input ProviderInput, actor string) (Provider, error) {
	existing, err := s.store.GetProvider(ctx, id)
	if err != nil {
		return Provider{}, err
	}
	item, err := s.validateProviderInput(input, false)
	if err != nil {
		return Provider{}, err
	}
	item.ID = existing.ID
	item.Protocol = existing.Protocol
	var secretID *string
	if secret := strings.TrimSpace(input.ClientSecret); secret != "" {
		id, err := s.storeSecret(ctx, "provider_client_secret", secret)
		if err != nil {
			return Provider{}, err
		}
		secretID = &id
	}
	updated, err := s.store.UpdateProvider(ctx, item, secretID)
	if err != nil {
		return Provider{}, err
	}
	_ = s.RecordEvent(ctx, AuthEvent{
		EventType:  "provider.update",
		Result:     ResultSuccess,
		Protocol:   updated.Protocol,
		ProviderID: updated.ID,
		Message:    "更新外部身份提供方",
		Username:   actor,
	})
	return updated, nil
}

func (s *Service) SetProviderStatus(ctx context.Context, id, status, actor string) error {
	status = strings.TrimSpace(status)
	if status != StatusEnabled && status != StatusDisabled {
		return invalid("状态不合法")
	}
	if err := s.store.SetProviderStatus(ctx, id, status); err != nil {
		return err
	}
	_ = s.RecordEvent(ctx, AuthEvent{
		EventType:  "provider.status",
		Result:     ResultSuccess,
		ProviderID: id,
		Message:    "更新提供方状态为 " + status,
		Username:   actor,
	})
	return nil
}

func (s *Service) ListIdentityLinks(ctx context.Context, query ListQuery) (Page[IdentityLink], error) {
	query = normalizeList(query)
	items, total, err := s.store.ListIdentityLinks(ctx, query)
	if err != nil {
		return Page[IdentityLink]{}, err
	}
	return Page[IdentityLink]{Items: items, Page: query.Page, PageSize: query.PageSize, Total: total}, nil
}

func (s *Service) DeleteIdentityLink(ctx context.Context, id, actor string) error {
	if err := s.store.DeleteIdentityLink(ctx, id); err != nil {
		return err
	}
	_ = s.RecordEvent(ctx, AuthEvent{
		EventType: "identity_link.delete",
		Result:    ResultSuccess,
		Message:   "解绑外部身份映射",
		Username:  actor,
	})
	return nil
}

func (s *Service) ListAuthEvents(ctx context.Context, query ListQuery) (Page[AuthEvent], error) {
	query = normalizeList(query)
	if query.Result != "" && query.Result != ResultSuccess && query.Result != ResultFailure {
		return Page[AuthEvent]{}, invalid("结果不合法")
	}
	items, total, err := s.store.ListAuthEvents(ctx, query)
	if err != nil {
		return Page[AuthEvent]{}, err
	}
	return Page[AuthEvent]{Items: items, Page: query.Page, PageSize: query.PageSize, Total: total}, nil
}

// RecordEvent writes a redacted auth event. Failures are ignored by callers that already succeeded.
func (s *Service) RecordEvent(ctx context.Context, item AuthEvent) error {
	item.Message = clip(item.Message, 500)
	item.Username = clip(item.Username, 64)
	item.ProviderID = clip(item.ProviderID, 64)
	item.ClientID = clip(item.ClientID, 64)
	item.UserID = clip(item.UserID, 64)
	item.RequestID = clip(item.RequestID, 64)
	item.IP = clip(item.IP, 64)
	item.EventType = clip(item.EventType, 64)
	item.Protocol = clip(item.Protocol, 16)
	if item.Result == "" {
		item.Result = ResultSuccess
	}
	if item.CreateTime.IsZero() {
		item.CreateTime = s.now().UTC()
	}
	return s.store.InsertAuthEvent(ctx, item)
}

func (s *Service) storeSecret(ctx context.Context, kind, plaintext string) (string, error) {
	if s.vault == nil {
		return "", notReady("联邦密钥服务未配置")
	}
	ciphertext, err := s.vault.Encrypt([]byte(plaintext))
	if err != nil {
		return "", err
	}
	return s.store.CreateSecret(ctx, kind, secretHint(plaintext), ciphertext)
}

func (s *Service) validateProviderInput(input ProviderInput, creating bool) (Provider, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" || utf8.RuneCountInString(name) > 128 {
		return Provider{}, invalid("提供方名称不合法")
	}
	protocol := strings.TrimSpace(input.Protocol)
	if creating {
		if !validProtocol(protocol) {
			return Provider{}, invalid("协议类型不合法")
		}
	} else if protocol == "" {
		protocol = ProtocolOIDC
	}
	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = StatusDisabled
	}
	if status != StatusEnabled && status != StatusDisabled {
		return Provider{}, invalid("状态不合法")
	}
	priority := input.Priority
	if priority <= 0 {
		priority = 100
	}
	item := Provider{
		Name:             name,
		Protocol:         protocol,
		Status:           status,
		Priority:         priority,
		Issuer:           strings.TrimSpace(input.Issuer),
		DisplayOnLogin:   input.DisplayOnLogin,
		AutoLogin:        input.AutoLogin,
		AutoCreateUser:   input.AutoCreateUser,
		DiscoveryURL:     strings.TrimSpace(input.DiscoveryURL),
		ClientID:         strings.TrimSpace(input.ClientID),
		RedirectURI:      strings.TrimSpace(input.RedirectURI),
		Scopes:           strings.TrimSpace(input.Scopes),
		ClaimMapping:     input.ClaimMapping,
		EntityID:         strings.TrimSpace(input.EntityID),
		MetadataURL:      strings.TrimSpace(input.MetadataURL),
		SSOURL:           strings.TrimSpace(input.SSOURL),
		SLOURL:           strings.TrimSpace(input.SLOURL),
		IDPCertificatePEM: strings.TrimSpace(input.IDPCertificatePEM),
		SPEntityID:       strings.TrimSpace(input.SPEntityID),
		ACSURL:           strings.TrimSpace(input.ACSURL),
		NameIDFormat:     strings.TrimSpace(input.NameIDFormat),
	}
	if item.ClaimMapping == nil {
		item.ClaimMapping = map[string]any{}
	}
	switch protocol {
	case ProtocolOIDC:
		if item.Issuer == "" {
			return Provider{}, invalid("OIDC Issuer 不能为空")
		}
		if err := validateAbsoluteURL(item.Issuer, "Issuer"); err != nil {
			return Provider{}, err
		}
		if item.ClientID == "" {
			return Provider{}, invalid("Client ID 不能为空")
		}
		if item.RedirectURI != "" {
			if err := validateAbsoluteURL(item.RedirectURI, "Redirect URI"); err != nil {
				return Provider{}, err
			}
		}
		if item.DiscoveryURL == "" {
			item.DiscoveryURL = strings.TrimRight(item.Issuer, "/") + "/.well-known/openid-configuration"
		}
	case ProtocolSAML:
		if item.EntityID == "" {
			return Provider{}, invalid("IdP Entity ID 不能为空")
		}
		if item.SSOURL == "" && item.MetadataURL == "" {
			return Provider{}, invalid("SAML 必须提供 SSO URL 或 Metadata URL")
		}
		if item.NameIDFormat == "" {
			item.NameIDFormat = "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"
		}
		if item.SSOURL != "" {
			if err := validateAbsoluteURL(item.SSOURL, "SSO URL"); err != nil {
				return Provider{}, err
			}
		}
		if item.MetadataURL != "" {
			if err := validateAbsoluteURL(item.MetadataURL, "Metadata URL"); err != nil {
				return Provider{}, err
			}
		}
		if utf8.RuneCountInString(item.IDPCertificatePEM) > 16*1024 {
			return Provider{}, invalid("证书内容过大")
		}
	}
	return item, nil
}

func normalizeList(query ListQuery) ListQuery {
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.PageSize > 100 {
		query.PageSize = 100
	}
	query.Keyword = clip(strings.TrimSpace(query.Keyword), 64)
	query.Protocol = strings.TrimSpace(query.Protocol)
	query.Status = strings.TrimSpace(query.Status)
	query.Result = strings.TrimSpace(query.Result)
	return query
}

func validProtocol(v string) bool {
	return v == ProtocolOIDC || v == ProtocolSAML
}

func validateAbsoluteURL(raw, label string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return invalid(label + " 不是合法 URL")
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return invalid(label + " 仅支持 http/https")
	}
	return nil
}

func normalizeExactURIs(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func normalizeListValues(values, defaults []string) []string {
	out := normalizeExactURIs(values)
	if len(out) == 0 {
		return append([]string{}, defaults...)
	}
	return out
}

func defaultString(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

func clip(v string, max int) string {
	if utf8.RuneCountInString(v) <= max {
		return v
	}
	runes := []rune(v)
	return string(runes[:max])
}
