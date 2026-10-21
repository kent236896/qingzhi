package modelcfg

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"

	"qingzhi/enterprise-api/internal/weknora"
)

// Remote is the WeKnora model surface. Tests substitute a fake.
type Remote interface {
	ListProviders(ctx context.Context, modelType string) ([]Provider, error)
	Resolve(ctx context.Context, provider, modelName, baseURL, modelType string) (map[string]any, error)
	ListModels(ctx context.Context) ([]Model, error)
	GetModel(ctx context.Context, id string) (Model, error)
	CreateModel(ctx context.Context, body map[string]any) (Model, error)
	UpdateModel(ctx context.Context, id string, body map[string]any) (Model, error)
	DeleteModel(ctx context.Context, id string) error
	CopyModel(ctx context.Context, id, displayName string) (Model, error)
	PutCredentials(ctx context.Context, id, apiKey, appSecret string) (Credentials, error)
	DeleteCredential(ctx context.Context, id, field string) error
	Test(ctx context.Context, kind string, body map[string]any) (TestResult, error)
	Debug(ctx context.Context, id, contentType string, body []byte) (map[string]any, error)
	OllamaStatus(ctx context.Context) (OllamaStatus, error)
	OllamaModels(ctx context.Context) ([]OllamaModel, error)
	OllamaDownload(ctx context.Context, name string) (OllamaTask, error)
	OllamaProgress(ctx context.Context, taskID string) (OllamaTask, error)
}

// HTTPRemote calls WeKnora with the shared client.
type HTTPRemote struct{ client *weknora.HTTPClient }

func NewHTTPRemote(client *weknora.HTTPClient) *HTTPRemote { return &HTTPRemote{client: client} }

func (r *HTTPRemote) ListProviders(ctx context.Context, modelType string) ([]Provider, error) {
	path := "/api/v1/models/providers"
	if modelType != "" {
		path += "?model_type=" + url.QueryEscape(modelType)
	}
	data, err := r.data(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var rows []map[string]any
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, err
	}
	items := make([]Provider, 0, len(rows))
	for _, row := range rows {
		items = append(items, projectProvider(row))
	}
	return items, nil
}

