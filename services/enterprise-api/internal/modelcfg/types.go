package modelcfg

import (
	"encoding/json"
	"strings"
)

const (
	TypeChat      = "chat"
	TypeEmbedding = "embedding"
	TypeRerank    = "rerank"
	TypeVLLM      = "vllm"
	TypeASR       = "asr"
)

const (
	RoleEmbedding     = "embedding"
	RoleSummary       = "summary"
	RoleWikiSynthesis = "wiki_synthesis"
	RoleVLM           = "vlm"
	RoleASR           = "asr"
	RoleRerank        = "rerank"
)

// RoleOrder is the stable order of scene bindings.
var RoleOrder = []string{RoleEmbedding, RoleSummary, RoleWikiSynthesis, RoleVLM, RoleASR, RoleRerank}

var roleType = map[string]string{
	RoleEmbedding:     TypeEmbedding,
	RoleSummary:       TypeChat,
	RoleWikiSynthesis: TypeChat,
	RoleVLM:           TypeVLLM,
	RoleASR:           TypeASR,
	RoleRerank:        TypeRerank,
}

var roleLabel = map[string]string{
	RoleEmbedding:     "向量",
	RoleSummary:       "摘要",
	RoleWikiSynthesis: "Wiki 生成",
	RoleVLM:           "视觉",
	RoleASR:           "语音",
	RoleRerank:        "重排",
}

var weknoraType = map[string]string{
	TypeChat:      "KnowledgeQA",
	TypeEmbedding: "Embedding",
	TypeRerank:    "Rerank",
	TypeVLLM:      "VLLM",
	TypeASR:       "ASR",
}

var semanticType = map[string]string{
	"KnowledgeQA": TypeChat,
	"Embedding":   TypeEmbedding,
	"Rerank":      TypeRerank,
	"VLLM":        TypeVLLM,
	"ASR":         TypeASR,
}

// ToWeKnoraType maps a frontend semantic type to the value WeKnora stores.
func ToWeKnoraType(semantic string) (string, bool) {
	value, ok := weknoraType[semantic]
	return value, ok
}

// FromWeKnoraType maps a stored WeKnora type to the frontend semantic type.
func FromWeKnoraType(raw string) (string, bool) {
	value, ok := semanticType[raw]
	return value, ok
}

// NormalizeType accepts either a semantic type or a WeKnora type and returns the semantic value.
func NormalizeType(raw string) (string, bool) {
	if value, ok := FromWeKnoraType(raw); ok {
		return value, true
	}
	if _, ok := ToWeKnoraType(raw); ok {
		return raw, true
	}
	return "", false
}

// RoleModelType returns the semantic model type required by a scene role.
func RoleModelType(role string) (string, bool) {
	value, ok := roleType[role]
	return value, ok
}

// RoleLabel is the short Chinese name of a scene role.
func RoleLabel(role string) string {
	if label, ok := roleLabel[role]; ok {
		return label
	}
	return role
}

// CredentialState reports whether a secret is stored, never the secret itself.
type CredentialState struct {
	Configured bool `json:"configured"`
}

// Credentials is the credential status WeKnora returns.
type Credentials struct {
	APIKey    CredentialState `json:"api_key"`
	AppSecret CredentialState `json:"app_secret"`
}

// Model is a secret-free model row.
type Model struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	DisplayName  string         `json:"display_name"`
	Type         string         `json:"type"`
	Source       string         `json:"source"`
	Description  string         `json:"description"`
	Parameters   map[string]any `json:"parameters"`
	IsBuiltin    bool           `json:"is_builtin"`
	Status       string         `json:"status"`
	Credentials  Credentials    `json:"credentials"`
	Capabilities map[string]any `json:"capabilities,omitempty"`
	BoundRoles   []string       `json:"bound_roles"`
	CreatedAt    string         `json:"created_at"`
	UpdatedAt    string         `json:"updated_at"`
}

// ExtraOption is one choice of a provider extra field.
type ExtraOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// ExtraField is a vendor-specific input rendered by the editor.
type ExtraField struct {
	Key         string        `json:"key"`
	Label       string        `json:"label"`
	Type        string        `json:"type"`
	Options     []ExtraOption `json:"options,omitempty"`
	Required    bool          `json:"required"`
	Placeholder string        `json:"placeholder,omitempty"`
	ModelTypes  []string      `json:"model_types"`
	Secret      bool          `json:"secret"`
}

// CatalogModel is one built-in model offered by a provider.
type CatalogModel struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Type            string   `json:"type"`
	API             string   `json:"api,omitempty"`
	Reasoning       bool     `json:"reasoning"`
	ContextWindow   int      `json:"context_window,omitempty"`
	MaxOutputTokens int      `json:"max_output_tokens,omitempty"`
	Dimension       int      `json:"dimension,omitempty"`
	ThinkingLevels  []string `json:"thinking_levels,omitempty"`
}

