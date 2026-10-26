package sso

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// GormStore stores SSO data in PostgreSQL.
type GormStore struct {
	db *gorm.DB
}

// NewGormStore returns a PostgreSQL store.
func NewGormStore(db *gorm.DB) *GormStore {
	return &GormStore{db: db}
}

type settingsRow struct {
	ID                int16     `gorm:"column:id;primaryKey"`
	LocalLoginEnabled bool      `gorm:"column:local_login_enabled"`
	Issuer            string    `gorm:"column:issuer"`
	DefaultTenantID   string    `gorm:"column:default_tenant_id"`
	FrontendBaseURL   string    `gorm:"column:frontend_base_url"`
	UpdatedBy         string    `gorm:"column:updated_by"`
	UpdatedAt         time.Time `gorm:"column:updated_at"`
}

type authRequestRow struct {
	ID           string     `gorm:"column:id;type:uuid;primaryKey"`
	State        string     `gorm:"column:state"`
	Nonce        string     `gorm:"column:nonce"`
	CodeVerifier string     `gorm:"column:code_verifier"`
	ProviderID   string     `gorm:"column:provider_id;type:uuid"`
	Protocol     string     `gorm:"column:protocol"`
	RedirectPath string     `gorm:"column:redirect_path"`
	TenantID     string     `gorm:"column:tenant_id"`
	ExpiresAt    time.Time  `gorm:"column:expires_at"`
	ConsumedAt   *time.Time `gorm:"column:consumed_at"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
}

func (authRequestRow) TableName() string { return "sys_sso_oidc_auth_requests" }

type loginTicketRow struct {
	ID         string     `gorm:"column:id;type:uuid;primaryKey"`
	Ticket     string     `gorm:"column:ticket"`
	UserID     string     `gorm:"column:user_id;type:uuid"`
	TenantID   string     `gorm:"column:tenant_id"`
	ProviderID string     `gorm:"column:provider_id;type:uuid"`
	ExpiresAt  time.Time  `gorm:"column:expires_at"`
	ConsumedAt *time.Time `gorm:"column:consumed_at"`
	CreatedAt  time.Time  `gorm:"column:created_at"`
}

func (loginTicketRow) TableName() string { return "sys_sso_login_tickets" }

func (settingsRow) TableName() string { return "sys_sso_settings" }

type secretRow struct {
	ID         string     `gorm:"column:id;type:uuid;primaryKey"`
	Kind       string     `gorm:"column:kind"`
	Ciphertext []byte     `gorm:"column:ciphertext"`
	KeyVersion int        `gorm:"column:key_version"`
	Hint       string     `gorm:"column:hint"`
	CreatedAt  time.Time  `gorm:"column:created_at"`
	RevokedAt  *time.Time `gorm:"column:revoked_at"`
}

func (secretRow) TableName() string { return "sys_sso_secrets" }

type providerRow struct {
	ID                   string         `gorm:"column:id;type:uuid;primaryKey"`
	Name                 string         `gorm:"column:name"`
	Protocol             string         `gorm:"column:protocol"`
	Status               string         `gorm:"column:status"`
	Priority             int            `gorm:"column:priority"`
	Issuer               string         `gorm:"column:issuer"`
	DisplayOnLogin       bool           `gorm:"column:display_on_login"`
	AutoLogin            bool           `gorm:"column:auto_login"`
	AutoCreateUser       bool           `gorm:"column:auto_create_user"`
	DiscoveryURL         string         `gorm:"column:discovery_url"`
	AuthorizationURL     string         `gorm:"column:authorization_url"`
	TokenURL             string         `gorm:"column:token_url"`
	UserinfoURL          string         `gorm:"column:userinfo_url"`
	JwksURL              string         `gorm:"column:jwks_url"`
	ClientID             string         `gorm:"column:client_id"`
	SecretID             *string        `gorm:"column:secret_id;type:uuid"`
	RedirectURI          string         `gorm:"column:redirect_uri"`
	Scopes               string         `gorm:"column:scopes"`
	ClaimMapping         map[string]any `gorm:"serializer:json;column:claim_mapping"`
	EntityID             string         `gorm:"column:entity_id"`
	MetadataURL          string         `gorm:"column:metadata_url"`
	SSOURL               string         `gorm:"column:sso_url"`
	SLOURL               string         `gorm:"column:slo_url"`
	IDPCertificatePEM    string         `gorm:"column:idp_certificate_pem"`
	CertificateNotBefore *time.Time     `gorm:"column:certificate_not_before"`
	CertificateNotAfter  *time.Time     `gorm:"column:certificate_not_after"`
	SPEntityID           string         `gorm:"column:sp_entity_id"`
	ACSURL               string         `gorm:"column:acs_url"`
	NameIDFormat         string         `gorm:"column:name_id_format"`
	LastSuccessAt        *time.Time     `gorm:"column:last_success_at"`
	LastError            string         `gorm:"column:last_error"`
	CreatedAt            time.Time      `gorm:"column:created_at"`
	UpdatedAt            time.Time      `gorm:"column:updated_at"`
	Deleted              bool           `gorm:"column:deleted"`
}

func (providerRow) TableName() string { return "sys_sso_providers" }

type oauthClientRow struct {
	ID                 string         `gorm:"column:id;type:uuid;primaryKey"`
	ClientID           string         `gorm:"column:client_id"`
	Name               string         `gorm:"column:name"`
	Description        string         `gorm:"column:description"`
	ClientType         string         `gorm:"column:client_type"`
	Status             string         `gorm:"column:status"`
	RedirectURIs       []string `gorm:"serializer:json;column:redirect_uris"`
	GrantTypes         []string `gorm:"serializer:json;column:grant_types"`
	Scopes             []string `gorm:"serializer:json;column:scopes"`
	PKCEPolicy         string         `gorm:"column:pkce_policy"`
	AccessTokenTTLSec  int            `gorm:"column:access_token_ttl_sec"`
	RefreshTokenTTLSec int            `gorm:"column:refresh_token_ttl_sec"`
	IDTokenTTLSec      int            `gorm:"column:id_token_ttl_sec"`
	SecretID           *string        `gorm:"column:secret_id;type:uuid"`
	LastUsedAt         *time.Time     `gorm:"column:last_used_at"`
	CreatedAt          time.Time      `gorm:"column:created_at"`
	UpdatedAt          time.Time      `gorm:"column:updated_at"`
	Deleted            bool           `gorm:"column:deleted"`
}

func (oauthClientRow) TableName() string { return "sys_sso_oauth_clients" }

type samlSPRow struct {
	ID                   string         `gorm:"column:id;type:uuid;primaryKey"`
	Name                 string         `gorm:"column:name"`
	EntityID             string         `gorm:"column:entity_id"`
	Status               string         `gorm:"column:status"`
	ACSURLs              []string       `gorm:"serializer:json;column:acs_urls"`
	SLOURL               string         `gorm:"column:slo_url"`
	NameIDFormat         string         `gorm:"column:name_id_format"`
	AttributeMapping     map[string]any `gorm:"serializer:json;column:attribute_mapping"`
	AllowIDPInitiated    bool           `gorm:"column:allow_idp_initiated"`
	CertificatePEM       string         `gorm:"column:certificate_pem"`
	CertificateNotBefore *time.Time     `gorm:"column:certificate_not_before"`
	CertificateNotAfter  *time.Time     `gorm:"column:certificate_not_after"`
	CreatedAt            time.Time      `gorm:"column:created_at"`
	UpdatedAt            time.Time      `gorm:"column:updated_at"`
	Deleted              bool           `gorm:"column:deleted"`
}

func (samlSPRow) TableName() string { return "sys_sso_saml_sps" }

type identityLinkRow struct {
	ID              string    `gorm:"column:id;type:uuid;primaryKey"`
	ProviderID      string    `gorm:"column:provider_id;type:uuid"`
	ExternalSubject string    `gorm:"column:external_subject"`
	UserID          string    `gorm:"column:user_id;type:uuid"`
	CreatedAt       time.Time `gorm:"column:created_at"`
	UpdatedAt       time.Time `gorm:"column:updated_at"`
	Deleted         bool      `gorm:"column:deleted"`
}

func (identityLinkRow) TableName() string { return "sys_sso_identity_links" }

type authEventRow struct {
	ID         string    `gorm:"column:id;type:uuid;primaryKey"`
	EventType  string    `gorm:"column:event_type"`
	Result     string    `gorm:"column:result"`
	Protocol   string    `gorm:"column:protocol"`
	ProviderID string    `gorm:"column:provider_id"`
	ClientID   string    `gorm:"column:client_id"`
	UserID     string    `gorm:"column:user_id"`
	Username   string    `gorm:"column:username"`
	Message    string    `gorm:"column:message"`
	RequestID  string    `gorm:"column:request_id"`
	IP         string    `gorm:"column:ip"`
	CreateTime time.Time `gorm:"column:create_time"`
}

func (authEventRow) TableName() string { return "sys_sso_auth_events" }

func (s *GormStore) Settings(ctx context.Context) (Settings, error) {
	var row settingsRow
	err := s.db.WithContext(ctx).First(&row, "id = ?", 1).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Settings{LocalLoginEnabled: true, UpdatedAt: time.Now().UTC()}, nil
	}
	if err != nil {
		return Settings{}, err
	}
	return Settings{
		LocalLoginEnabled: row.LocalLoginEnabled,
		Issuer:            row.Issuer,
		DefaultTenantID:   row.DefaultTenantID,
		FrontendBaseURL:   row.FrontendBaseURL,
		Updater:           row.UpdatedBy,
		UpdatedAt:         row.UpdatedAt,
	}, nil
}

func (s *GormStore) SaveSettings(ctx context.Context, item Settings) error {
	row := settingsRow{
		ID:                1,
		LocalLoginEnabled: item.LocalLoginEnabled,
		Issuer:            item.Issuer,
		DefaultTenantID:   item.DefaultTenantID,
		FrontendBaseURL:   item.FrontendBaseURL,
		UpdatedBy:         item.Updater,
		UpdatedAt:         item.UpdatedAt,
	}
	return s.db.WithContext(ctx).Save(&row).Error
}

func (s *GormStore) CountProviders(ctx context.Context, protocol string) (int64, int64, error) {
	var total, enabled int64
	q := s.db.WithContext(ctx).Model(&providerRow{}).Where("deleted = false AND protocol = ?", protocol)
	if err := q.Count(&total).Error; err != nil {
		return 0, 0, err
	}
	if err := s.db.WithContext(ctx).Model(&providerRow{}).
		Where("deleted = false AND protocol = ? AND status = ?", protocol, StatusEnabled).
		Count(&enabled).Error; err != nil {
		return 0, 0, err
	}
	return total, enabled, nil
}

func (s *GormStore) CountOAuthClients(ctx context.Context) (int64, int64, error) {
	var total, enabled int64
	if err := s.db.WithContext(ctx).Model(&oauthClientRow{}).Where("deleted = false").Count(&total).Error; err != nil {
		return 0, 0, err
	}
	if err := s.db.WithContext(ctx).Model(&oauthClientRow{}).
		Where("deleted = false AND status = ?", StatusEnabled).Count(&enabled).Error; err != nil {
		return 0, 0, err
	}
	return total, enabled, nil
}

func (s *GormStore) CountSamlSPs(ctx context.Context) (int64, int64, error) {
	var total, enabled int64
	if err := s.db.WithContext(ctx).Model(&samlSPRow{}).Where("deleted = false").Count(&total).Error; err != nil {
		return 0, 0, err
	}
	if err := s.db.WithContext(ctx).Model(&samlSPRow{}).
		Where("deleted = false AND status = ?", StatusEnabled).Count(&enabled).Error; err != nil {
		return 0, 0, err
	}
	return total, enabled, nil
}

func (s *GormStore) CountIdentityLinks(ctx context.Context) (int64, error) {
	var total int64
	err := s.db.WithContext(ctx).Model(&identityLinkRow{}).Where("deleted = false").Count(&total).Error
	return total, err
}

func (s *GormStore) CountAuthEventsSince(ctx context.Context, result, eventTypePrefix string, since time.Time) (int64, error) {
	var total int64
	q := s.db.WithContext(ctx).Model(&authEventRow{}).
		Where("result = ? AND create_time >= ?", result, since)
	if eventTypePrefix != "" {
		q = q.Where("event_type LIKE ?", eventTypePrefix+"%")
	}
	err := q.Count(&total).Error
	return total, err
}

func (s *GormStore) ListRecentAuthEvents(ctx context.Context, result string, limit int) ([]AuthEvent, error) {
	var rows []authEventRow
	q := s.db.WithContext(ctx).Order("create_time DESC").Limit(limit)
	if result != "" {
		q = q.Where("result = ?", result)
	}
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	return authEventsFrom(rows), nil
}

func (s *GormStore) ListCertificateWarnings(ctx context.Context, withinDays int) ([]CertWarning, error) {
	deadline := time.Now().UTC().AddDate(0, 0, withinDays)
	var warnings []CertWarning

	var providers []providerRow
	if err := s.db.WithContext(ctx).
		Where("deleted = false AND certificate_not_after IS NOT NULL AND certificate_not_after <= ?", deadline).
		Find(&providers).Error; err != nil {
		return nil, err
	}
	for _, p := range providers {
		warnings = append(warnings, certWarning("provider", p.ID, p.Name, p.CertificateNotAfter))
	}

	var sps []samlSPRow
	if err := s.db.WithContext(ctx).
		Where("deleted = false AND certificate_not_after IS NOT NULL AND certificate_not_after <= ?", deadline).
		Find(&sps).Error; err != nil {
		return nil, err
	}
	for _, sp := range sps {
		warnings = append(warnings, certWarning("saml_sp", sp.ID, sp.Name, sp.CertificateNotAfter))
	}
	return warnings, nil
}

func certWarning(kind, id, name string, notAfter *time.Time) CertWarning {
	w := CertWarning{Kind: kind, RefID: id, Name: name, NotAfter: notAfter}
	if notAfter != nil {
		days := int(time.Until(*notAfter).Hours() / 24)
		w.DaysLeft = &days
		w.Expired = notAfter.Before(time.Now().UTC())
	}
	return w
}

func (s *GormStore) ListProviders(ctx context.Context, query ListQuery) ([]Provider, int64, error) {
	q := s.db.WithContext(ctx).Model(&providerRow{}).Where("deleted = false")
	if query.Protocol != "" {
		q = q.Where("protocol = ?", query.Protocol)
	}
	if query.Status != "" {
		q = q.Where("status = ?", query.Status)
	}
	if query.Keyword != "" {
		like := "%" + query.Keyword + "%"
		q = q.Where("name ILIKE ? OR issuer ILIKE ? OR client_id ILIKE ?", like, like, like)
	}
	items, total, err := pageQuery(q, query.Page, query.PageSize, "priority ASC, updated_at DESC", func(rows []providerRow) []Provider {
		out := make([]Provider, 0, len(rows))
		for _, row := range rows {
			out = append(out, s.providerFrom(ctx, row))
		}
		return out
	})
	return items, total, err
}

func (s *GormStore) GetProvider(ctx context.Context, id string) (Provider, error) {
	var row providerRow
	err := s.db.WithContext(ctx).Where("id = ? AND deleted = false", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Provider{}, notFound("身份提供方不存在")
	}
	if err != nil {
		return Provider{}, err
	}
	return s.providerFrom(ctx, row), nil
}

func (s *GormStore) CreateProvider(ctx context.Context, item Provider, secretID *string) (Provider, error) {
	row := providerToRow(item)
	row.ID = newID()
	row.SecretID = secretID
	now := time.Now().UTC()
	row.CreatedAt = now
	row.UpdatedAt = now
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return Provider{}, err
	}
	return s.providerFrom(ctx, row), nil
}

func (s *GormStore) UpdateProvider(ctx context.Context, item Provider, secretID *string) (Provider, error) {
	var row providerRow
	err := s.db.WithContext(ctx).Where("id = ? AND deleted = false", item.ID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Provider{}, notFound("身份提供方不存在")
	}
	if err != nil {
		return Provider{}, err
	}
	updated := providerToRow(item)
	updated.ID = row.ID
	updated.CreatedAt = row.CreatedAt
	updated.UpdatedAt = time.Now().UTC()
	updated.Deleted = false
	if secretID != nil {
		updated.SecretID = secretID
	} else {
		updated.SecretID = row.SecretID
	}
	updated.LastSuccessAt = row.LastSuccessAt
	if item.LastError != "" || row.LastError != "" {
		updated.LastError = item.LastError
	}
	if err := s.db.WithContext(ctx).Save(&updated).Error; err != nil {
		return Provider{}, err
	}
	return s.providerFrom(ctx, updated), nil
}

func (s *GormStore) SetProviderStatus(ctx context.Context, id, status string) error {
	res := s.db.WithContext(ctx).Model(&providerRow{}).
		Where("id = ? AND deleted = false", id).
		Updates(map[string]any{"status": status, "updated_at": time.Now().UTC()})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return notFound("身份提供方不存在")
	}
	return nil
}

func (s *GormStore) ListOAuthClients(ctx context.Context, query ListQuery) ([]OAuthClient, int64, error) {
	q := s.db.WithContext(ctx).Model(&oauthClientRow{}).Where("deleted = false")
	if query.Status != "" {
		q = q.Where("status = ?", query.Status)
	}
	if query.Keyword != "" {
		like := "%" + query.Keyword + "%"
		q = q.Where("name ILIKE ? OR client_id ILIKE ?", like, like)
	}
	return pageQuery(q, query.Page, query.PageSize, "updated_at DESC", func(rows []oauthClientRow) []OAuthClient {
		out := make([]OAuthClient, 0, len(rows))
		for _, row := range rows {
			out = append(out, s.oauthClientFrom(ctx, row))
		}
		return out
	})
}

func (s *GormStore) GetOAuthClient(ctx context.Context, id string) (OAuthClient, error) {
	var row oauthClientRow
	err := s.db.WithContext(ctx).Where("id = ? AND deleted = false", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return OAuthClient{}, notFound("OAuth 客户端不存在")
	}
	if err != nil {
		return OAuthClient{}, err
	}
	return s.oauthClientFrom(ctx, row), nil
}

func (s *GormStore) CreateOAuthClient(ctx context.Context, item OAuthClient, secretID *string) (OAuthClient, error) {
	row := oauthClientToRow(item)
	row.ID = newID()
	row.SecretID = secretID
	now := time.Now().UTC()
	row.CreatedAt = now
	row.UpdatedAt = now
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return OAuthClient{}, err
	}
	return s.oauthClientFrom(ctx, row), nil
}

func (s *GormStore) UpdateOAuthClient(ctx context.Context, item OAuthClient, secretID *string) (OAuthClient, error) {
	var row oauthClientRow
	err := s.db.WithContext(ctx).Where("id = ? AND deleted = false", item.ID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return OAuthClient{}, notFound("OAuth 客户端不存在")
	}
	if err != nil {
		return OAuthClient{}, err
	}
	updated := oauthClientToRow(item)
	updated.ID = row.ID
	updated.ClientID = row.ClientID
	updated.CreatedAt = row.CreatedAt
	updated.UpdatedAt = time.Now().UTC()
	updated.LastUsedAt = row.LastUsedAt
	updated.Deleted = false
	if secretID != nil {
		updated.SecretID = secretID
	} else {
		updated.SecretID = row.SecretID
	}
	if err := s.db.WithContext(ctx).Save(&updated).Error; err != nil {
		return OAuthClient{}, err
	}
	return s.oauthClientFrom(ctx, updated), nil
}

func (s *GormStore) SetOAuthClientStatus(ctx context.Context, id, status string) error {
	res := s.db.WithContext(ctx).Model(&oauthClientRow{}).
		Where("id = ? AND deleted = false", id).
		Updates(map[string]any{"status": status, "updated_at": time.Now().UTC()})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return notFound("OAuth 客户端不存在")
	}
	return nil
}

func (s *GormStore) ListSamlSPs(ctx context.Context, query ListQuery) ([]SamlSP, int64, error) {
	q := s.db.WithContext(ctx).Model(&samlSPRow{}).Where("deleted = false")
	if query.Status != "" {
		q = q.Where("status = ?", query.Status)
	}
	if query.Keyword != "" {
		like := "%" + query.Keyword + "%"
		q = q.Where("name ILIKE ? OR entity_id ILIKE ?", like, like)
	}
	return pageQuery(q, query.Page, query.PageSize, "updated_at DESC", func(rows []samlSPRow) []SamlSP {
		out := make([]SamlSP, 0, len(rows))
		for _, row := range rows {
			out = append(out, samlSPFrom(row))
		}
		return out
	})
}

func (s *GormStore) GetSamlSP(ctx context.Context, id string) (SamlSP, error) {
	var row samlSPRow
	err := s.db.WithContext(ctx).Where("id = ? AND deleted = false", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return SamlSP{}, notFound("SAML 服务提供方不存在")
	}
	if err != nil {
		return SamlSP{}, err
	}
	return samlSPFrom(row), nil
}

func (s *GormStore) CreateSamlSP(ctx context.Context, item SamlSP) (SamlSP, error) {
	row := samlSPToRow(item)
	row.ID = newID()
	now := time.Now().UTC()
	row.CreatedAt = now
	row.UpdatedAt = now
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return SamlSP{}, err
	}
	return samlSPFrom(row), nil
}

func (s *GormStore) UpdateSamlSP(ctx context.Context, item SamlSP) (SamlSP, error) {
	var row samlSPRow
	err := s.db.WithContext(ctx).Where("id = ? AND deleted = false", item.ID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return SamlSP{}, notFound("SAML 服务提供方不存在")
	}
	if err != nil {
		return SamlSP{}, err
	}
	updated := samlSPToRow(item)
	updated.ID = row.ID
	updated.CreatedAt = row.CreatedAt
	updated.UpdatedAt = time.Now().UTC()
	updated.Deleted = false
	if err := s.db.WithContext(ctx).Save(&updated).Error; err != nil {
		return SamlSP{}, err
	}
	return samlSPFrom(updated), nil
}

func (s *GormStore) SetSamlSPStatus(ctx context.Context, id, status string) error {
	res := s.db.WithContext(ctx).Model(&samlSPRow{}).
		Where("id = ? AND deleted = false", id).
		Updates(map[string]any{"status": status, "updated_at": time.Now().UTC()})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return notFound("SAML 服务提供方不存在")
	}
	return nil
}

func (s *GormStore) ListIdentityLinks(ctx context.Context, query ListQuery) ([]IdentityLink, int64, error) {
	type linkJoin struct {
		identityLinkRow
		ProviderName string `gorm:"column:provider_name"`
		Protocol     string `gorm:"column:protocol"`
	}
	q := s.db.WithContext(ctx).Table("sys_sso_identity_links AS l").
		Select("l.*, p.name AS provider_name, p.protocol AS protocol").
		Joins("JOIN sys_sso_providers p ON p.id = l.provider_id").
		Where("l.deleted = false AND p.deleted = false")
	if query.Keyword != "" {
		like := "%" + query.Keyword + "%"
		q = q.Where("l.external_subject ILIKE ? OR p.name ILIKE ? OR CAST(l.user_id AS text) ILIKE ?", like, like, like)
	}
	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []linkJoin
	err := q.Session(&gorm.Session{}).Order("l.updated_at DESC").
		Limit(query.PageSize).Offset((query.Page - 1) * query.PageSize).
		Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	out := make([]IdentityLink, 0, len(rows))
	for _, row := range rows {
		out = append(out, IdentityLink{
			ID:              row.ID,
			ProviderID:      row.ProviderID,
			ProviderName:    row.ProviderName,
			Protocol:        row.Protocol,
			ExternalSubject: row.ExternalSubject,
			UserID:          row.UserID,
			CreatedAt:       row.CreatedAt,
			UpdatedAt:       row.UpdatedAt,
		})
	}
	return out, total, nil
}

func (s *GormStore) DeleteIdentityLink(ctx context.Context, id string) error {
	res := s.db.WithContext(ctx).Model(&identityLinkRow{}).
		Where("id = ? AND deleted = false", id).
		Updates(map[string]any{"deleted": true, "updated_at": time.Now().UTC()})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return notFound("身份映射不存在")
	}
	return nil
}

func (s *GormStore) FindIdentityLink(ctx context.Context, providerID, externalSubject string) (IdentityLink, error) {
	var row identityLinkRow
	err := s.db.WithContext(ctx).
		Where("provider_id = ? AND external_subject = ? AND deleted = false", providerID, externalSubject).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return IdentityLink{}, notFound("身份映射不存在")
	}
	if err != nil {
		return IdentityLink{}, err
	}
	return IdentityLink{
		ID:              row.ID,
		ProviderID:      row.ProviderID,
		ExternalSubject: row.ExternalSubject,
		UserID:          row.UserID,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}, nil
}

func (s *GormStore) CreateIdentityLink(ctx context.Context, item IdentityLink) (IdentityLink, error) {
	now := time.Now().UTC()
	row := identityLinkRow{
		ID:              newID(),
		ProviderID:      item.ProviderID,
		ExternalSubject: item.ExternalSubject,
		UserID:          item.UserID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return IdentityLink{}, err
	}
	item.ID = row.ID
	item.CreatedAt = row.CreatedAt
	item.UpdatedAt = row.UpdatedAt
	return item, nil
}

func (s *GormStore) ListLoginProviders(ctx context.Context) ([]LoginOption, error) {
	var rows []providerRow
	err := s.db.WithContext(ctx).
		Where("deleted = false AND status = ? AND display_on_login = true", StatusEnabled).
		Order("priority ASC, updated_at DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]LoginOption, 0, len(rows))
	for _, row := range rows {
		out = append(out, LoginOption{
			ID:             row.ID,
			Name:           row.Name,
			Protocol:       row.Protocol,
			DisplayOnLogin: row.DisplayOnLogin,
			AutoLogin:      row.AutoLogin,
		})
	}
	return out, nil
}

func (s *GormStore) GetProviderSecretID(ctx context.Context, providerID string) (string, error) {
	var row providerRow
	err := s.db.WithContext(ctx).Select("secret_id").Where("id = ? AND deleted = false", providerID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", notFound("身份提供方不存在")
	}
	if err != nil {
		return "", err
	}
	if row.SecretID == nil || *row.SecretID == "" {
		return "", notFound("提供方未配置 Client Secret")
	}
	return *row.SecretID, nil
}

func (s *GormStore) MarkProviderAuthResult(ctx context.Context, providerID string, success bool, message string) error {
	updates := map[string]any{
		"updated_at": time.Now().UTC(),
		"last_error": clip(message, 500),
	}
	if success {
		now := time.Now().UTC()
		updates["last_success_at"] = now
		updates["last_error"] = ""
	}
	res := s.db.WithContext(ctx).Model(&providerRow{}).Where("id = ? AND deleted = false", providerID).Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return notFound("身份提供方不存在")
	}
	return nil
}

func (s *GormStore) CreateAuthRequest(ctx context.Context, item AuthRequest) (AuthRequest, error) {
	row := authRequestRow{
		ID:           newID(),
		State:        item.State,
		Nonce:        item.Nonce,
		CodeVerifier: item.CodeVerifier,
		ProviderID:   item.ProviderID,
		Protocol:     item.Protocol,
		RedirectPath: item.RedirectPath,
		TenantID:     item.TenantID,
		ExpiresAt:    item.ExpiresAt,
		CreatedAt:    time.Now().UTC(),
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return AuthRequest{}, err
	}
	item.ID = row.ID
	item.CreatedAt = row.CreatedAt
	return item, nil
}

func (s *GormStore) ConsumeAuthRequest(ctx context.Context, state string) (AuthRequest, error) {
	var row authRequestRow
	err := s.db.WithContext(ctx).Where("state = ?", state).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return AuthRequest{}, invalid("登录状态无效或已过期")
	}
	if err != nil {
		return AuthRequest{}, err
	}
	if row.ConsumedAt != nil {
		return AuthRequest{}, invalid("登录状态已被使用")
	}
	if time.Now().UTC().After(row.ExpiresAt) {
		return AuthRequest{}, invalid("登录状态无效或已过期")
	}
	now := time.Now().UTC()
	res := s.db.WithContext(ctx).Model(&authRequestRow{}).
		Where("id = ? AND consumed_at IS NULL", row.ID).
		Update("consumed_at", now)
	if res.Error != nil {
		return AuthRequest{}, res.Error
	}
	if res.RowsAffected == 0 {
		return AuthRequest{}, invalid("登录状态已被使用")
	}
	return AuthRequest{
		ID:           row.ID,
		State:        row.State,
		Nonce:        row.Nonce,
		CodeVerifier: row.CodeVerifier,
		ProviderID:   row.ProviderID,
		Protocol:     row.Protocol,
		RedirectPath: row.RedirectPath,
		TenantID:     row.TenantID,
		ExpiresAt:    row.ExpiresAt,
		ConsumedAt:   &now,
		CreatedAt:    row.CreatedAt,
	}, nil
}

func (s *GormStore) CreateLoginTicket(ctx context.Context, item LoginTicket) (LoginTicket, error) {
	row := loginTicketRow{
		ID:         newID(),
		Ticket:     item.Ticket,
		UserID:     item.UserID,
		TenantID:   item.TenantID,
		ProviderID: item.ProviderID,
		ExpiresAt:  item.ExpiresAt,
		CreatedAt:  time.Now().UTC(),
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return LoginTicket{}, err
	}
	item.ID = row.ID
	item.CreatedAt = row.CreatedAt
	return item, nil
}

func (s *GormStore) ConsumeLoginTicket(ctx context.Context, ticket string) (LoginTicket, error) {
	var row loginTicketRow
	err := s.db.WithContext(ctx).Where("ticket = ?", ticket).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return LoginTicket{}, invalid("登录票据无效或已过期")
	}
	if err != nil {
		return LoginTicket{}, err
	}
	if row.ConsumedAt != nil {
		return LoginTicket{}, invalid("登录票据已被使用")
	}
	if time.Now().UTC().After(row.ExpiresAt) {
		return LoginTicket{}, invalid("登录票据无效或已过期")
	}
	now := time.Now().UTC()
	res := s.db.WithContext(ctx).Model(&loginTicketRow{}).
		Where("id = ? AND consumed_at IS NULL", row.ID).
		Update("consumed_at", now)
	if res.Error != nil {
		return LoginTicket{}, res.Error
	}
	if res.RowsAffected == 0 {
		return LoginTicket{}, invalid("登录票据已被使用")
	}
	return LoginTicket{
		ID:         row.ID,
		Ticket:     row.Ticket,
		UserID:     row.UserID,
		TenantID:   row.TenantID,
		ProviderID: row.ProviderID,
		ExpiresAt:  row.ExpiresAt,
		ConsumedAt: &now,
		CreatedAt:  row.CreatedAt,
	}, nil
}

func (s *GormStore) InsertAuthEvent(ctx context.Context, item AuthEvent) error {
	row := authEventRow{
		ID:         item.ID,
		EventType:  item.EventType,
		Result:     item.Result,
		Protocol:   item.Protocol,
		ProviderID: item.ProviderID,
		ClientID:   item.ClientID,
		UserID:     item.UserID,
		Username:   item.Username,
		Message:    item.Message,
		RequestID:  item.RequestID,
		IP:         item.IP,
		CreateTime: item.CreateTime,
	}
	if row.ID == "" {
		row.ID = newID()
	}
	if row.CreateTime.IsZero() {
		row.CreateTime = time.Now().UTC()
	}
	return s.db.WithContext(ctx).Create(&row).Error
}

func (s *GormStore) ListAuthEvents(ctx context.Context, query ListQuery) ([]AuthEvent, int64, error) {
	q := s.db.WithContext(ctx).Model(&authEventRow{})
	if query.Protocol != "" {
		q = q.Where("protocol = ?", query.Protocol)
	}
	if query.Result != "" {
		q = q.Where("result = ?", query.Result)
	}
	if query.Keyword != "" {
		like := "%" + query.Keyword + "%"
		q = q.Where("event_type ILIKE ? OR message ILIKE ? OR username ILIKE ? OR client_id ILIKE ?", like, like, like, like)
	}
	return pageQuery(q, query.Page, query.PageSize, "create_time DESC", authEventsFrom)
}

func (s *GormStore) CreateSecret(ctx context.Context, kind, hint string, ciphertext []byte) (string, error) {
	row := secretRow{
		ID:         newID(),
		Kind:       kind,
		Ciphertext: ciphertext,
		KeyVersion: 1,
		Hint:       hint,
		CreatedAt:  time.Now().UTC(),
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return "", err
	}
	return row.ID, nil
}

func (s *GormStore) GetSecretHint(ctx context.Context, id string) (string, error) {
	var row secretRow
	err := s.db.WithContext(ctx).Select("hint").Where("id = ? AND revoked_at IS NULL", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", nil
	}
	return row.Hint, err
}

func (s *GormStore) GetSecretCiphertext(ctx context.Context, id string) ([]byte, error) {
	var row secretRow
	err := s.db.WithContext(ctx).Where("id = ? AND revoked_at IS NULL", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, notFound("密钥不存在或已吊销")
	}
	if err != nil {
		return nil, err
	}
	return append([]byte{}, row.Ciphertext...), nil
}

func (s *GormStore) RevokeSecret(ctx context.Context, id string) error {
	now := time.Now().UTC()
	return s.db.WithContext(ctx).Model(&secretRow{}).Where("id = ?", id).Update("revoked_at", now).Error
}

func (s *GormStore) providerFrom(ctx context.Context, row providerRow) Provider {
	item := Provider{
		ID:                   row.ID,
		Name:                 row.Name,
		Protocol:             row.Protocol,
		Status:               row.Status,
		Priority:             row.Priority,
		Issuer:               row.Issuer,
		DisplayOnLogin:       row.DisplayOnLogin,
		AutoLogin:            row.AutoLogin,
		AutoCreateUser:       row.AutoCreateUser,
		DiscoveryURL:         row.DiscoveryURL,
		AuthorizationURL:     row.AuthorizationURL,
		TokenURL:             row.TokenURL,
		UserinfoURL:          row.UserinfoURL,
		JwksURL:              row.JwksURL,
		ClientID:             row.ClientID,
		ClientSecretSet:      row.SecretID != nil && *row.SecretID != "",
		RedirectURI:          row.RedirectURI,
		Scopes:               row.Scopes,
		ClaimMapping:         cloneMap(row.ClaimMapping),
		EntityID:             row.EntityID,
		MetadataURL:          row.MetadataURL,
		SSOURL:               row.SSOURL,
		SLOURL:               row.SLOURL,
		IDPCertificatePEM:    row.IDPCertificatePEM,
		CertificateNotBefore: row.CertificateNotBefore,
		CertificateNotAfter:  row.CertificateNotAfter,
		SPEntityID:           row.SPEntityID,
		ACSURL:               row.ACSURL,
		NameIDFormat:         row.NameIDFormat,
		LastSuccessAt:        row.LastSuccessAt,
		LastError:            row.LastError,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
	if row.SecretID != nil && *row.SecretID != "" {
		if hint, err := s.GetSecretHint(ctx, *row.SecretID); err == nil {
			item.ClientSecretHint = hint
		}
	}
	return item
}

func providerToRow(item Provider) providerRow {
	return providerRow{
		ID:                   item.ID,
		Name:                 item.Name,
		Protocol:             item.Protocol,
		Status:               item.Status,
		Priority:             item.Priority,
		Issuer:               item.Issuer,
		DisplayOnLogin:       item.DisplayOnLogin,
		AutoLogin:            item.AutoLogin,
		AutoCreateUser:       item.AutoCreateUser,
		DiscoveryURL:         item.DiscoveryURL,
		AuthorizationURL:     item.AuthorizationURL,
		TokenURL:             item.TokenURL,
		UserinfoURL:          item.UserinfoURL,
		JwksURL:              item.JwksURL,
		ClientID:             item.ClientID,
		RedirectURI:          item.RedirectURI,
		Scopes:               item.Scopes,
		ClaimMapping:         cloneMap(item.ClaimMapping),
		EntityID:             item.EntityID,
		MetadataURL:          item.MetadataURL,
		SSOURL:               item.SSOURL,
		SLOURL:               item.SLOURL,
		IDPCertificatePEM:    item.IDPCertificatePEM,
		CertificateNotBefore: item.CertificateNotBefore,
		CertificateNotAfter:  item.CertificateNotAfter,
		SPEntityID:           item.SPEntityID,
		ACSURL:               item.ACSURL,
		NameIDFormat:         item.NameIDFormat,
		LastSuccessAt:        item.LastSuccessAt,
		LastError:            item.LastError,
	}
}

func (s *GormStore) oauthClientFrom(ctx context.Context, row oauthClientRow) OAuthClient {
	item := OAuthClient{
		ID:                 row.ID,
		ClientID:           row.ClientID,
		Name:               row.Name,
		Description:        row.Description,
		ClientType:         row.ClientType,
		Status:             row.Status,
		RedirectURIs:       cloneStrings(row.RedirectURIs),
		GrantTypes:         cloneStrings(row.GrantTypes),
		Scopes:             cloneStrings(row.Scopes),
		PKCEPolicy:         row.PKCEPolicy,
		AccessTokenTTLSec:  row.AccessTokenTTLSec,
		RefreshTokenTTLSec: row.RefreshTokenTTLSec,
		IDTokenTTLSec:      row.IDTokenTTLSec,
		ClientSecretSet:    row.SecretID != nil && *row.SecretID != "",
		LastUsedAt:         row.LastUsedAt,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
	if row.SecretID != nil && *row.SecretID != "" {
		if hint, err := s.GetSecretHint(ctx, *row.SecretID); err == nil {
			item.ClientSecretHint = hint
		}
	}
	return item
}

func oauthClientToRow(item OAuthClient) oauthClientRow {
	return oauthClientRow{
		ID:                 item.ID,
		ClientID:           item.ClientID,
		Name:               item.Name,
		Description:        item.Description,
		ClientType:         item.ClientType,
		Status:             item.Status,
		RedirectURIs:       cloneStrings(item.RedirectURIs),
		GrantTypes:         cloneStrings(item.GrantTypes),
		Scopes:             cloneStrings(item.Scopes),
		PKCEPolicy:         item.PKCEPolicy,
		AccessTokenTTLSec:  item.AccessTokenTTLSec,
		RefreshTokenTTLSec: item.RefreshTokenTTLSec,
		IDTokenTTLSec:      item.IDTokenTTLSec,
		LastUsedAt:         item.LastUsedAt,
	}
}

func samlSPFrom(row samlSPRow) SamlSP {
	return SamlSP{
		ID:                   row.ID,
		Name:                 row.Name,
		EntityID:             row.EntityID,
		Status:               row.Status,
		ACSURLs:              cloneStrings(row.ACSURLs),
		SLOURL:               row.SLOURL,
		NameIDFormat:         row.NameIDFormat,
		AttributeMapping:     cloneMap(row.AttributeMapping),
		AllowIDPInitiated:    row.AllowIDPInitiated,
		CertificatePEM:       row.CertificatePEM,
		CertificateNotBefore: row.CertificateNotBefore,
		CertificateNotAfter:  row.CertificateNotAfter,
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
}

func samlSPToRow(item SamlSP) samlSPRow {
	return samlSPRow{
		ID:                   item.ID,
		Name:                 item.Name,
		EntityID:             item.EntityID,
		Status:               item.Status,
		ACSURLs:              cloneStrings(item.ACSURLs),
		SLOURL:               item.SLOURL,
		NameIDFormat:         item.NameIDFormat,
		AttributeMapping:     cloneMap(item.AttributeMapping),
		AllowIDPInitiated:    item.AllowIDPInitiated,
		CertificatePEM:       item.CertificatePEM,
		CertificateNotBefore: item.CertificateNotBefore,
		CertificateNotAfter:  item.CertificateNotAfter,
	}
}

func authEventsFrom(rows []authEventRow) []AuthEvent {
	out := make([]AuthEvent, 0, len(rows))
	for _, row := range rows {
		out = append(out, AuthEvent{
			ID:         row.ID,
			EventType:  row.EventType,
			Result:     row.Result,
			Protocol:   row.Protocol,
			ProviderID: row.ProviderID,
			ClientID:   row.ClientID,
			UserID:     row.UserID,
			Username:   row.Username,
			Message:    row.Message,
			RequestID:  row.RequestID,
			IP:         row.IP,
			CreateTime: row.CreateTime,
		})
	}
	return out
}

func pageQuery[Row any, Item any](query *gorm.DB, page, pageSize int, order string, convert func([]Row) []Item) ([]Item, int64, error) {
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []Row
	err := query.Session(&gorm.Session{}).Order(order).Limit(pageSize).Offset((page - 1) * pageSize).Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return convert(rows), total, nil
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneStrings(in []string) []string {
	if in == nil {
		return []string{}
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}

func newID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		panic(err)
	}
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16])
}