func (r *HTTPRemote) Resolve(ctx context.Context, provider, modelName, baseURL, modelType string) (map[string]any, error) {
	query := url.Values{}
	query.Set("provider", provider)
	query.Set("model", modelName)
	query.Set("base_url", baseURL)
	if modelType != "" {
		query.Set("model_type", modelType)
	}
	data, err := r.data(ctx, http.MethodGet, "/api/v1/models/catalog/resolve?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	cleaned, _ := scrubJSON(payload).(map[string]any)
	return cleaned, nil
}

func (r *HTTPRemote) ListModels(ctx context.Context) ([]Model, error) {
	data, err := r.data(ctx, http.MethodGet, "/api/v1/models", nil)
	if err != nil {
		return nil, err
	}
	var rows []json.RawMessage
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, err
	}
	items := make([]Model, 0, len(rows))
	for _, row := range rows {
		item, err := decodeModel(row)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *HTTPRemote) GetModel(ctx context.Context, id string) (Model, error) {
	data, err := r.data(ctx, http.MethodGet, "/api/v1/models/"+url.PathEscape(id), nil)
	if err != nil {
		return Model{}, err
	}
	return decodeModel(data)
}

func (r *HTTPRemote) CreateModel(ctx context.Context, body map[string]any) (Model, error) {
	data, err := r.data(ctx, http.MethodPost, "/api/v1/models", body)
	if err != nil {
		return Model{}, err
	}
	return decodeModel(data)
}

func (r *HTTPRemote) UpdateModel(ctx context.Context, id string, body map[string]any) (Model, error) {
	data, err := r.data(ctx, http.MethodPut, "/api/v1/models/"+url.PathEscape(id), body)
	if err != nil {
		return Model{}, err
	}
	return decodeModel(data)
}

func (r *HTTPRemote) DeleteModel(ctx context.Context, id string) error {
	_, err := r.data(ctx, http.MethodDelete, "/api/v1/models/"+url.PathEscape(id), nil)
	return err
}

func (r *HTTPRemote) CopyModel(ctx context.Context, id, displayName string) (Model, error) {
	data, err := r.data(ctx, http.MethodPost, "/api/v1/models/"+url.PathEscape(id)+"/copy", map[string]string{"display_name": displayName})
	if err != nil {
		return Model{}, err
	}
	return decodeModel(data)
}

func (r *HTTPRemote) PutCredentials(ctx context.Context, id, apiKey, appSecret string) (Credentials, error) {
	body := map[string]string{}
	if apiKey != "" {
		body["api_key"] = apiKey
	}
	if appSecret != "" {
		body["app_secret"] = appSecret
	}
	data, err := r.data(ctx, http.MethodPut, "/api/v1/models/"+url.PathEscape(id)+"/credentials", body)
	if err != nil {
		return Credentials{}, err
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return Credentials{}, err
	}
	if nested, ok := payload["credentials"]; ok {
		return decodeCredentials(nested), nil
	}
	return decodeCredentials(payload), nil
}

func (r *HTTPRemote) DeleteCredential(ctx context.Context, id, field string) error {
	_, err := r.data(ctx, http.MethodDelete, "/api/v1/models/"+url.PathEscape(id)+"/credentials/"+url.PathEscape(field), nil)
	return err
}

func (r *HTTPRemote) Test(ctx context.Context, kind string, body map[string]any) (TestResult, error) {
	path, ok := testPath(kind)
	if !ok {
		return TestResult{}, invalid("不支持的模型类型")
	}
	data, err := r.data(ctx, http.MethodPost, path, body)
	if err != nil {
		return TestResult{}, err
	}
	var payload struct {
		Available bool   `json:"available"`
		Message   string `json:"message"`
		Dimension *int   `json:"dimension"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return TestResult{}, err
	}
	return TestResult{Available: payload.Available, Message: payload.Message, Dimension: payload.Dimension}, nil
}

func (r *HTTPRemote) Debug(ctx context.Context, id, contentType string, body []byte) (map[string]any, error) {
	status, payload, err := r.client.ExchangeMultipart(ctx, "/api/v1/models/"+url.PathEscape(id)+"/debug", contentType, body)
	if err != nil {
		return nil, err
	}
	data, err := unwrap(status, payload)
	if err != nil {
		return nil, err
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		return nil, err
	}
	cleaned, _ := scrubJSON(parsed).(map[string]any)
	return cleaned, nil
}

func (r *HTTPRemote) OllamaStatus(ctx context.Context) (OllamaStatus, error) {
	data, err := r.data(ctx, http.MethodGet, "/api/v1/initialization/ollama/status", nil)
	if err != nil {
		return OllamaStatus{}, err
	}
	var payload struct {
		Available bool   `json:"available"`
		Version   string `json:"version"`
		BaseURL   string `json:"baseUrl"`
		Error     string `json:"error"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return OllamaStatus{}, err
	}
	return OllamaStatus{Available: payload.Available, Version: payload.Version, BaseURL: payload.BaseURL, Error: payload.Error}, nil
}

func (r *HTTPRemote) OllamaModels(ctx context.Context) ([]OllamaModel, error) {
	data, err := r.data(ctx, http.MethodGet, "/api/v1/initialization/ollama/models", nil)
	if err != nil {
		return nil, err
	}
	var rows []OllamaModel
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []OllamaModel{}
	}
	return rows, nil
}

func (r *HTTPRemote) OllamaDownload(ctx context.Context, name string) (OllamaTask, error) {
	data, err := r.data(ctx, http.MethodPost, "/api/v1/initialization/ollama/models/download", map[string]string{"model": name})
	if err != nil {
		return OllamaTask{}, err
	}
	return decodeTask(data)
}

