package weknora

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExternalTokenTransportInjectsHeader(t *testing.T) {
	var got string
	base := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		got = req.Header.Get(ExternalUserTokenHeader)
		return &http.Response{StatusCode: 200, Body: http.NoBody, Request: req}, nil
	})
	tr := &externalTokenTransport{base: base}
	ctx := WithExternalUserToken(context.Background(), "signed.jwt.token")
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "http://example.test/api", nil)
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if got != "signed.jwt.token" {
		t.Fatalf("header = %q", got)
	}
}

func TestExternalTokenTransportSkipsWhenAbsent(t *testing.T) {
	var got string
	base := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		got = req.Header.Get(ExternalUserTokenHeader)
		return &http.Response{StatusCode: 200, Body: http.NoBody, Request: req}, nil
	})
	tr := &externalTokenTransport{base: base}
	req := httptest.NewRequest(http.MethodGet, "http://example.test/api", nil)
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if got != "" {
		t.Fatalf("unexpected header %q", got)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
