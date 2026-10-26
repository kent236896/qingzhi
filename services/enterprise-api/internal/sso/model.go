package sso

import "time"

const (
	ProtocolOIDC   = "oidc"
	ProtocolOAuth2 = "oauth2"
	ProtocolSAML   = "saml"

	StatusEnabled  = "enabled"
	StatusDisabled = "disabled"
	StatusRevoked  = "revoked"

	ClientPublic       = "public"
	ClientConfidential = "confidential"

	PKCERequired  = "required"
	PKCEOptional  = "optional"
	PKCEForbidden = "forbidden"

	ResultSuccess = "success"
	ResultFailure = "failure"
)

// Settings is the global SSO policy.
type Settings struct {
	LocalLoginEnabled bool      `json:"local_login_enabled"`
	Issuer            string    `json:"issuer"`
	DefaultTenantID   string    `json:"default_tenant_id"`
	FrontendBaseURL   string    `json:"frontend_base_url"`
	Updater           string    `json:"updater"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// LoginOption is a public login-page entry (no secrets).
type LoginOption struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Protocol       string `json:"protocol"`
	DisplayOnLogin bool   `json:"display_on_login"`
	AutoLogin      bool   `json:"auto_login"`
}

// AuthRequest is a pending external authorization request.
type AuthRequest struct {
	ID           string
	State        string
	Nonce        string
	CodeVerifier string
	ProviderID   string
	Protocol     string
	RedirectPath string
	TenantID     string
	ExpiresAt    time.Time
	ConsumedAt   *time.Time
	CreatedAt    time.Time
}

// LoginTicket is a one-time code the frontend exchanges for a local JWT.
type LoginTicket struct {
	ID         string
	Ticket     string
	UserID     string
	TenantID   string
	ProviderID string
	ExpiresAt  time.Time
	ConsumedAt *time.Time
	CreatedAt  time.Time
}

// Overview is the authentication dashboard payload. All counts come from the database.
type Overview struct {
	LocalLoginEnabled      bool           `json:"local_login_enabled"`
	Issuer                 string         `json:"issuer"`
	OIDCProviders          ProviderCount  `json:"oidc_providers"`
	OAuth2Providers        ProviderCount  `json:"oauth2_providers"`
	SAMLProviders          ProviderCount  `json:"saml_providers"`
	OAuthClients           ClientCount    `json:"oauth_clients"`
	SAMLSps                ClientCount    `json:"saml_sps"`
	IdentityLinks          int64          `json:"identity_links"`
	RecentLoginSuccess     int64          `json:"recent_login_success"`
	RecentLoginFailure     int64          `json:"recent_login_failure"`
	RecentProtocolFailures []AuthEvent    `json:"recent_protocol_failures"`
	CertificateWarnings    []CertWarning  `json:"certificate_warnings"`
	CollectedAt            time.Time      `json:"collected_at"`
}

// ProviderCount summarizes external IdP rows.
type ProviderCount struct {
	Total   int64 `json:"total"`
	Enabled int64 `json:"enabled"`
}

// ClientCount summarizes registered relying parties / SPs.
type ClientCount struct {
	Total   int64 `json:"total"`
	Enabled int64 `json:"enabled"`
}

// CertWarning is a certificate approaching or past expiry.
type CertWarning struct {
	Kind      string     `json:"kind"`
	RefID     string     `json:"ref_id"`
	Name      string     `json:"name"`
	NotAfter  *time.Time `json:"not_after,omitempty"`
	DaysLeft  *int       `json:"days_left,omitempty"`
	Expired   bool       `json:"expired"`
}

// Provider is an external identity provider configuration. Secrets are never included.
type Provider struct {
	ID                  string         `json:"id"`
	Name                string         `json:"name"`
	Protocol            string         `json:"protocol"`
	Status              string         `json:"status"`
	Priority            int            `json:"priority"`
	Issuer              string         `json:"issuer"`
	DisplayOnLogin      bool           `json:"display_on_login"`
	AutoLogin           bool           `json:"auto_login"`
	AutoCreateUser      bool           `json:"auto_create_user"`
	DiscoveryURL        string         `json:"discovery_url,omitempty"`
	AuthorizationURL    string         `json:"authorization_url,omitempty"`
	TokenURL            string         `json:"token_url,omitempty"`
	UserinfoURL         string         `json:"userinfo_url,omitempty"`
	JwksURL             string         `json:"jwks_url,omitempty"`
	ClientID            string         `json:"client_id,omitempty"`
	ClientSecretSet     bool           `json:"client_secret_set"`
	ClientSecretHint    string         `json:"client_secret_hint,omitempty"`
	RedirectURI         string         `json:"redirect_uri,omitempty"`
	Scopes              string         `json:"scopes,omitempty"`
	ClaimMapping        map[string]any `json:"claim_mapping,omitempty"`
	EntityID            string         `json:"entity_id,omitempty"`
	MetadataURL         string         `json:"metadata_url,omitempty"`
	SSOURL              string         `json:"sso_url,omitempty"`
	SLOURL              string         `json:"slo_url,omitempty"`
	IDPCertificatePEM   string         `json:"idp_certificate_pem,omitempty"`
	CertificateNotBefore *time.Time    `json:"certificate_not_before,omitempty"`
	CertificateNotAfter  *time.Time    `json:"certificate_not_after,omitempty"`
	SPEntityID          string         `json:"sp_entity_id,omitempty"`
	ACSURL              string         `json:"acs_url,omitempty"`
	NameIDFormat        string         `json:"name_id_format,omitempty"`
	LastSuccessAt       *time.Time     `json:"last_success_at,omitempty"`
	LastError           string         `json:"last_error,omitempty"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
}