func (r *HTTPRemote) OllamaProgress(ctx context.Context, taskID string) (OllamaTask, error) {
	data, err := r.data(ctx, http.MethodGet, "/api/v1/initialization/ollama/download/progress/"+url.PathEscape(taskID), nil)
	if err != nil {
		return OllamaTask{}, err
	}
	return decodeTask(data)
}

func (r *HTTPRemote) data(ctx context.Context, method, path string, body any) (json.RawMessage, error) {
	if r == nil || r.client == nil {
		return nil, notReady("请先在系统配置中填写 WeKnora 地址和 API Key")
	}
	status, payload, err := r.client.Exchange(ctx, method, path, body)
	if err != nil {
		return nil, err
	}
	return unwrap(status, payload)
}

func unwrap(status int, payload []byte) (json.RawMessage, error) {
	if status == http.StatusNoContent {
		return nil, nil
	}
	var envelope struct {
		Success bool            `json:"success"`
		Data    json.RawMessage `json:"data"`
		Message string          `json:"message"`
		Error   struct {
			Code    int             `json:"code"`
			Message string          `json:"message"`
			Details json.RawMessage `json:"details"`
		} `json:"error"`
	}
	_ = json.Unmarshal(payload, &envelope)
	if status < 200 || status >= 300 || (len(payload) > 0 && !envelope.Success && envelope.Error.Message != "") {
		message := envelope.Error.Message
		if message == "" {
			message = envelope.Message
		}
		return nil, &RemoteError{Status: status, Code: envelope.Error.Code, Message: message, Details: envelope.Error.Details}
	}
	if len(envelope.Data) == 0 {
		return json.RawMessage("null"), nil
	}
	return envelope.Data, nil
}

func testPath(kind string) (string, bool) {
	switch kind {
	case TypeChat, TypeVLLM:
		return "/api/v1/initialization/remote/check", true
	case TypeEmbedding:
		return "/api/v1/initialization/embedding/test", true
	case TypeRerank:
		return "/api/v1/initialization/rerank/check", true
	case TypeASR:
		return "/api/v1/initialization/asr/check", true
	default:
		return "", false
	}
}

