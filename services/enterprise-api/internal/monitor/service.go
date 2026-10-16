package monitor

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"qingzhi/enterprise-api/internal/config"
	"qingzhi/enterprise-api/internal/response"
)

// Service collects health, traffic, resources, and alerts.
type Service struct {
	cfg       config.MonitorConfig
	log       *slog.Logger
	database  Pinger
	redis     Pinger
	store     AlertStore
	readHost  func(context.Context) (Resources, error)
	recorder  *Recorder
	limiter   *Limiter
	evalMu    sync.Mutex
	sampleMu  sync.Mutex
	cached    Resources
	cachedErr error
	cachedAt  time.Time
	hasCache  bool
}

// New builds a monitor. A nil store still serves live overview data.
func New(cfg config.MonitorConfig, log *slog.Logger, database, redis Pinger, store AlertStore) *Service {
	if cfg.Window <= 0 {
		cfg.Window = 5 * time.Minute
	}
	if cfg.P95 <= 0 {
		cfg.P95 = 2 * time.Second
	}
	return &Service{
		cfg:      cfg,
		log:      log,
		database: database,
		redis:    redis,
		store:    store,
		readHost: ReadResources,
		recorder: &Recorder{},
		limiter:  NewLimiter(cfg.RateLimitRPS, cfg.RateLimitBurst),
	}
}

// Run samples alerts until the context is canceled.
func (s *Service) Run(ctx context.Context) {
	if s == nil {
		return
	}
	s.Overview(ctx)
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.Overview(ctx)
		}
	}
}

// Overview pings dependencies, reads the request window, and syncs alerts.
func (s *Service) Overview(ctx context.Context) Overview {
	s.evalMu.Lock()
	defer s.evalMu.Unlock()
	now := time.Now()
	state := ServiceState{API: "up", Database: ping(ctx, s.log, "database", s.database), Redis: ping(ctx, s.log, "redis", s.redis)}
	resources, resourceErr := s.resources(ctx)
	resourceFailed := resourceErr != nil
	traffic, errors, recent := s.recorder.Snapshot(s.cfg.Window, now)
	if errors == nil {
		errors = []ErrorStat{}
	}
	if recent == nil {
		recent = []RecentError{}
	}
	drafts := buildAlerts(s.cfg, state, traffic, resources, resourceFailed)
	alerts := draftsAsAlerts(drafts, now)
	if s.store != nil {
		stored, err := s.store.Sync(ctx, drafts)
		if err != nil {
			if s.log != nil {
				s.log.Warn("monitor alert sync failed", slog.String("component", "app"), slog.String("error", err.Error()))
			}
		} else {
			alerts = stored
		}
	}
	if alerts == nil {
		alerts = []Alert{}
	}
	return Overview{
		CollectedAt:   now,
		WindowSeconds: int(s.cfg.Window / time.Second),
		Service:       state,
		Traffic:       traffic,
		Errors:        errors,
		RecentErrors:  recent,
		Resources:     resources,
		Protection: Protection{
			RateLimitRPS:     s.cfg.RateLimitRPS,
			RateLimitBurst:   s.cfg.RateLimitBurst,
			MaxBodyBytes:     s.cfg.MaxBodyBytes,
			RequestTimeoutMS: int(s.cfg.RequestTimeout / time.Millisecond),
		},
		Alerts: alerts,
	}
}

// ListAlerts returns stored alert history.
func (s *Service) ListAlerts(ctx context.Context, page, pageSize int, status string) (Page[Alert], error) {
	page, pageSize, err := normalizePage(page, pageSize)
	if err != nil {
		return Page[Alert]{}, err
	}
	status, err = normalizeStatus(status)
	if err != nil {
		return Page[Alert]{}, err
	}
	if s == nil || s.store == nil {
		return Page[Alert]{Items: []Alert{}, Page: page, PageSize: pageSize}, nil
	}
	return s.store.List(ctx, page, pageSize, status)
}

// Protect applies the body limit, per-IP rate limit, and request timeout, then records the call.
func (s *Service) Protect() gin.HandlerFunc {
	if s == nil {
		return func(c *gin.Context) { c.Next() }
	}
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		if !exempt(path) && !s.limiter.Allow(c.ClientIP()) {
			response.Fail(c, http.StatusTooManyRequests, response.CodeTooManyRequests, "请求过于频繁", nil)
			s.record(c, start, false)
			return
		}
		if s.cfg.MaxBodyBytes > 0 && c.Request.ContentLength > s.cfg.MaxBodyBytes {
			response.Fail(c, http.StatusRequestEntityTooLarge, response.CodePayloadTooLarge, "请求体过大", nil)
			s.record(c, start, false)
			return
		}
		if s.cfg.MaxBodyBytes > 0 && c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, s.cfg.MaxBodyBytes)
		}
		timedOut := false
		if s.cfg.RequestTimeout > 0 {
			ctx, cancel := context.WithTimeout(c.Request.Context(), s.cfg.RequestTimeout)
			defer cancel()
			c.Request = c.Request.WithContext(ctx)
			c.Next()
			timedOut = errors.Is(ctx.Err(), context.DeadlineExceeded)
			if timedOut && !c.Writer.Written() {
				response.Fail(c, http.StatusGatewayTimeout, response.CodeTimeout, "请求超时", nil)
			}
		} else {
			c.Next()
		}
		s.record(c, start, timedOut)
	}
}

func (s *Service) record(c *gin.Context, start time.Time, timedOut bool) {
	path := c.FullPath()
	if path == "" && c.Request != nil && c.Request.URL != nil {
		path = c.Request.URL.Path
	}
	if strings.HasPrefix(path, "/api/v1/health/") || strings.HasPrefix(path, "/api/v1/monitor/") {
		return
	}
	status := c.Writer.Status()
	if !c.Writer.Written() {
		status = http.StatusOK
	}
	method := ""
	if c.Request != nil {
		method = c.Request.Method
	}
	s.recorder.Add(Sample{
		Time:      time.Now(),
		Method:    method,
		Path:      redactPath(path),
		Status:    status,
		Duration:  time.Since(start),
		TimedOut:  timedOut,
		RequestID: response.RequestID(c),
	})
}

func (s *Service) resources(ctx context.Context) (Resources, error) {
	s.sampleMu.Lock()
	defer s.sampleMu.Unlock()
	if s.hasCache && time.Since(s.cachedAt) < 5*time.Second {
		return s.cached, s.cachedErr
	}
	sample, err := s.readHost(ctx)
	s.cached = sample
	s.cachedErr = err
	s.cachedAt = time.Now()
	s.hasCache = true
	if err != nil && s.log != nil {
		s.log.Warn("resource sample failed", slog.String("component", "app"), slog.String("error", err.Error()))
	}
	return sample, err
}

func ping(ctx context.Context, log *slog.Logger, name string, pinger Pinger) string {
	if pinger == nil {
		return "down"
	}
	checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := pinger.Ping(checkCtx); err != nil {
		if log != nil {
			log.Debug("dependency check failed",
				slog.String("component", "app"),
				slog.String("dependency", name),
				slog.String("error", err.Error()),
			)
		}
		return "down"
	}
	return "up"
}

func exempt(path string) bool {
	return strings.HasPrefix(path, "/api/v1/health/") || path == "/api/v1/openapi.json"
}