// ProviderInput creates or updates an external provider. ClientSecret is write-only.
type ProviderInput struct {
	Name             string         `json:"name"`
	Protocol         string         `json:"protocol"`
	Status           string         `json:"status"`
	Priority         int            `json:"priority"`
	Issuer           string         `json:"issuer"`
	DisplayOnLogin   bool           `json:"display_on_login"`
	AutoLogin        bool           `json:"auto_login"`
	AutoCreateUser   bool           `json:"auto_create_user"`
	DiscoveryURL     string         `json:"discovery_url"`
	AuthorizationURL string         `json:"authorization_url"`
	TokenURL         string         `json:"token_url"`
	UserinfoURL      string         `json:"userinfo_url"`
	JwksURL          string         `json:"jwks_url"`
	ClientID         string         `json:"client_id"`
	ClientSecret     string         `json:"client_secret"`
	RedirectURI      string         `json:"redirect_uri"`
	Scopes           string         `json:"scopes"`
	ClaimMapping     map[string]any `json:"claim_mapping"`
	EntityID         string         `json:"entity_id"`
	MetadataURL      string         `json:"metadata_url"`
	SSOURL           string         `json:"sso_url"`
	SLOURL           string         `json:"slo_url"`
	IDPCertificatePEM string        `json:"idp_certificate_pem"`
	SPEntityID       string         `json:"sp_entity_id"`
	ACSURL           string         `json:"acs_url"`
	NameIDFormat     string         `json:"name_id_format"`
}