func decodeTask(data json.RawMessage) (OllamaTask, error) {
	var payload struct {
		ID        string  `json:"id"`
		ModelName string  `json:"modelName"`
		Status    string  `json:"status"`
		Progress  float64 `json:"progress"`
		Message   string  `json:"message"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return OllamaTask{}, err
	}
	return OllamaTask{ID: payload.ID, ModelName: payload.ModelName, Status: payload.Status, Progress: payload.Progress, Message: payload.Message}, nil
}

func projectProvider(raw map[string]any) Provider {
	item := Provider{
		Value: stringField(raw, "value"), Label: localeLabel(raw, "label", "labels"),
		Description: localeLabel(raw, "description", "descriptions"), API: stringField(raw, "api"),
		RequiresAuth: boolField(raw, "requiresAuth"), Order: int(numberField(raw, "order")),
		DefaultURLs: map[string]string{}, ModelTypes: []string{}, ExtraFields: []ExtraField{},
		CredentialLabels: []map[string]any{}, Models: []CatalogModel{},
	}
	if auth, ok := raw["auth"].(string); ok {
		item.Auth = auth
	}
	if urls, ok := raw["defaultUrls"].(map[string]any); ok {
		for key, value := range urls {
			text, _ := value.(string)
			item.DefaultURLs[normalizeKey(key)] = text
		}
	}
	item.ModelTypes = stringList(raw["modelTypes"])
	item.ExtraFields = extraFields(raw["extraFields"])
	item.CredentialLabels = credentialLabels(raw["credentialLabels"])
	item.Models = catalogModels(raw["models"])
	if thinking, ok := raw["thinking"].(map[string]any); ok {
		item.Thinking = thinking
	}
	if icon, ok := raw["icon"].(string); ok {
		item.Icon = icon
	}
	return item
}

func extraFields(value any) []ExtraField {
	rows, _ := value.([]any)
	items := make([]ExtraField, 0, len(rows))
	for _, row := range rows {
		raw, ok := row.(map[string]any)
		if !ok {
			continue
		}
		field := ExtraField{
			Key: stringField(raw, "key"), Label: localeLabel(raw, "label", "labels"),
			Type: stringField(raw, "type"), Required: boolField(raw, "required"),
			Placeholder: localeLabel(raw, "placeholder", "placeholders"),
			ModelTypes:  normalizeList(firstList(raw, "model_types", "modelTypes")),
			Secret:      boolField(raw, "secret") || stringField(raw, "type") == "password",
		}
		field.Options = extraOptions(raw["options"])
		items = append(items, field)
	}
	return items
}

func extraOptions(value any) []ExtraOption {
	rows, _ := value.([]any)
	items := make([]ExtraOption, 0, len(rows))
	for _, row := range rows {
		switch item := row.(type) {
		case string:
			items = append(items, ExtraOption{Label: item, Value: item})
		case map[string]any:
			option := ExtraOption{Label: localeLabel(item, "label", "labels"), Value: stringField(item, "value")}
			if option.Label == "" {
				option.Label = option.Value
			}
			items = append(items, option)
		}
	}
	return items
}

func credentialLabels(value any) []map[string]any {
	rows, _ := value.([]any)
	items := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		raw, ok := row.(map[string]any)
		if !ok {
			continue
		}
		cleaned, _ := scrubJSON(raw).(map[string]any)
		if types := firstList(cleaned, "model_types", "modelTypes"); types != nil {
			cleaned["model_types"] = normalizeList(types)
			delete(cleaned, "modelTypes")
		}
		items = append(items, cleaned)
	}
	return items
}

func catalogModels(value any) []CatalogModel {
	rows, _ := value.([]any)
	items := make([]CatalogModel, 0, len(rows))
	for _, row := range rows {
		raw, ok := row.(map[string]any)
		if !ok {
			continue
		}
		item := CatalogModel{
			ID: stringField(raw, "id"), Name: stringField(raw, "name"), API: stringField(raw, "api"),
			Reasoning: boolField(raw, "reasoning"), ContextWindow: int(numberField(raw, "context_window")),
			MaxOutputTokens: int(numberField(raw, "max_output_tokens")), Dimension: int(numberField(raw, "dimension")),
			ThinkingLevels: stringList(raw["thinking_levels"]),
		}
		if semantic, ok := NormalizeType(stringField(raw, "type")); ok {
			item.Type = semantic
		}
		items = append(items, item)
	}
	return items
}

func localeLabel(raw map[string]any, key, plural string) string {
	if text := stringField(raw, key); text != "" {
		return text
	}
	labels, _ := raw[plural].(map[string]any)
	if text, _ := labels["zh-CN"].(string); text != "" {
		return text
	}
	if text, _ := labels["en-US"].(string); text != "" {
		return text
	}
	return ""
}

func stringList(value any) []string {
	rows, _ := value.([]any)
	items := make([]string, 0, len(rows))
	for _, row := range rows {
		text, _ := row.(string)
		if text != "" {
			items = append(items, normalizeKey(text))
		}
	}
	return items
}

func firstList(raw map[string]any, keys ...string) []any {
	for _, key := range keys {
		if rows, ok := raw[key].([]any); ok {
			return rows
		}
	}
	return nil
}

func normalizeList(rows []any) []string {
	items := make([]string, 0, len(rows))
	for _, row := range rows {
		text, _ := row.(string)
		if semantic, ok := NormalizeType(text); ok {
			items = append(items, semantic)
			continue
		}
		if text != "" {
			items = append(items, text)
		}
	}
	return items
}

func normalizeKey(value string) string {
	if semantic, ok := NormalizeType(value); ok {
		return semantic
	}
	return value
}

func numberField(raw map[string]any, key string) float64 {
	switch value := raw[key].(type) {
	case float64:
		return value
	case int:
		return float64(value)
	default:
		return 0
	}
}

func stripIcons(items []Provider) []Provider {
	out := make([]Provider, len(items))
	for i, item := range items {
		item.Icon = ""
		out[i] = item
	}
	return out
}
