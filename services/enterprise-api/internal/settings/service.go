package settings

import (
	"context"
	"log/slog"
	"net/url"
	"strings"
	"time"
)

// View is the connection shown to administrators. It never contains secrets.
type View struct {
	BaseURL               string    `json:"base_url"`
	APIKeyConfigured      bool      `json:"api_key_configured"`
	WeKnoraTenantID       string    `json:"weknora_tenant_id"`
	AdminEmail            string    `json:"admin_email"`
	AdminPasswordConfigured bool    `json:"admin_password_configured"`
	Reachable             bool      `json:"reachable"`
	Version               string    `json:"version,omitempty"`
	UpdatedAt             time.Time `json:"updated_at"`
	Updater               string    `json:"updater"`
}

// Update is a page save. Empty secret fields keep stored values.
type Update struct {
	BaseURL         string
	APIKey          string
	WeKnoraTenantID string
	AdminEmail      string
	AdminPassword   string
	Updater         string
}

// Hooks apply a saved connection to the running process and check it.
type Hooks struct {
	Apply           func(baseURL, apiKey, adminEmail, adminPassword string)
	Probe           func(ctx context.Context) (string, error)
	EnsurePrincipal func(ctx context.Context, baseURL, apiKey, tenantID, adminEmail, adminPassword string) (string, error)
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

const DefaultWeKnoraTenantID = "10000"

// Ensure copies the environment values into an empty row once.
// After a row has an address or a key, the stored values win.
// tenantID is seeded only when the stored weknora_tenant_id is empty (default 10000).
func (s *Service) Ensure(ctx context.Context, baseURL, apiKey, tenantID string) (Record, error) {
	item, err := s.store.Get(ctx)
	if err != nil {
		return Record{}, err
	}
	baseURL = strings.TrimSpace(baseURL)
	apiKey = strings.TrimSpace(apiKey)
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		tenantID = DefaultWeKnoraTenantID
	}
	changed := false
	if item.BaseURL == "" && item.APIKey == "" && baseURL != "" && apiKey != "" {
		item.BaseURL = strings.TrimRight(baseURL, "/")
		item.APIKey = apiKey
		item.UpdatedBy = "环境变量"
		item.UpdatedAt = s.now()
		changed = true
	}
	if strings.TrimSpace(item.WeKnoraTenantID) == "" {
		item.WeKnoraTenantID = tenantID
		if item.UpdatedBy == "" {
			item.UpdatedBy = "环境变量"
		}
		item.UpdatedAt = s.now()
		changed = true
	}
	if changed {
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
	tenantID := strings.TrimSpace(input.WeKnoraTenantID)
	if tenantID == "" {
		tenantID = strings.TrimSpace(current.WeKnoraTenantID)
	}
	if tenantID == "" {
		tenantID = DefaultWeKnoraTenantID
	}
	if len(tenantID) > 64 {
		return View{}, invalid("WeKnora 空间 ID 过长")
	}
	adminEmail := strings.TrimSpace(input.AdminEmail)
	if adminEmail == "" {
		adminEmail = strings.TrimSpace(current.AdminEmail)
	}
	if len(adminEmail) > 255 {
		return View{}, invalid("WeKnora 管理员邮箱过长")
	}
	adminPassword := strings.TrimSpace(input.AdminPassword)
	if adminPassword == "" {
		adminPassword = current.AdminPassword
	}
	if strings.ContainsAny(adminPassword, "\r\n") || len(adminPassword) > 512 {
		return View{}, invalid("WeKnora 管理员密码不合法")
	}
	item := Record{
		BaseURL: baseURL, APIKey: apiKey, WeKnoraTenantID: tenantID,
		AdminEmail: adminEmail, AdminPassword: adminPassword,
		UpdatedBy: strings.TrimSpace(input.Updater), UpdatedAt: s.now(),
	}
	if err := s.store.Save(ctx, item); err != nil {
		return View{}, err
	}
	return s.present(ctx, item), nil
}

// WeKnoraTenantID returns the configured space id for signed_token tenant_id claims.
// Empty storage falls back to DefaultWeKnoraTenantID (10000).
func (s *Service) WeKnoraTenantID(ctx context.Context) (string, error) {
	item, err := s.store.Get(ctx)
	if err != nil {
		return "", err
	}
	id := strings.TrimSpace(item.WeKnoraTenantID)
	if id == "" {
		return DefaultWeKnoraTenantID, nil
	}
	return id, nil
}

// EnsureSignedTokenPrincipal logs into WeKnora with stored admin credentials and sets
// api-principal-config.mode=signed_token (HMAC comes from the EnsurePrincipal hook).
func (s *Service) EnsureSignedTokenPrincipal(ctx context.Context) (string, error) {
	if s.hooks.EnsurePrincipal == nil {
		return "", invalid("未配置 WeKnora 主体同步能力")
	}
	item, err := s.store.Get(ctx)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(item.BaseURL) == "" || strings.TrimSpace(item.APIKey) == "" {
		return "", invalid("请先保存 WeKnora 地址和 API Key")
	}
	email := strings.TrimSpace(item.AdminEmail)
	password := strings.TrimSpace(item.AdminPassword)
	if email == "" || password == "" {
		return "", invalid("请先填写并保存 WeKnora 管理员邮箱与密码")
	}
	tenantID := strings.TrimSpace(item.WeKnoraTenantID)
	if tenantID == "" {
		tenantID = DefaultWeKnoraTenantID
	}
	mode, err := s.hooks.EnsurePrincipal(ctx, item.BaseURL, item.APIKey, tenantID, email, password)
	if err != nil {
		return "", err
	}
	return mode, nil
}

func (s *Service) present(ctx context.Context, item Record) View {
	if s.hooks.Apply != nil && item.BaseURL != "" && item.APIKey != "" {
		s.hooks.Apply(item.BaseURL, item.APIKey, item.AdminEmail, item.AdminPassword)
	}
	tenantID := strings.TrimSpace(item.WeKnoraTenantID)
	if tenantID == "" {
		tenantID = DefaultWeKnoraTenantID
	}
	view := View{
		BaseURL:                 item.BaseURL,
		APIKeyConfigured:        item.APIKey != "",
		WeKnoraTenantID:         tenantID,
		AdminEmail:              strings.TrimSpace(item.AdminEmail),
		AdminPasswordConfigured: item.AdminPassword != "",
		UpdatedAt:               item.UpdatedAt,
		Updater:                 item.UpdatedBy,
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
