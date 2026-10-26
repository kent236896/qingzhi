package sso

import (
	"context"
	"strings"
	"sync"
	"time"
)

// MemoryStore is an in-memory Store for unit tests.
type MemoryStore struct {
	mu             sync.Mutex
	settings       Settings
	providers      map[string]Provider
	secrets        map[string]secretMem
	clients        map[string]OAuthClient
	sps            map[string]SamlSP
	links          map[string]IdentityLink
	events         []AuthEvent
	authRequests   map[string]AuthRequest
	tickets        map[string]LoginTicket
	providerSecret map[string]string
	clientSecret   map[string]string
}

type secretMem struct {
	kind       string
	hint       string
	ciphertext []byte
	revoked    bool
}

// NewMemoryStore returns an empty memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		settings:       Settings{LocalLoginEnabled: true, UpdatedAt: time.Now().UTC()},
		providers:      map[string]Provider{},
		secrets:        map[string]secretMem{},
		clients:        map[string]OAuthClient{},
		sps:            map[string]SamlSP{},
		links:          map[string]IdentityLink{},
		authRequests:   map[string]AuthRequest{},
		tickets:        map[string]LoginTicket{},
		providerSecret: map[string]string{},
		clientSecret:   map[string]string{},
	}
}

func (s *MemoryStore) Settings(context.Context) (Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.settings, nil
}

func (s *MemoryStore) SaveSettings(_ context.Context, item Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settings = item
	return nil
}

func (s *MemoryStore) CountProviders(_ context.Context, protocol string) (int64, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var total, enabled int64
	for _, p := range s.providers {
		if p.Protocol != protocol {
			continue
		}
		total++
		if p.Status == StatusEnabled {
			enabled++
		}
	}
	return total, enabled, nil
}

func (s *MemoryStore) CountOAuthClients(context.Context) (int64, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var total, enabled int64
	for _, c := range s.clients {
		total++
		if c.Status == StatusEnabled {
			enabled++
		}
	}
	return total, enabled, nil
}

func (s *MemoryStore) CountSamlSPs(context.Context) (int64, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var total, enabled int64
	for _, sp := range s.sps {
		total++
		if sp.Status == StatusEnabled {
			enabled++
		}
	}
	return total, enabled, nil
}

func (s *MemoryStore) CountIdentityLinks(context.Context) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return int64(len(s.links)), nil
}

func (s *MemoryStore) CountAuthEventsSince(_ context.Context, result, eventTypePrefix string, since time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var n int64
	for _, e := range s.events {
		if e.Result != result || e.CreateTime.Before(since) {
			continue
		}
		if eventTypePrefix != "" && !strings.HasPrefix(e.EventType, eventTypePrefix) {
			continue
		}
		n++
	}
	return n, nil
}

func (s *MemoryStore) ListRecentAuthEvents(_ context.Context, result string, limit int) ([]AuthEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]AuthEvent, 0, limit)
	for i := len(s.events) - 1; i >= 0 && len(out) < limit; i-- {
		if result != "" && s.events[i].Result != result {
			continue
		}
		out = append(out, s.events[i])
	}
	return out, nil
}

func (s *MemoryStore) ListCertificateWarnings(_ context.Context, withinDays int) ([]CertWarning, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	deadline := time.Now().UTC().AddDate(0, 0, withinDays)
	var out []CertWarning
	for _, p := range s.providers {
		if p.CertificateNotAfter != nil && !p.CertificateNotAfter.After(deadline) {
			out = append(out, certWarning("provider", p.ID, p.Name, p.CertificateNotAfter))
		}
	}
	for _, sp := range s.sps {
		if sp.CertificateNotAfter != nil && !sp.CertificateNotAfter.After(deadline) {
			out = append(out, certWarning("saml_sp", sp.ID, sp.Name, sp.CertificateNotAfter))
		}
	}
	return out, nil
}