// Provider is a vendor definition with icons optional.
type Provider struct {
	Value            string            `json:"value"`
	Label            string            `json:"label"`
	Description      string            `json:"description"`
	Icon             string            `json:"icon,omitempty"`
	API              string            `json:"api,omitempty"`
	Auth             string            `json:"auth,omitempty"`
	RequiresAuth     bool              `json:"requires_auth"`
	DefaultURLs      map[string]string `json:"default_urls"`
	ModelTypes       []string          `json:"model_types"`
	ExtraFields      []ExtraField      `json:"extra_fields"`
	CredentialLabels []map[string]any  `json:"credential_labels"`
	Models           []CatalogModel    `json:"models"`
	Thinking         map[string]any    `json:"thinking,omitempty"`
	Order            int               `json:"order"`
}

// UsageResource is a knowledge base or agent that still references a model.
type UsageResource struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Bindings []string `json:"bindings"`
}

// Usage is the structured reason a model cannot be deleted.
type Usage struct {
	KnowledgeBases     []UsageResource `json:"knowledge_bases"`
	Agents             []UsageResource `json:"agents"`
	LongTermMemory     MemoryUsage     `json:"long_term_memory"`
	KnowledgeBaseTotal int64           `json:"knowledge_base_total"`
	AgentTotal         int64           `json:"agent_total"`
	LocalRoles         []string        `json:"local_roles,omitempty"`
}

// MemoryUsage lists long-term memory pins.
type MemoryUsage struct {
	Bindings []string `json:"bindings"`
}

// TestResult is a connectivity check outcome.
type TestResult struct {
	Available bool   `json:"available"`
	Message   string `json:"message"`
	Dimension *int   `json:"dimension,omitempty"`
	ElapsedMS int64  `json:"elapsed_ms"`
}

// OllamaStatus is the local runtime status.
type OllamaStatus struct {
	Available bool   `json:"available"`
	Version   string `json:"version"`
	BaseURL   string `json:"base_url"`
	Error     string `json:"error,omitempty"`
}

// OllamaModel is an installed local model.
type OllamaModel struct {
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	ModifiedAt string `json:"modified_at"`
}

// OllamaTask is a download progress record.
type OllamaTask struct {
	ID        string  `json:"id"`
	ModelName string  `json:"model_name"`
	Status    string  `json:"status"`
	Progress  float64 `json:"progress"`
	Message   string  `json:"message"`
}

func scrubJSON(value any) any {
	switch item := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(item))
		for key, child := range item {
			if secretString(key, child) {
				continue
			}
			out[key] = scrubJSON(child)
		}
		return out
	case []any:
		out := make([]any, len(item))
		for i, child := range item {
			out[i] = scrubJSON(child)
		}
		return out
	default:
		return value
	}
}

func secretString(key string, value any) bool {
	if _, ok := value.(string); !ok {
		return false
	}
	lower := strings.ToLower(key)
	switch lower {
	case "api_key", "apikey", "app_secret", "appsecret", "authorization":
		return true
	}
	return strings.Contains(lower, "secret") || strings.Contains(lower, "password") || strings.Contains(lower, "token") || strings.Contains(lower, "api_key")
}

func decodeModel(raw json.RawMessage) (Model, error) {
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return Model{}, err
	}
	payload, _ = scrubJSON(payload).(map[string]any)
	model := Model{
		ID: stringField(payload, "id"), Name: stringField(payload, "name"),
		DisplayName: stringField(payload, "display_name"), Source: stringField(payload, "source"),
		Description: stringField(payload, "description"), Status: stringField(payload, "status"),
		IsBuiltin: boolField(payload, "is_builtin"), CreatedAt: stringField(payload, "created_at"),
		UpdatedAt: stringField(payload, "updated_at"), BoundRoles: []string{},
		Parameters: map[string]any{},
	}
	if semantic, ok := NormalizeType(stringField(payload, "type")); ok {
		model.Type = semantic
	}
	if params, ok := payload["parameters"].(map[string]any); ok {
		model.Parameters = params
	}
	if caps, ok := payload["capabilities"].(map[string]any); ok {
		model.Capabilities = caps
	}
	model.Credentials = decodeCredentials(payload["credentials"])
	return model, nil
}

func decodeCredentials(value any) Credentials {
	raw, _ := json.Marshal(value)
	var creds Credentials
	_ = json.Unmarshal(raw, &creds)
	return creds
}

func stringField(payload map[string]any, key string) string {
	value, _ := payload[key].(string)
	return value
}

func boolField(payload map[string]any, key string) bool {
	value, _ := payload[key].(bool)
	return value
}

func dimensionOf(params map[string]any) int {
	embedding, _ := params["embedding_parameters"].(map[string]any)
	switch value := embedding["dimension"].(type) {
	case float64:
		return int(value)
	case int:
		return value
	case json.Number:
		number, _ := value.Int64()
		return int(number)
	default:
		return 0
	}
}
