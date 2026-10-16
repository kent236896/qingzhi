package monitor

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"qingzhi/enterprise-api/internal/config"
)

type stubPinger struct{ err error }

func (s stubPinger) Ping(context.Context) error { return s.err }

func testConfig() config.MonitorConfig {
	return config.MonitorConfig{
		Window:         time.Minute,
		ErrorCount:     2,
		TimeoutCount:   1,
		P95:            200 * time.Millisecond,
		CPUPercent:     90,
		MemoryPercent:  90,
		DiskPercent:    90,
		RequestTimeout: time.Second,
		MaxBodyBytes:   32,
		RateLimitRPS:   1,
		RateLimitBurst: 1,
	}
}

func TestBuildAlerts(t *testing.T) {
	cfg := testConfig()
	drafts := buildAlerts(cfg, ServiceState{API: "up", Database: "down", Redis: "up"}, Traffic{}, Resources{}, false)
	if len(drafts) != 1 || drafts[0].Target != "database" {
		t.Fatalf("drafts = %+v", drafts)
	}
	drafts = buildAlerts(cfg, ServiceState{API: "up", Database: "up", Redis: "up"}, Traffic{Requests: 4, ServerErrors: 2, Timeouts: 1, P95MS: 500}, Resources{Available: true, CPUPercent: 95}, true)
	seen := map[string]bool{}
	for _, draft := range drafts {
		seen[draft.Kind+"/"+draft.Target] = true
	}
	for _, key := range []string{"traffic/errors", "traffic/latency", "traffic/timeout", "task/collector", "capacity/cpu"} {
		if !seen[key] {
			t.Fatalf("missing %s in %+v", key, drafts)
		}
	}
}

func TestRecorderAggregatesRealSamples(t *testing.T) {
	recorder := &Recorder{}
	now := time.Now()
	recorder.Add(Sample{Time: now, Status: 200, Duration: 10 * time.Millisecond})
	recorder.Add(Sample{Time: now, Status: 500, Duration: 40 * time.Millisecond})
	recorder.Add(Sample{Time: now, Status: 429, Duration: time.Millisecond})
	recorder.Add(Sample{Time: now, Status: 200, Duration: 5 * time.Millisecond, TimedOut: true})
	traffic, stats, recent := recorder.Snapshot(time.Minute, now.Add(time.Second))
	if traffic.Requests != 4 || traffic.ServerErrors != 1 || traffic.Rejected != 1 || traffic.Timeouts != 1 {
		t.Fatalf("traffic = %+v", traffic)
	}
	if traffic.P95MS < 10 {
		t.Fatalf("p95 = %d", traffic.P95MS)
	}
	if len(stats) != 3 || len(recent) != 3 {
		t.Fatalf("stats %d recent %d", len(stats), len(recent))
	}
}

func TestAlertSyncResolvesClearedCondition(t *testing.T) {
	store := &MemoryStore{}
	firing, err := store.Sync(context.Background(), []Draft{{Kind: "service", Target: "redis", Level: "critical", Title: "Redis 不可用"}})
	if err != nil || len(firing) != 1 || firing[0].ID == "" {
		t.Fatalf("firing = %+v err %v", firing, err)
	}
	cleared, err := store.Sync(context.Background(), nil)
	if err != nil || len(cleared) != 0 {
		t.Fatalf("cleared = %+v err %v", cleared, err)
	}
	page, err := store.List(context.Background(), 1, 20, "resolved")
	if err != nil || page.Total != 1 || page.Items[0].Status != "resolved" {
		t.Fatalf("page = %+v err %v", page, err)
	}
}

func TestOverviewUsesProbeResults(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := New(testConfig(), nil, stubPinger{}, stubPinger{err: errors.New("redis down")}, &MemoryStore{})
	svc.readHost = func(context.Context) (Resources, error) {
		return Resources{Available: true, CPUPercent: 3, MemoryPercent: 10, DiskPercent: 20, DiskMount: `C:\`}, nil
	}
	overview := svc.Overview(context.Background())
	if overview.Service.Database != "up" || overview.Service.Redis != "down" || overview.Service.API != "up" {
		t.Fatalf("service = %+v", overview.Service)
	}
	if len(overview.Alerts) != 1 || overview.Alerts[0].Target != "redis" {
		t.Fatalf("alerts = %+v", overview.Alerts)
	}
	if overview.Resources.DiskMount != `C:\` {
		t.Fatalf("resources = %+v", overview.Resources)
	}
}

func TestProtectLimitsAndRecords(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := New(testConfig(), nil, stubPinger{}, stubPinger{}, &MemoryStore{})
	svc.readHost = func(context.Context) (Resources, error) { return Resources{Available: true}, nil }
	engine := gin.New()
	engine.Use(svc.Protect())
	engine.POST("/api/v1/items", func(c *gin.Context) {
		c.Status(http.StatusCreated)
	})
	first := httptest.NewRequest(http.MethodPost, "/api/v1/items", strings.NewReader(`{}`))
	first.Header.Set("Content-Type", "application/json")
	ok := httptest.NewRecorder()
	engine.ServeHTTP(ok, first)
	if ok.Code != http.StatusCreated {
		t.Fatalf("first status = %d", ok.Code)
	}
	second := httptest.NewRequest(http.MethodPost, "/api/v1/items", strings.NewReader(`{}`))
	limited := httptest.NewRecorder()
	engine.ServeHTTP(limited, second)
	if limited.Code != http.StatusTooManyRequests {
		t.Fatalf("second status = %d body %s", limited.Code, limited.Body.String())
	}
	large := httptest.NewRequest(http.MethodPost, "/api/v1/items", strings.NewReader(`{"name":"too-large-body"}`))
	large.ContentLength = 64
	large.RemoteAddr = "10.0.0.8:1000"
	svc.limiter.now = func() time.Time { return time.Now().Add(2 * time.Second) }
	rejected := httptest.NewRecorder()
	engine.ServeHTTP(rejected, large)
	if rejected.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("large status = %d", rejected.Code)
	}
	traffic, _, _ := svc.recorder.Snapshot(time.Minute, time.Now().Add(3*time.Second))
	if traffic.Requests < 3 || traffic.Rejected < 2 {
		t.Fatalf("traffic = %+v", traffic)
	}
}

func TestProtectTimesOut(t *testing.T) {
	gin.SetMode(gin.TestMode)
	cfg := testConfig()
	cfg.RateLimitRPS = 0
	cfg.RequestTimeout = 20 * time.Millisecond
	svc := New(cfg, nil, nil, nil, nil)
	engine := gin.New()
	engine.Use(svc.Protect())
	engine.GET("/api/v1/slow", func(c *gin.Context) {
		<-c.Request.Context().Done()
	})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/slow", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusGatewayTimeout {
		t.Fatalf("status = %d body %s", recorder.Code, recorder.Body.String())
	}
	traffic, _, _ := svc.recorder.Snapshot(time.Minute, time.Now().Add(time.Second))
	if traffic.Timeouts != 1 {
		t.Fatalf("traffic = %+v", traffic)
	}
}

func TestRedactPath(t *testing.T) {
	got := redactPath("/api/v1/users/a1000000-0000-4000-8000-000000000001")
	if got != "/api/v1/users/{id}" {
		t.Fatal(got)
	}
}