func (s *MemoryStore) ListProviders(_ context.Context, query ListQuery) ([]Provider, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var all []Provider
	for _, p := range s.providers {
		if query.Protocol != "" && p.Protocol != query.Protocol {
			continue
		}
		if query.Status != "" && p.Status != query.Status {
			continue
		}
		if query.Keyword != "" && !strings.Contains(strings.ToLower(p.Name+p.Issuer+p.ClientID), strings.ToLower(query.Keyword)) {
			continue
		}
		all = append(all, p)
	}
	return pageSlice(all, query.Page, query.PageSize)
}

func (s *MemoryStore) GetProvider(_ context.Context, id string) (Provider, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.providers[id]
	if !ok {
		return Provider{}, notFound("身份提供方不存在")
	}
	return p, nil
}

func (s *MemoryStore) CreateProvider(_ context.Context, item Provider, secretID *string) (Provider, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item.ID = newID()
	now := time.Now().UTC()
	item.CreatedAt = now
	item.UpdatedAt = now
	if secretID != nil {
		item.ClientSecretSet = true
		if sec, ok := s.secrets[*secretID]; ok {
			item.ClientSecretHint = sec.hint
		}
		s.providerSecret[item.ID] = *secretID
	}
	s.providers[item.ID] = item
	return item, nil
}

func (s *MemoryStore) UpdateProvider(_ context.Context, item Provider, secretID *string) (Provider, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.providers[item.ID]
	if !ok {
		return Provider{}, notFound("身份提供方不存在")
	}
	item.CreatedAt = existing.CreatedAt
	item.UpdatedAt = time.Now().UTC()
	item.ClientSecretSet = existing.ClientSecretSet
	item.ClientSecretHint = existing.ClientSecretHint
	if secretID != nil {
		item.ClientSecretSet = true
		if sec, ok := s.secrets[*secretID]; ok {
			item.ClientSecretHint = sec.hint
		}
		s.providerSecret[item.ID] = *secretID
	}
	s.providers[item.ID] = item
	return item, nil
}

func (s *MemoryStore) SetProviderStatus(_ context.Context, id, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.providers[id]
	if !ok {
		return notFound("身份提供方不存在")
	}
	p.Status = status
	p.UpdatedAt = time.Now().UTC()
	s.providers[id] = p
	return nil
}

func (s *MemoryStore) ListOAuthClients(_ context.Context, query ListQuery) ([]OAuthClient, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var all []OAuthClient
	for _, c := range s.clients {
		if query.Status != "" && c.Status != query.Status {
			continue
		}
		if query.Keyword != "" && !strings.Contains(strings.ToLower(c.Name+c.ClientID), strings.ToLower(query.Keyword)) {
			continue
		}
		all = append(all, c)
	}
	return pageSlice(all, query.Page, query.PageSize)
}

func (s *MemoryStore) GetOAuthClient(_ context.Context, id string) (OAuthClient, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.clients[id]
	if !ok {
		return OAuthClient{}, notFound("OAuth 客户端不存在")
	}
	return c, nil
}

func (s *MemoryStore) CreateOAuthClient(_ context.Context, item OAuthClient, secretID *string) (OAuthClient, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item.ID = newID()
	now := time.Now().UTC()
	item.CreatedAt = now
	item.UpdatedAt = now
	if secretID != nil {
		item.ClientSecretSet = true
		if sec, ok := s.secrets[*secretID]; ok {
			item.ClientSecretHint = sec.hint
		}
		s.clientSecret[item.ID] = *secretID
	}
	s.clients[item.ID] = item
	return item, nil
}

func (s *MemoryStore) UpdateOAuthClient(_ context.Context, item OAuthClient, secretID *string) (OAuthClient, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.clients[item.ID]
	if !ok {
		return OAuthClient{}, notFound("OAuth 客户端不存在")
	}
	item.CreatedAt = existing.CreatedAt
	item.UpdatedAt = time.Now().UTC()
	item.ClientSecretSet = existing.ClientSecretSet
	item.ClientSecretHint = existing.ClientSecretHint
	if secretID != nil {
		item.ClientSecretSet = true
		if sec, ok := s.secrets[*secretID]; ok {
			item.ClientSecretHint = sec.hint
		}
		s.clientSecret[item.ID] = *secretID
	}
	s.clients[item.ID] = item
	return item, nil
}

