package weknora

import (
	"context"
	"net/http"
)

const ExternalUserTokenHeader = "X-External-User-Token"

type externalUserTokenKey struct{}

// WithExternalUserToken stores a short-lived signed_token JWT on the context.
// The HTTP transport injects it as X-External-User-Token on WeKnora requests.
func WithExternalUserToken(ctx context.Context, token string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, externalUserTokenKey{}, token)
}

// ExternalUserTokenFrom reads a token previously stored with WithExternalUserToken.
func ExternalUserTokenFrom(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	token, _ := ctx.Value(externalUserTokenKey{}).(string)
	return token
}

type externalTokenTransport struct {
	base http.RoundTripper
}

func (t *externalTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	token := ExternalUserTokenFrom(req.Context())
	if token == "" {
		return base.RoundTrip(req)
	}
	clone := req.Clone(req.Context())
	clone.Header.Set(ExternalUserTokenHeader, token)
	return base.RoundTrip(clone)
}

func wrapExternalTokenTransport(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	if _, ok := base.(*externalTokenTransport); ok {
		return base
	}
	return &externalTokenTransport{base: base}
}
