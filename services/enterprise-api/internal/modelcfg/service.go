package modelcfg

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime/multipart"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"qingzhi/enterprise-api/internal/config"
	"qingzhi/enterprise-api/internal/knowledge"
)

const (
	providerCacheTTL = 10 * time.Minute
	sensitiveLimit   = 5
	maxDebugInput    = 64 * 1024
)

// Service proxies WeKnora models and stores scene default bindings.
type Service struct {
	store  Store
	remote Remote
	cfg    config.WeKnoraConfig
	log    *slog.Logger
	now    func() time.Time

	mu    sync.Mutex
	cache map[string]cacheEntry
	hits  map[string][]time.Time
}

type cacheEntry struct {
	at    time.Time
	items []Provider
}

func NewService(store Store, remote Remote, cfg config.WeKnoraConfig, log *slog.Logger) *Service {
	return &Service{
		store: store, remote: remote, cfg: cfg, log: log, now: time.Now,
		cache: map[string]cacheEntry{}, hits: map[string][]time.Time{},
	}
}

func (s *Service) ready() error {
	if s == nil || s.remote == nil || !s.cfg.Enabled() {
		return notReady("模型服务未配置")
	}
	return nil
}

// Allow reports whether this user may run another test or debug call.
func (s *Service) Allow(userID, action string) bool {
	if userID == "" {
		userID = "unknown"
	}
	key := action + ":" + userID
	now := s.now()
	cutoff := now.Add(-time.Minute)
	s.mu.Lock()
	defer s.mu.Unlock()
	recent := s.hits[key][:0]
	for _, at := range s.hits[key] {
		if at.After(cutoff) {
			recent = append(recent, at)
		}
	}
	if len(recent) >= sensitiveLimit {
		s.hits[key] = recent
		return false
	}
	s.hits[key] = append(recent, now)
	return true
}

func (s *Service) Providers(ctx context.Context, modelType string, includeIcons bool) ([]Provider, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	if modelType != "" {
		if _, ok := ToWeKnoraType(modelType); !ok {
			return nil, invalid("不支持的模型类型")
		}
	}
	items, err := s.cachedProviders(ctx, modelType)
	if err != nil {
		return nil, s.translate(err)
	}
	if !includeIcons {
		items = stripIcons(items)
	}
	return items, nil
}

func (s *Service) cachedProviders(ctx context.Context, modelType string) ([]Provider, error) {
	now := s.now()
	s.mu.Lock()
	if entry, ok := s.cache[modelType]; ok && now.Sub(entry.at) < providerCacheTTL {
		items := append([]Provider(nil), entry.items...)
		s.mu.Unlock()
		return items, nil
	}
	s.mu.Unlock()
	items, err := s.remote.ListProviders(ctx, modelType)
	if err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.cache[modelType] = cacheEntry{at: now, items: items}
	s.mu.Unlock()
	return append([]Provider(nil), items...), nil
}

func (s *Service) Resolve(ctx context.Context, provider, modelName, baseURL, modelType string) (map[string]any, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	if modelType != "" {
		if _, ok := ToWeKnoraType(modelType); !ok {
			return nil, invalid("不支持的模型类型")
		}
	}
	if baseURL != "" && !validHTTPURL(baseURL) {
		return nil, invalid("接口地址必须以 http 或 https 开头")
	}
	data, err := s.remote.Resolve(ctx, provider, modelName, baseURL, modelType)
	return data, s.translate(err)
}

func (s *Service) List(ctx context.Context, modelType, keyword string) ([]Model, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	if modelType != "" {
		if _, ok := ToWeKnoraType(modelType); !ok {
			return nil, invalid("不支持的模型类型")
		}
	}
	items, err := s.remote.ListModels(ctx)
	if err != nil {
		return nil, s.translate(err)
	}
	roles := s.rolesByModel(ctx)
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	filtered := make([]Model, 0)
	for _, item := range items {
		if modelType != "" && item.Type != modelType {
			continue
		}
		if keyword != "" && !strings.Contains(strings.ToLower(item.Name), keyword) && !strings.Contains(strings.ToLower(item.DisplayName), keyword) {
			continue
		}
		item.BoundRoles = roles[item.ID]
		if item.BoundRoles == nil {
			item.BoundRoles = []string{}
		}
		filtered = append(filtered, item)
	}
	return filtered, nil
}

