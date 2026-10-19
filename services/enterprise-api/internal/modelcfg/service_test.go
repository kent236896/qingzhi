package modelcfg

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"qingzhi/enterprise-api/internal/config"
)

func TestTypeMappingBothWays(t *testing.T) {
	pairs := map[string]string{
		TypeChat: "KnowledgeQA", TypeEmbedding: "Embedding", TypeRerank: "Rerank", TypeVLLM: "VLLM", TypeASR: "ASR",
	}
	for semantic, remote := range pairs {
		got, ok := ToWeKnoraType(semantic)
		if !ok || got != remote {
			t.Fatalf("to %s = %s %v", semantic, got, ok)
		}
		back, ok := FromWeKnoraType(remote)
		if !ok || back != semantic {
			t.Fatalf("from %s = %s %v", remote, back, ok)
		}
		normalized, ok := NormalizeType(remote)
		if !ok || normalized != semantic {
			t.Fatalf("normalize %s = %s", remote, normalized)
		}
	}
	if _, ok := ToWeKnoraType("KnowledgeQA"); ok {
		t.Fatal("raw weknora type must not be accepted as a frontend type")
	}
}

func TestResponseOmitsSecrets(t *testing.T) {
	raw := []byte(`{
		"id":"m1","name":"demo","type":"Embedding","source":"remote","parameters":{
			"api_key":"sk-secret-value","app_secret":"app-secret-value","base_url":"http://example.test/v1",
			"embedding_parameters":{"dimension":8},"extra_config":{"secret_key":"hidden","region":"cn"}
		},
		"credentials":{"api_key":{"configured":true},"app_secret":{"configured":false}},
		"is_builtin":false
	}`)
	model, err := decodeModel(raw)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(model)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	if strings.Contains(text, "sk-secret-value") || strings.Contains(text, "app-secret-value") || strings.Contains(text, "hidden") {
		t.Fatalf("secret leaked: %s", text)
	}
	if !strings.Contains(text, `"configured":true`) || !strings.Contains(text, "http://example.test/v1") || !strings.Contains(text, `"region":"cn"`) {
		t.Fatalf("status or public fields missing: %s", text)
	}
	if model.Type != TypeEmbedding {
		t.Fatalf("type = %s", model.Type)
	}
}

func TestUpdateEmptyAPIKeySkipsCredentials(t *testing.T) {
	remote := &fakeRemote{model: sampleModel()}
	svc := NewService(&memStore{}, remote, enabledConfig(), nil)
	_, err := svc.Update(context.Background(), "m1", SaveInput{
		Name: "demo", Source: "remote", Type: TypeChat, Parameters: map[string]any{"base_url": "https://example.test/v1"}, APIKey: "",
	})
	if err != nil {
		t.Fatal(err)
	}
	if remote.credentials != 0 {
		t.Fatalf("credentials calls = %d", remote.credentials)
	}
	if _, ok := remote.updated["api_key"]; ok {
		t.Fatal("update body included api_key")
	}
}

func TestDeleteInUseConverts2300(t *testing.T) {
	remote := &fakeRemote{model: sampleModel(), deleteErr: &RemoteError{
		Status: 400, Code: 2300, Message: "model is in use",
		Details: []byte(`{"knowledge_bases":[{"id":"kb","name":"资料","bindings":["embedding_model"]}],"agents":[],"long_term_memory":{"bindings":["extract_model"]},"knowledge_base_total":1,"agent_total":0}`),
	}}
	svc := NewService(&memStore{}, remote, enabledConfig(), nil)
	err := svc.Delete(context.Background(), "m1")
	var domain *Error
	if !errors.As(err, &domain) || !errors.Is(err, ErrConflict) {
		t.Fatalf("err = %#v", err)
	}
	usage, _ := domain.Data.(Usage)
	if usage.KnowledgeBaseTotal != 1 || len(usage.KnowledgeBases) != 1 || usage.KnowledgeBases[0].Name != "资料" {
		t.Fatalf("usage = %#v", usage)
	}
	if len(usage.LongTermMemory.Bindings) != 1 {
		t.Fatalf("memory = %#v", usage.LongTermMemory)
	}
}

func TestDeleteBlockedByLocalBinding(t *testing.T) {
	remote := &fakeRemote{model: sampleModel()}
	store := &memStore{items: []Binding{{Role: RoleEmbedding, ModelID: "m1"}}}
	svc := NewService(store, remote, enabledConfig(), nil)
	err := svc.Delete(context.Background(), "m1")
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("err = %v", err)
	}
	if remote.deleted {
		t.Fatal("remote delete was called")
	}
}

func TestBindingTypeMustMatchRole(t *testing.T) {
	remote := &fakeRemote{model: sampleModel()}
	svc := NewService(&memStore{}, remote, enabledConfig(), nil)
	_, err := svc.SaveBinding(context.Background(), RoleEmbedding, "m1", "admin")
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("err = %v", err)
	}
	remote.model.Type = TypeEmbedding
	item, err := svc.SaveBinding(context.Background(), RoleEmbedding, "m1", "admin")
	if err != nil {
		t.Fatal(err)
	}
	if item.Warning == "" {
		t.Fatal("expected embedding warning")
	}
}

