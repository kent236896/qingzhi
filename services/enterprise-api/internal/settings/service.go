package settings

import (
	"context"
	"log/slog"
	"net/url"
	"strings"
	"time"
)

// View is the connection shown to administrators. It never contains the API key.
type View struct {
	BaseURL          string    `json:"base_url"`
	APIKeyConfigured bool      `json:"api_key_configured"`
	Reachable        bool      `json:"reachable"`
	Version          string    `json:"version,omitempty"`
	UpdatedAt        time.Time `json:"updated_at"`
	Updater          string    `json:"updater"`
}

// Update is a page save. An empty APIKey keeps the stored key.
type Update struct {
	BaseURL string
	APIKey  string
	Updater string
}

// Hooks apply a saved connection to the running process and check it.
type Hooks struct {
	Apply func(baseURL, apiKey string)
	Probe func(ctx context.Context) (string, error)
}

// Service stores the WeKnora address and API key.
type Service struct {
	store Store
	hooks Hooks
	log   *slog.Logger
	now   func() time.Time
}

func NewService(store Store, log *slog.Logger) *Service {
	return &Service{store: store, log: log, now: time.Now}
}

func (s *Service) SetHooks(hooks Hooks) {
	if s != nil {
		s.hooks = hooks
	}
}

// Ensure copies the environment values into an empty row once.
// After a row has an address or a key, the stored values win.
func (s *Service) Ensure(ctx context.Context, baseURL, apiKey string) (Record, error) {
	item, err := s.store.Get(ctx)
	if err != nil {
		return Record{}, err
	}
	baseURL = strings.TrimSpace(baseURL)
	apiKey = strings.TrimSpace(apiKey)
	if item.BaseURL == "" && item.APIKey == "" && baseURL != "" && apiKey != "" {
		item = Record{BaseURL: strings.TrimRight(baseURL, "/"), APIKey: apiKey, UpdatedBy: "环境变量", UpdatedAt: s.now()}
		if err := s.store.Save(ctx, item); err != nil {
			return Record{}, err
		}
	}
	return item, nil
}

func (s *Service) Get(ctx context.Context) (View, error) {
	item, err := s.store.Get(ctx)
	if err != nil {
		return View{}, err
	}
	return s.present(ctx, item), nil
}

func (s *Service) Update(ctx context.Context, input Update) (View, error) {
	baseURL, err := normalizeBaseURL(input.BaseURL)
	if err != nil {
		return View{}, err
	}
	current, err := s.store.Get(ctx)
	if err != nil {
		return View{}, err
	}
	apiKey := strings.TrimSpace(input.APIKey)
	if apiKey == "" {
		apiKey = current.APIKey
	}
	if apiKey == "" {
		return View{}, invalid("请填写 API Key")
	}
	if strings.ContainsAny(apiKey, "\r\n") || len(apiKey) > 512 {
		return View{}, invalid("API Key 不合法")
	}
	item := Record{BaseURL: baseURL, APIKey: apiKey, UpdatedBy: strings.TrimSpace(input.Updater), UpdatedAt: s.now()}
	if err := s.store.Save(ctx, item); err != nil {
		return View{}, err
	}
	return s.present(ctx, item), nil
}

func (s *Service) present(ctx context.Context, item Record) View {
	if s.hooks.Apply != nil && item.BaseURL != "" && item.APIKey != "" {
		s.hooks.Apply(item.BaseURL, item.APIKey)
	}
	view := View{
		BaseURL:          item.BaseURL,
		APIKeyConfigured: item.APIKey != "",
		UpdatedAt:        item.UpdatedAt,
		Updater:          item.UpdatedBy,
	}
	if !view.APIKeyConfigured || item.BaseURL == "" || s.hooks.Probe == nil {
		return view
	}
	version, err := s.hooks.Probe(ctx)
	if err != nil {
		if s.log != nil {
			s.log.Warn("weknora settings probe failed", slog.String("component", "settings"), slog.String("error", safeProbeError(err)))
		}
		return view
	}
	view.Reachable = true
	view.Version = version
	return view
}

func normalizeBaseURL(raw string) (string, error) {
	value := strings.TrimRight(strings.TrimSpace(raw), "/")
	if value == "" {
		return "", invalid("请填写 WeKnora 地址")
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", invalid("WeKnora 地址必须是 http 或 https")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return "", invalid("WeKnora 地址不要带路径，也不要带 /api/v1")
	}
	return value, nil
}

func safeProbeError(err error) string {
	text := err.Error()
	lower := strings.ToLower(text)
	if strings.Contains(lower, "api_key") || strings.Contains(lower, "sk-") {
		return "weknora probe failed"
	}
	return text
}