func (s *Service) Get(ctx context.Context, id string) (Model, error) {
	if err := s.ready(); err != nil {
		return Model{}, err
	}
	item, err := s.remote.GetModel(ctx, id)
	if err != nil {
		return Model{}, s.translate(err)
	}
	item.BoundRoles = s.rolesByModel(ctx)[item.ID]
	if item.BoundRoles == nil {
		item.BoundRoles = []string{}
	}
	return item, nil
}

type SaveInput struct {
	Name        string         `json:"name"`
	DisplayName string         `json:"display_name"`
	Description string         `json:"description"`
	Type        string         `json:"type"`
	Source      string         `json:"source"`
	Parameters  map[string]any `json:"parameters"`
	APIKey      string         `json:"api_key"`
	AppSecret   string         `json:"app_secret"`
}

func (s *Service) Create(ctx context.Context, input SaveInput) (Model, error) {
	if err := s.ready(); err != nil {
		return Model{}, err
	}
	body, err := s.createBody(input)
	if err != nil {
		return Model{}, err
	}
	item, err := s.remote.CreateModel(ctx, body)
	if err != nil {
		return Model{}, s.translate(err)
	}
	item.BoundRoles = []string{}
	s.logAction("model created", item.ID)
	return item, nil
}

func (s *Service) Update(ctx context.Context, id string, input SaveInput) (Model, error) {
	if err := s.ready(); err != nil {
		return Model{}, err
	}
	current, err := s.remote.GetModel(ctx, id)
	if err != nil {
		return Model{}, s.translate(err)
	}
	if current.IsBuiltin {
		return Model{}, forbidden("内置模型不可修改")
	}
	input.Name = strings.TrimSpace(input.Name)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.Source = strings.TrimSpace(input.Source)
	if err := s.validateSave(input, false); err != nil {
		return Model{}, err
	}
	body := map[string]any{
		"name": input.Name, "display_name": input.DisplayName, "description": input.Description,
		"source": input.Source, "parameters": input.Parameters,
	}
	if input.Type != "" {
		remoteType, _ := ToWeKnoraType(input.Type)
		body["type"] = remoteType
	}
	item, err := s.remote.UpdateModel(ctx, id, body)
	if err != nil {
		return Model{}, s.translate(err)
	}
	if input.APIKey != "" || input.AppSecret != "" {
		creds, credErr := s.remote.PutCredentials(ctx, id, input.APIKey, input.AppSecret)
		if credErr != nil {
			return Model{}, s.translate(credErr)
		}
		item.Credentials = creds
	}
	item.BoundRoles = s.rolesByModel(ctx)[item.ID]
	if item.BoundRoles == nil {
		item.BoundRoles = []string{}
	}
	s.logAction("model updated", id)
	return item, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.ready(); err != nil {
		return err
	}
	current, err := s.remote.GetModel(ctx, id)
	if err != nil {
		return s.translate(err)
	}
	if current.IsBuiltin {
		return forbidden("内置模型不可修改")
	}
	roles := s.rolesByModel(ctx)[id]
	if len(roles) > 0 {
		labels := make([]string, 0, len(roles))
		for _, role := range roles {
			labels = append(labels, RoleLabel(role))
		}
		return conflictData("该模型已被默认场景占用（"+strings.Join(labels, "、")+"），请先在默认模型中更换", Usage{LocalRoles: roles, KnowledgeBases: []UsageResource{}, Agents: []UsageResource{}})
	}
	if err := s.remote.DeleteModel(ctx, id); err != nil {
		return s.translate(err)
	}
	s.logAction("model deleted", id)
	return nil
}