func (s *MemoryStore) SetOAuthClientStatus(_ context.Context, id, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.clients[id]
	if !ok {
		return notFound("OAuth 客户端不存在")
	}
	c.Status = status
	c.UpdatedAt = time.Now().UTC()
	s.clients[id] = c
	return nil
}

func (s *MemoryStore) ListSamlSPs(_ context.Context, query ListQuery) ([]SamlSP, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var all []SamlSP
	for _, sp := range s.sps {
		if query.Status != "" && sp.Status != query.Status {
			continue
		}
		if query.Keyword != "" && !strings.Contains(strings.ToLower(sp.Name+sp.EntityID), strings.ToLower(query.Keyword)) {
			continue
		}
		all = append(all, sp)
	}
	return pageSlice(all, query.Page, query.PageSize)
}

func (s *MemoryStore) GetSamlSP(_ context.Context, id string) (SamlSP, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sp, ok := s.sps[id]
	if !ok {
		return SamlSP{}, notFound("SAML 服务提供方不存在")
	}
	return sp, nil
}

func (s *MemoryStore) CreateSamlSP(_ context.Context, item SamlSP) (SamlSP, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item.ID = newID()
	now := time.Now().UTC()
	item.CreatedAt = now
	item.UpdatedAt = now
	s.sps[item.ID] = item
	return item, nil
}

func (s *MemoryStore) UpdateSamlSP(_ context.Context, item SamlSP) (SamlSP, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	existing, ok := s.sps[item.ID]
	if !ok {
		return SamlSP{}, notFound("SAML 服务提供方不存在")
	}
	item.CreatedAt = existing.CreatedAt
	item.UpdatedAt = time.Now().UTC()
	s.sps[item.ID] = item
	return item, nil
}

func (s *MemoryStore) SetSamlSPStatus(_ context.Context, id, status string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sp, ok := s.sps[id]
	if !ok {
		return notFound("SAML 服务提供方不存在")
	}
	sp.Status = status
	sp.UpdatedAt = time.Now().UTC()
	s.sps[id] = sp
	return nil
}

func (s *MemoryStore) ListIdentityLinks(_ context.Context, query ListQuery) ([]IdentityLink, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var all []IdentityLink
	for _, link := range s.links {
		if query.Keyword != "" && !strings.Contains(strings.ToLower(link.ExternalSubject+link.ProviderName+link.UserID), strings.ToLower(query.Keyword)) {
			continue
		}
		all = append(all, link)
	}
	return pageSlice(all, query.Page, query.PageSize)
}

func (s *MemoryStore) DeleteIdentityLink(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.links[id]; !ok {
		return notFound("身份映射不存在")
	}
	delete(s.links, id)
	return nil
}

func (s *MemoryStore) FindIdentityLink(_ context.Context, providerID, externalSubject string) (IdentityLink, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, link := range s.links {
		if link.ProviderID == providerID && link.ExternalSubject == externalSubject {
			return link, nil
		}
	}
	return IdentityLink{}, notFound("身份映射不存在")
}

func (s *MemoryStore) CreateIdentityLink(_ context.Context, item IdentityLink) (IdentityLink, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item.ID = newID()
	now := time.Now().UTC()
	item.CreatedAt = now
	item.UpdatedAt = now
	s.links[item.ID] = item
	return item, nil
}

func (s *MemoryStore) ListLoginProviders(context.Context) ([]LoginOption, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]LoginOption, 0)
	for _, p := range s.providers {
		if p.Status != StatusEnabled || !p.DisplayOnLogin {
			continue
		}
		out = append(out, LoginOption{
			ID: p.ID, Name: p.Name, Protocol: p.Protocol,
			DisplayOnLogin: p.DisplayOnLogin, AutoLogin: p.AutoLogin,
		})
	}
	return out, nil
}

func (s *MemoryStore) GetProviderSecretID(_ context.Context, providerID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.providerSecret[providerID]
	if !ok || id == "" {
		return "", notFound("提供方未配置 Client Secret")
	}
	return id, nil
}

