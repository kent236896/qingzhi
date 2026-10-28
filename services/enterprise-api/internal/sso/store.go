package sso

import (
	"context"
	"time"
)

// Store persists SSO configuration and audit events.
type Store interface {
	Settings(ctx context.Context) (Settings, error)
	SaveSettings(ctx context.Context, item Settings) error

	CountProviders(ctx context.Context, protocol string) (int64, int64, error)
	CountIdentityLinks(ctx context.Context) (int64, error)
	CountAuthEventsSince(ctx context.Context, result, eventTypePrefix string, since time.Time) (int64, error)
	ListRecentAuthEvents(ctx context.Context, result string, limit int) ([]AuthEvent, error)
	ListCertificateWarnings(ctx context.Context, withinDays int) ([]CertWarning, error)

	ListProviders(ctx context.Context, query ListQuery) ([]Provider, int64, error)
	GetProvider(ctx context.Context, id string) (Provider, error)
	CreateProvider(ctx context.Context, item Provider, secretID *string) (Provider, error)
	UpdateProvider(ctx context.Context, item Provider, secretID *string) (Provider, error)
	SetProviderStatus(ctx context.Context, id, status string) error

	ListIdentityLinks(ctx context.Context, query ListQuery) ([]IdentityLink, int64, error)
	FindIdentityLink(ctx context.Context, providerID, externalSubject string) (IdentityLink, error)
	CreateIdentityLink(ctx context.Context, item IdentityLink) (IdentityLink, error)
	DeleteIdentityLink(ctx context.Context, id string) error

	ListLoginProviders(ctx context.Context) ([]LoginOption, error)
	GetProviderSecretID(ctx context.Context, providerID string) (string, error)
	MarkProviderAuthResult(ctx context.Context, providerID string, success bool, message string) error

	CreateAuthRequest(ctx context.Context, item AuthRequest) (AuthRequest, error)
	ConsumeAuthRequest(ctx context.Context, state string) (AuthRequest, error)

	CreateLoginTicket(ctx context.Context, item LoginTicket) (LoginTicket, error)
	ConsumeLoginTicket(ctx context.Context, ticket string) (LoginTicket, error)

	InsertAuthEvent(ctx context.Context, item AuthEvent) error
	ListAuthEvents(ctx context.Context, query ListQuery) ([]AuthEvent, int64, error)

	CreateSecret(ctx context.Context, kind, hint string, ciphertext []byte) (id string, err error)
	GetSecretHint(ctx context.Context, id string) (hint string, err error)
	GetSecretCiphertext(ctx context.Context, id string) ([]byte, error)
	RevokeSecret(ctx context.Context, id string) error
}