func (s *Service) Copy(ctx context.Context, id, displayName string) (Model, error) {
	if err := s.ready(); err != nil {
		return Model{}, err
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" || utf8.RuneCountInString(displayName) > 255 {
		return Model{}, invalid("请填写新的显示名")
	}
	current, err := s.remote.GetModel(ctx, id)
	if err != nil {
		return Model{}, s.translate(err)
	}
	if current.IsBuiltin {
		return Model{}, forbidden("内置模型不可修改")
	}
	item, err := s.remote.CopyModel(ctx, id, displayName)
	if err != nil {
		return Model{}, s.translate(err)
	}
	item.BoundRoles = []string{}
	s.logAction("model copied", id)
	return item, nil
}

func (s *Service) PutCredentials(ctx context.Context, id, apiKey, appSecret string) (Credentials, error) {
	if err := s.ready(); err != nil {
		return Credentials{}, err
	}
	current, err := s.remote.GetModel(ctx, id)
	if err != nil {
		return Credentials{}, s.translate(err)
	}
	if current.IsBuiltin {
		return Credentials{}, forbidden("内置模型不可修改")
	}
	creds, err := s.remote.PutCredentials(ctx, id, apiKey, appSecret)
	if err != nil {
		return Credentials{}, s.translate(err)
	}
	s.logAction("model credentials updated", id)
	return creds, nil
}

func (s *Service) DeleteCredential(ctx context.Context, id, field string) error {
	if err := s.ready(); err != nil {
		return err
	}
	if field != "api_key" && field != "app_secret" {
		return invalid("只能清除 api_key 或 app_secret")
	}
	current, err := s.remote.GetModel(ctx, id)
	if err != nil {
		return s.translate(err)
	}
	if current.IsBuiltin {
		return forbidden("内置模型不可修改")
	}
	if err := s.remote.DeleteCredential(ctx, id, field); err != nil {
		return s.translate(err)
	}
	s.logAction("model credential cleared", id)
	return nil
}

type TestInput struct {
	Type          string            `json:"type"`
	ModelName     string            `json:"model_name"`
	Source        string            `json:"source"`
	BaseURL       string            `json:"base_url"`
	APIKey        string            `json:"api_key"`
	AppSecret     string            `json:"app_secret"`
	Provider      string            `json:"provider"`
	InterfaceType string            `json:"interface_type"`
	Dimension     int               `json:"dimension"`
	CustomHeaders map[string]string `json:"custom_headers"`
	ExtraConfig   map[string]string `json:"extra_config"`
	Spec          map[string]any    `json:"spec"`
	ModelID       string            `json:"model_id"`
}

func (s *Service) Test(ctx context.Context, input TestInput) (TestResult, error) {
	if err := s.ready(); err != nil {
		return TestResult{}, err
	}
	if _, ok := ToWeKnoraType(input.Type); !ok {
		return TestResult{}, invalid("不支持的模型类型")
	}
	if strings.TrimSpace(input.ModelName) == "" {
		return TestResult{}, invalid("请填写模型名")
	}
	if input.BaseURL != "" && !validHTTPURL(input.BaseURL) {
		return TestResult{}, invalid("接口地址必须以 http 或 https 开头")
	}
	body := map[string]any{"modelName": input.ModelName}
	if input.Source != "" {
		body["source"] = input.Source
	}
	if input.BaseURL != "" {
		body["baseUrl"] = input.BaseURL
	}
	if input.APIKey != "" {
		body["apiKey"] = input.APIKey
	}
	if input.AppSecret != "" {
		body["appSecret"] = input.AppSecret
	}
	if input.Provider != "" {
		body["provider"] = input.Provider
	}
	if input.InterfaceType != "" {
		body["interfaceType"] = input.InterfaceType
	}
	if input.Dimension > 0 {
		body["dimension"] = input.Dimension
	}
	if len(input.CustomHeaders) > 0 {
		body["customHeaders"] = input.CustomHeaders
	}
	if len(input.ExtraConfig) > 0 {
		body["extraConfig"] = input.ExtraConfig
	}
	if len(input.Spec) > 0 {
		body["spec"] = input.Spec
	}
	if input.ModelID != "" {
		body["modelId"] = input.ModelID
	}
	started := s.now()
	result, err := s.remote.Test(ctx, input.Type, body)
	if err != nil {
		return TestResult{}, s.translate(err)
	}
	result.ElapsedMS = s.now().Sub(started).Milliseconds()
	s.logAction("model tested", input.ModelID)
	return result, nil
}