func (s *MemoryStore) MarkProviderAuthResult(_ context.Context, providerID string, success bool, message string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.providers[providerID]
	if !ok {
		return notFound("身份提供方不存在")
	}
	if success {
		now := time.Now().UTC()
		p.LastSuccessAt = &now
		p.LastError = ""
	} else {
		p.LastError = message
	}
	p.UpdatedAt = time.Now().UTC()
	s.providers[providerID] = p
	return nil
}

func (s *MemoryStore) CreateAuthRequest(_ context.Context, item AuthRequest) (AuthRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item.ID = newID()
	item.CreatedAt = time.Now().UTC()
	s.authRequests[item.State] = item
	return item, nil
}

func (s *MemoryStore) ConsumeAuthRequest(_ context.Context, state string) (AuthRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.authRequests[state]
	if !ok {
		return AuthRequest{}, invalid("登录状态无效或已过期")
	}
	if item.ConsumedAt != nil {
		return AuthRequest{}, invalid("登录状态已被使用")
	}
	if time.Now().UTC().After(item.ExpiresAt) {
		return AuthRequest{}, invalid("登录状态无效或已过期")
	}
	now := time.Now().UTC()
	item.ConsumedAt = &now
	s.authRequests[state] = item
	return item, nil
}

func (s *MemoryStore) CreateLoginTicket(_ context.Context, item LoginTicket) (LoginTicket, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item.ID = newID()
	item.CreatedAt = time.Now().UTC()
	s.tickets[item.Ticket] = item
	return item, nil
}

func (s *MemoryStore) ConsumeLoginTicket(_ context.Context, ticket string) (LoginTicket, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.tickets[ticket]
	if !ok {
		return LoginTicket{}, invalid("登录票据无效或已过期")
	}
	if item.ConsumedAt != nil {
		return LoginTicket{}, invalid("登录票据已被使用")
	}
	if time.Now().UTC().After(item.ExpiresAt) {
		return LoginTicket{}, invalid("登录票据无效或已过期")
	}
	now := time.Now().UTC()
	item.ConsumedAt = &now
	s.tickets[ticket] = item
	return item, nil
}

func (s *MemoryStore) InsertAuthEvent(_ context.Context, item AuthEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if item.ID == "" {
		item.ID = newID()
	}
	if item.CreateTime.IsZero() {
		item.CreateTime = time.Now().UTC()
	}
	s.events = append(s.events, item)
	return nil
}

func (s *MemoryStore) ListAuthEvents(_ context.Context, query ListQuery) ([]AuthEvent, int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var all []AuthEvent
	for _, e := range s.events {
		if query.Protocol != "" && e.Protocol != query.Protocol {
			continue
		}
		if query.Result != "" && e.Result != query.Result {
			continue
		}
		if query.Keyword != "" && !strings.Contains(strings.ToLower(e.EventType+e.Message+e.Username+e.ClientID), strings.ToLower(query.Keyword)) {
			continue
		}
		all = append(all, e)
	}
	return pageSlice(all, query.Page, query.PageSize)
}

func (s *MemoryStore) CreateSecret(_ context.Context, kind, hint string, ciphertext []byte) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := newID()
	s.secrets[id] = secretMem{kind: kind, hint: hint, ciphertext: append([]byte{}, ciphertext...)}
	return id, nil
}

func (s *MemoryStore) GetSecretHint(_ context.Context, id string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sec, ok := s.secrets[id]
	if !ok || sec.revoked {
		return "", nil
	}
	return sec.hint, nil
}

func (s *MemoryStore) GetSecretCiphertext(_ context.Context, id string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sec, ok := s.secrets[id]
	if !ok || sec.revoked {
		return nil, notFound("密钥不存在或已吊销")
	}
	return append([]byte{}, sec.ciphertext...), nil
}

func (s *MemoryStore) RevokeSecret(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sec, ok := s.secrets[id]
	if !ok {
		return nil
	}
	sec.revoked = true
	s.secrets[id] = sec
	return nil
}

func pageSlice[T any](all []T, page, pageSize int) ([]T, int64, error) {
	total := int64(len(all))
	start := (page - 1) * pageSize
	if start >= len(all) {
		return []T{}, total, nil
	}
	end := start + pageSize
	if end > len(all) {
		end = len(all)
	}
	return all[start:end], total, nil
}