// OAuthClient is a registered OAuth2/OIDC client. Full secrets are never returned.
type OAuthClient struct {
	ID                 string     `json:"id"`
	ClientID           string     `json:"client_id"`
	Name               string     `json:"name"`
	Description        string     `json:"description"`
	ClientType         string     `json:"client_type"`
	Status             string     `json:"status"`
	RedirectURIs       []string   `json:"redirect_uris"`
	GrantTypes         []string   `json:"grant_types"`
	Scopes             []string   `json:"scopes"`
	PKCEPolicy         string     `json:"pkce_policy"`
	AccessTokenTTLSec  int        `json:"access_token_ttl_sec"`
	RefreshTokenTTLSec int        `json:"refresh_token_ttl_sec"`
	IDTokenTTLSec      int        `json:"id_token_ttl_sec"`
	ClientSecretSet    bool       `json:"client_secret_set"`
	ClientSecretHint   string     `json:"client_secret_hint,omitempty"`
	LastUsedAt         *time.Time `json:"last_used_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// OAuthClientInput creates or updates a client. ClientSecret is write-only.
type OAuthClientInput struct {
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	ClientType         string   `json:"client_type"`
	Status             string   `json:"status"`
	RedirectURIs       []string `json:"redirect_uris"`
	GrantTypes         []string `json:"grant_types"`
	Scopes             []string `json:"scopes"`
	PKCEPolicy         string   `json:"pkce_policy"`
	AccessTokenTTLSec  int      `json:"access_token_ttl_sec"`
	RefreshTokenTTLSec int      `json:"refresh_token_ttl_sec"`
	IDTokenTTLSec      int      `json:"id_token_ttl_sec"`
	ClientSecret       string   `json:"client_secret"`
}

// OAuthClientCreated is returned only on create/rotate with a one-time plaintext secret.
type OAuthClientCreated struct {
	OAuthClient
	ClientSecret string `json:"client_secret,omitempty"`
}

// SamlSP is a SAML service provider registered against this platform as IdP.
type SamlSP struct {
	ID                   string         `json:"id"`
	Name                 string         `json:"name"`
	EntityID             string         `json:"entity_id"`
	Status               string         `json:"status"`
	ACSURLs              []string       `json:"acs_urls"`
	SLOURL               string         `json:"slo_url"`
	NameIDFormat         string         `json:"name_id_format"`
	AttributeMapping     map[string]any `json:"attribute_mapping"`
	AllowIDPInitiated    bool           `json:"allow_idp_initiated"`
	CertificatePEM       string         `json:"certificate_pem,omitempty"`
	CertificateNotBefore *time.Time     `json:"certificate_not_before,omitempty"`
	CertificateNotAfter  *time.Time     `json:"certificate_not_after,omitempty"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
}

// SamlSPInput creates or updates a SAML SP registration.
type SamlSPInput struct {
	Name              string         `json:"name"`
	EntityID          string         `json:"entity_id"`
	Status            string         `json:"status"`
	ACSURLs           []string       `json:"acs_urls"`
	SLOURL            string         `json:"slo_url"`
	NameIDFormat      string         `json:"name_id_format"`
	AttributeMapping  map[string]any `json:"attribute_mapping"`
	AllowIDPInitiated bool           `json:"allow_idp_initiated"`
	CertificatePEM    string         `json:"certificate_pem"`
}

// IdentityLink maps an external subject to a local user.
type IdentityLink struct {
	ID              string    `json:"id"`
	ProviderID      string    `json:"provider_id"`
	ProviderName    string    `json:"provider_name"`
	Protocol        string    `json:"protocol"`
	ExternalSubject string    `json:"external_subject"`
	UserID          string    `json:"user_id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// AuthEvent is a redacted authentication or federation audit row.
type AuthEvent struct {
	ID         string    `json:"id"`
	EventType  string    `json:"event_type"`
	Result     string    `json:"result"`
	Protocol   string    `json:"protocol"`
	ProviderID string    `json:"provider_id"`
	ClientID   string    `json:"client_id"`
	UserID     string    `json:"user_id"`
	Username   string    `json:"username"`
	Message    string    `json:"message"`
	RequestID  string    `json:"request_id"`
	IP         string    `json:"ip"`
	CreateTime time.Time `json:"create_time"`
}

// ListQuery is shared pagination for SSO list APIs.
type ListQuery struct {
	Page     int
	PageSize int
	Keyword  string
	Protocol string
	Status   string
	Result   string
}

// Page is a paginated list.
type Page[T any] struct {
	Items    []T   `json:"items"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}