func (s *Service) Debug(ctx context.Context, id, contentType string, body []byte) (map[string]any, error) {
	if err := s.ready(); err != nil {
		return nil, err
	}
	if len(body) == 0 {
		return nil, invalid("请填写调试内容")
	}
	result, err := s.remote.Debug(ctx, id, contentType, body)
	if err != nil {
		return nil, s.translate(err)
	}
	s.logAction("model debugged", id)
	return result, nil
}

// BuildDebugForm validates and repacks a multipart debug request without logging the input.
func BuildDebugForm(input string, optionsJSON, documents string, fileName string, file []byte) (string, []byte, error) {
	if len(input) > maxDebugInput {
		return "", nil, invalid("调试输入不能超过 64KB")
	}
	if optionsJSON != "" {
		var options map[string]any
		if err := json.Unmarshal([]byte(optionsJSON), &options); err != nil {
			return "", nil, invalid("调试选项不是合法 JSON")
		}
		if value, ok := options["temperature"].(float64); ok && (value < 0 || value > 2) {
			return "", nil, invalid("温度必须在 0 到 2 之间")
		}
		if value, ok := options["max_tokens"].(float64); ok && (value < 1 || value > 8192) {
			return "", nil, invalid("最大输出必须在 1 到 8192 之间")
		}
	}
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	if err := writer.WriteField("input", input); err != nil {
		return "", nil, err
	}
	if optionsJSON != "" {
		if err := writer.WriteField("options", optionsJSON); err != nil {
			return "", nil, err
		}
	}
	if documents != "" {
		if err := writer.WriteField("documents", documents); err != nil {
			return "", nil, err
		}
	}
	if fileName != "" && file != nil {
		part, err := writer.CreateFormFile("file", fileName)
		if err != nil {
			return "", nil, err
		}
		if _, err := part.Write(file); err != nil {
			return "", nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return "", nil, err
	}
	return writer.FormDataContentType(), buf.Bytes(), nil
}

func (s *Service) OllamaStatus(ctx context.Context) (OllamaStatus, error) {
	if err := s.ollamaReady(); err != nil {
		return OllamaStatus{}, err
	}
	item, err := s.remote.OllamaStatus(ctx)
	return item, s.translate(err)
}

func (s *Service) OllamaModels(ctx context.Context) ([]OllamaModel, error) {
	if err := s.ollamaReady(); err != nil {
		return nil, err
	}
	items, err := s.remote.OllamaModels(ctx)
	return items, s.translate(err)
}

func (s *Service) OllamaDownload(ctx context.Context, name string) (OllamaTask, error) {
	if err := s.ollamaReady(); err != nil {
		return OllamaTask{}, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return OllamaTask{}, invalid("请填写要下载的模型名")
	}
	item, err := s.remote.OllamaDownload(ctx, name)
	return item, s.translate(err)
}

func (s *Service) OllamaProgress(ctx context.Context, taskID string) (OllamaTask, error) {
	if err := s.ollamaReady(); err != nil {
		return OllamaTask{}, err
	}
	item, err := s.remote.OllamaProgress(ctx, taskID)
	return item, s.translate(err)
}

func (s *Service) ollamaReady() error {
	if err := s.ready(); err != nil {
		return err
	}
	if !s.cfg.OllamaEnabled {
		return notFound("本地模型未启用")
	}
	return nil
}

func (s *Service) ListBindings(ctx context.Context) ([]Binding, error) {
	if s.store == nil {
		return nil, notReady("模型服务未配置")
	}
	rows, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}
	byRole := map[string]Binding{}
	for _, row := range rows {
		byRole[row.Role] = row
	}
	names := map[string]Model{}
	if s.remote != nil && s.cfg.Enabled() {
		if items, listErr := s.remote.ListModels(ctx); listErr == nil {
			for _, item := range items {
				names[item.ID] = item
			}
		}
	}
	out := make([]Binding, 0, len(RoleOrder))
	for _, role := range RoleOrder {
		item := byRole[role]
		item.Role = role
		if model, ok := names[item.ModelID]; ok {
			item.ModelName = model.Name
			item.DisplayName = model.DisplayName
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *Service) SaveBinding(ctx context.Context, role, modelID, updatedBy string) (Binding, error) {
	if s.store == nil {
		return Binding{}, notReady("模型服务未配置")
	}
	expected, ok := RoleModelType(role)
	if !ok {
		return Binding{}, invalid("未知的默认场景")
	}
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return Binding{}, invalid("请选择模型")
	}
	if err := s.ready(); err != nil {
		return Binding{}, err
	}
	model, err := s.remote.GetModel(ctx, modelID)
	if err != nil {
		return Binding{}, s.translate(err)
	}
	if model.Type != expected {
		return Binding{}, invalid("所选模型类型与场景不匹配")
	}
	item := Binding{Role: role, ModelID: modelID, ModelName: model.Name, DisplayName: model.DisplayName, UpdatedBy: updatedBy, UpdatedAt: s.now().UTC()}
	if err := s.store.Upsert(ctx, item); err != nil {
		return Binding{}, err
	}
	if role == RoleEmbedding {
		item.Warning = "仅影响之后新建的知识库；已有文件的知识库不能更换向量模型"
	}
	s.logAction("model binding updated", modelID)
	return item, nil
}

// SeedFallback inserts env model ids only when a role has no row yet.
func (s *Service) SeedFallback(ctx context.Context) error {
	if s.store == nil {
		return nil
	}
	seeds := map[string]string{
		RoleEmbedding: s.cfg.EmbeddingModelID, RoleSummary: s.cfg.SummaryModelID, RoleWikiSynthesis: s.cfg.WikiSynthesisModelID,
	}
	for role, modelID := range seeds {
		if strings.TrimSpace(modelID) == "" {
			continue
		}
		_, err := s.store.Find(ctx, role)
		if err == nil {
			continue
		}
		if !errors.Is(err, ErrNotFound) {
			return err
		}
		if upsertErr := s.store.Upsert(ctx, Binding{Role: role, ModelID: modelID, UpdatedBy: "system", UpdatedAt: s.now().UTC()}); upsertErr != nil {
			return upsertErr
		}
	}
	return nil
}

// KnowledgeModels resolves the models used when creating a knowledge base.
func (s *Service) KnowledgeModels(ctx context.Context) (knowledge.KnowledgeModels, error) {
	rows, err := s.ListBindings(ctx)
	if err != nil {
		return knowledge.KnowledgeModels{}, err
	}
	out := knowledge.KnowledgeModels{}
	for _, row := range rows {
		bound := knowledge.BoundModel{ID: row.ModelID, Name: row.ModelName, DisplayName: row.DisplayName}
		switch row.Role {
		case RoleEmbedding:
			out.Embedding = bound
		case RoleSummary:
			out.Summary = bound
		case RoleWikiSynthesis:
			out.WikiSynthesis = bound
		}
	}
	return out, nil
}

func (s *Service) createBody(input SaveInput) (map[string]any, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	input.Source = strings.TrimSpace(input.Source)
	if err := s.validateSave(input, true); err != nil {
		return nil, err
	}
	remoteType, _ := ToWeKnoraType(input.Type)
	body := map[string]any{
		"name": input.Name, "display_name": input.DisplayName, "description": input.Description,
		"type": remoteType, "source": input.Source, "parameters": input.Parameters,
	}
	if input.APIKey != "" {
		body["api_key"] = input.APIKey
	}
	if input.AppSecret != "" {
		body["app_secret"] = input.AppSecret
	}
	return body, nil
}

func (s *Service) validateSave(input SaveInput, creating bool) error {
	input.Name = strings.TrimSpace(input.Name)
	input.Source = strings.TrimSpace(input.Source)
	if input.Name == "" || input.Source == "" || (creating && input.Type == "") {
		return invalid("名称、类型和来源不能为空")
	}
	if input.Type != "" {
		if _, ok := ToWeKnoraType(input.Type); !ok {
			return invalid("不支持的模型类型")
		}
	}
	if input.Parameters == nil {
		return invalid("请填写模型参数")
	}
	if baseURL, _ := input.Parameters["base_url"].(string); baseURL != "" && !validHTTPURL(baseURL) {
		return invalid("接口地址必须以 http 或 https 开头")
	}
	if input.Type == TypeEmbedding && dimensionOf(input.Parameters) <= 0 {
		return invalid("向量模型必须填写维度")
	}
	return nil
}

func (s *Service) rolesByModel(ctx context.Context) map[string][]string {
	out := map[string][]string{}
	if s.store == nil {
		return out
	}
	rows, err := s.store.List(ctx)
	if err != nil {
		return out
	}
	for _, row := range rows {
		if row.ModelID == "" {
			continue
		}
		out[row.ModelID] = append(out[row.ModelID], row.Role)
	}
	return out
}

func (s *Service) translate(err error) error {
	if err == nil {
		return nil
	}
	var remote *RemoteError
	if errors.As(err, &remote) {
		switch {
		case remote.Status == 403:
			return forbidden("后端凭证权限不足")
		case remote.Status == 404:
			message := remote.Message
			if message == "" {
				message = "模型不存在"
			}
			return notFound(message)
		case remote.Code == 2300:
			return conflictData(firstText(remote.Message, "模型仍被引用，请先更换这些位置的模型"), parseUsage(remote.Details))
		case remote.Status == 408 || remote.Status == 504:
			return timeout("模型服务响应超时")
		case remote.Status == 400:
			return invalid(firstText(remote.Message, "模型服务拒绝了这个请求"))
		default:
			return invalid(firstText(remote.Message, "模型服务暂时无法完成这个操作"))
		}
	}
	if errors.Is(err, context.DeadlineExceeded) || isTimeout(err) {
		return timeout("模型服务响应超时")
	}
	return err
}

func parseUsage(raw []byte) Usage {
	usage := Usage{KnowledgeBases: []UsageResource{}, Agents: []UsageResource{}, LongTermMemory: MemoryUsage{Bindings: []string{}}}
	if len(raw) == 0 {
		return usage
	}
	_ = json.Unmarshal(raw, &usage)
	if usage.KnowledgeBases == nil {
		usage.KnowledgeBases = []UsageResource{}
	}
	if usage.Agents == nil {
		usage.Agents = []UsageResource{}
	}
	if usage.LongTermMemory.Bindings == nil {
		usage.LongTermMemory.Bindings = []string{}
	}
	return usage
}

func (s *Service) logAction(action, modelID string) {
	if s.log == nil {
		return
	}
	s.log.Info(action, slog.String("component", "model"), slog.String("model_id", modelID))
}

func validHTTPURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	return err == nil && (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

func firstText(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

// ReadDebugParts extracts the debug form. The caller must not log the returned input.
func ReadDebugParts(form *multipart.Form) (string, string, string, string, []byte, error) {
	input := firstValue(form, "input")
	options := firstValue(form, "options")
	documents := firstValue(form, "documents")
	var fileName string
	var file []byte
	if headers := form.File["file"]; len(headers) > 0 {
		opened, err := headers[0].Open()
		if err != nil {
			return "", "", "", "", nil, invalid("无法读取调试文件")
		}
		defer opened.Close()
		file, err = io.ReadAll(io.LimitReader(opened, 32<<20+1))
		if err != nil {
			return "", "", "", "", nil, invalid("无法读取调试文件")
		}
		if len(file) > 32<<20 {
			return "", "", "", "", nil, invalid("调试文件不能超过 32MB")
		}
		fileName = headers[0].Filename
	}
	return input, options, documents, fileName, file, nil
}

func firstValue(form *multipart.Form, key string) string {
	if form == nil || len(form.Value[key]) == 0 {
		return ""
	}
	return form.Value[key][0]
}
