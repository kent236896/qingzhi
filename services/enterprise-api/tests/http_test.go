package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"qingzhi/enterprise-api/api"
	"qingzhi/enterprise-api/internal/config"
	"qingzhi/enterprise-api/internal/platform/logger"
	"qingzhi/enterprise-api/internal/server"
)

func init() {
	gin.SetMode(gin.TestMode)
}

type stubPinger struct {
	err error
}

func (s stubPinger) Ping(context.Context) error { return s.err }

type stubOrNil interface {
	Ping(context.Context) error
}

func TestHealthLiveAndRequestID(t *testing.T) {
	srv, _ := newServer(t, nil, nil)
	recorder := perform(srv, http.MethodGet, "/api/v1/health/live", "req-live-01")

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("X-Request-ID") != "req-live-01" {
		t.Fatalf("request id header = %s", recorder.Header().Get("X-Request-ID"))
	}
	body := decode(t, recorder.Body.Bytes())
	if body["code"] != float64(0) || body["message"] != "ok" || body["request_id"] != "req-live-01" {
		t.Fatalf("body = %#v", body)
	}
	data := body["data"].(map[string]any)
	if data["status"] != "live" {
		t.Fatalf("data = %#v", data)
	}
}

func TestHealthReadySuccessAndFailure(t *testing.T) {
	ready, _ := newServer(t, stubPinger{}, stubPinger{})
	okResp := perform(ready, http.MethodGet, "/api/v1/health/ready", "req-ready-ok")
	if okResp.Code != http.StatusOK {
		t.Fatalf("ready status = %d", okResp.Code)
	}

	notReady, logs := newServer(t, stubPinger{err: errors.New("database unavailable")}, stubPinger{})
	failResp := perform(notReady, http.MethodGet, "/api/v1/health/ready", "req-ready-down")
	if failResp.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d", failResp.Code)
	}
	body := decode(t, failResp.Body.Bytes())
	if body["code"] != float64(50301) || body["message"] != "服务暂不可用" || body["request_id"] != "req-ready-down" {
		t.Fatalf("body = %#v", body)
	}
	data := body["data"].(map[string]any)
	checks := data["checks"].(map[string]any)
	if data["status"] != "not_ready" || checks["database"] != "down" || checks["redis"] != "up" {
		t.Fatalf("data = %#v", data)
	}
	if strings.Contains(failResp.Body.String(), "postgres://") || strings.Contains(logs.String(), "panic recovered") {
		t.Fatalf("unexpected diagnostic leak: %s", failResp.Body.String())
	}
}

func TestPanicRecovery(t *testing.T) {
	srv, logs := newServer(t, stubPinger{}, stubPinger{})
	srv.Engine().GET("/api/v1/panic-test", func(*gin.Context) {
		panic("boom secret stack")
	})

	recorder := perform(srv, http.MethodGet, "/api/v1/panic-test", "req-panic-1")
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", recorder.Code)
	}
	bodyText := recorder.Body.String()
	if strings.Contains(bodyText, "goroutine") || strings.Contains(bodyText, "boom") || strings.Contains(bodyText, "secret") {
		t.Fatalf("response leaked panic: %s", bodyText)
	}
	body := decode(t, recorder.Body.Bytes())
	if body["code"] != float64(50000) || body["message"] != "服务器内部错误" || body["request_id"] != "req-panic-1" {
		t.Fatalf("body = %#v", body)
	}
	if strings.Count(logs.String(), "panic recovered") != 1 {
		t.Fatalf("panic log count = %d\n%s", strings.Count(logs.String(), "panic recovered"), logs.String())
	}
}

func TestNotFoundAndCORS(t *testing.T) {
	srv, _ := newServer(t, nil, nil)
	missing := perform(srv, http.MethodGet, "/api/v1/missing", "req-missing")
	body := decode(t, missing.Body.Bytes())
	if missing.Code != http.StatusNotFound || body["code"] != float64(40400) {
		t.Fatalf("status %d body %#v", missing.Code, body)
	}

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/health/live", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "GET")
	recorder := httptest.NewRecorder()
	srv.Engine().ServeHTTP(recorder, req)
	if recorder.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Fatalf("cors origin = %q status = %d", recorder.Header().Get("Access-Control-Allow-Origin"), recorder.Code)
	}
}

func TestOpenAPIMatchesRoutes(t *testing.T) {
	srv, _ := newServer(t, stubPinger{}, stubPinger{})
	var spec struct {
		Paths map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(api.OpenAPI, &spec); err != nil {
		t.Fatal(err)
	}

	expected := map[string]struct{}{}
	for path, methods := range spec.Paths {
		for method := range methods {
			expected[strings.ToUpper(method)+" "+path] = struct{}{}
		}
	}
	actual := map[string]struct{}{}
	for _, route := range srv.Routes() {
		actual[route.Method+" "+route.Path] = struct{}{}
	}
	for route := range expected {
		if _, ok := actual[route]; !ok {
			t.Errorf("documented route is not registered: %s", route)
		}
	}
	for route := range actual {
		if _, ok := expected[route]; !ok {
			t.Errorf("registered route is missing from OpenAPI: %s", route)
		}
	}

	recorder := perform(srv, http.MethodGet, "/api/v1/openapi.json", "req-openapi")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte(`"/api/v1/health/live"`)) {
		t.Fatal("openapi response does not document the live route")
	}
}

func newServer(t *testing.T, database, redis stubOrNil) (*server.Server, *bytes.Buffer) {
	t.Helper()
	logs := &bytes.Buffer{}
	srv := server.New(server.Dependencies{
		Config: config.Config{
			AppName: "enterprise-api",
			AppEnv:  config.EnvTest,
			HTTP:    config.HTTPConfig{Host: "127.0.0.1", Port: 8080},
			CORS:    config.CORSConfig{AllowedOrigins: []string{"http://localhost:5173"}},
		},
		Logger:   logger.NewWriter(logs, "debug", "enterprise-api"),
		Database: database,
		Redis:    redis,
	})
	return srv, logs
}

func perform(srv *server.Server, method, path, requestID string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("X-Request-ID", requestID)
	recorder := httptest.NewRecorder()
	srv.Engine().ServeHTTP(recorder, req)
	return recorder
}

func decode(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("json: %v body=%s", err, raw)
	}
	return body
}