func TestTestForwardsModelID(t *testing.T) {
	remote := &fakeRemote{model: sampleModel(), test: TestResult{Available: true, Message: "ok", Dimension: intPtr(1024)}}
	svc := NewService(&memStore{}, remote, enabledConfig(), nil)
	svc.now = func() time.Time { return time.Unix(0, 0) }
	result, err := svc.Test(context.Background(), TestInput{Type: TypeEmbedding, ModelName: "bge", ModelID: "stored-id", BaseURL: "http://10.0.0.8/v1"})
	if err != nil {
		t.Fatal(err)
	}
	if remote.tested["modelId"] != "stored-id" || remote.tested["modelName"] != "bge" {
		t.Fatalf("body = %#v", remote.tested)
	}
	if _, ok := remote.tested["apiKey"]; ok {
		t.Fatal("empty api key was forwarded")
	}
	if result.Dimension == nil || *result.Dimension != 1024 {
		t.Fatalf("result = %#v", result)
	}
}

func TestSeedDoesNotOverrideExisting(t *testing.T) {
	store := &memStore{items: []Binding{{Role: RoleEmbedding, ModelID: "kept"}}}
	cfg := enabledConfig()
	cfg.EmbeddingModelID = "from-env"
	svc := NewService(store, &fakeRemote{}, cfg, nil)
	if err := svc.SeedFallback(context.Background()); err != nil {
		t.Fatal(err)
	}
	item, err := store.Find(context.Background(), RoleEmbedding)
	if err != nil || item.ModelID != "kept" {
		t.Fatalf("item = %#v err = %v", item, err)
	}
	summary, err := store.Find(context.Background(), RoleSummary)
	if err != nil || summary.ModelID != "summary-env" {
		t.Fatalf("summary = %#v err = %v", summary, err)
	}
}

type fakeRemote struct {
	model       Model
	credentials int
	updated     map[string]any
	deleted     bool
	deleteErr   error
	tested      map[string]any
	test        TestResult
}

func (f *fakeRemote) ListProviders(context.Context, string) ([]Provider, error) { return nil, nil }
func (f *fakeRemote) Resolve(context.Context, string, string, string, string) (map[string]any, error) {
	return map[string]any{}, nil
}
func (f *fakeRemote) ListModels(context.Context) ([]Model, error)                { return []Model{f.model}, nil }
func (f *fakeRemote) GetModel(context.Context, string) (Model, error)            { return f.model, nil }
func (f *fakeRemote) CreateModel(context.Context, map[string]any) (Model, error) { return f.model, nil }
func (f *fakeRemote) UpdateModel(_ context.Context, _ string, body map[string]any) (Model, error) {
	f.updated = body
	return f.model, nil
}
func (f *fakeRemote) DeleteModel(context.Context, string) error {
	f.deleted = true
	return f.deleteErr
}
func (f *fakeRemote) CopyModel(context.Context, string, string) (Model, error) { return f.model, nil }
func (f *fakeRemote) PutCredentials(context.Context, string, string, string) (Credentials, error) {
	f.credentials++
	return Credentials{}, nil
}
func (f *fakeRemote) DeleteCredential(context.Context, string, string) error { return nil }
func (f *fakeRemote) Test(_ context.Context, _ string, body map[string]any) (TestResult, error) {
	f.tested = body
	return f.test, nil
}
func (f *fakeRemote) Debug(context.Context, string, string, []byte) (map[string]any, error) {
	return map[string]any{}, nil
}
func (f *fakeRemote) OllamaStatus(context.Context) (OllamaStatus, error)  { return OllamaStatus{}, nil }
func (f *fakeRemote) OllamaModels(context.Context) ([]OllamaModel, error) { return nil, nil }
func (f *fakeRemote) OllamaDownload(context.Context, string) (OllamaTask, error) {
	return OllamaTask{}, nil
}
func (f *fakeRemote) OllamaProgress(context.Context, string) (OllamaTask, error) {
	return OllamaTask{}, nil
}

type memStore struct{ items []Binding }

func (s *memStore) List(context.Context) ([]Binding, error) {
	return append([]Binding(nil), s.items...), nil
}
func (s *memStore) Find(_ context.Context, role string) (Binding, error) {
	for _, item := range s.items {
		if item.Role == role {
			return item, nil
		}
	}
	return Binding{}, ErrNotFound
}
func (s *memStore) Upsert(_ context.Context, item Binding) error {
	for i, current := range s.items {
		if current.Role == item.Role {
			s.items[i] = item
			return nil
		}
	}
	s.items = append(s.items, item)
	return nil
}

func sampleModel() Model {
	return Model{ID: "m1", Name: "demo", Type: TypeChat, Source: "remote", Parameters: map[string]any{"base_url": "https://example.test/v1"}, BoundRoles: []string{}}
}

func enabledConfig() config.WeKnoraConfig {
	return config.WeKnoraConfig{BaseURL: "http://example.test", APIKey: "test-key", SummaryModelID: "summary-env"}
}

func intPtr(value int) *int { return &value }
