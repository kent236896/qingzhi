package weknora

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	sdk "github.com/Tencent/WeKnora/client"
)

const batchLimit = 200

var (
	ErrDuplicate = sdk.ErrDuplicateFile
	ErrNotFound  = errors.New("not found")
)

// APIError is a non-2xx response from WeKnora.
type APIError = sdk.APIError

// Client is the WeKnora operations this service needs.
type Client interface {
	GetSystemInfo(ctx context.Context) (SystemInfo, error)
	GetDeploymentCapabilities(ctx context.Context) (DeploymentCapabilities, error)
	CreateKnowledgeBase(ctx context.Context, input CreateKnowledgeBaseInput) (RemoteKnowledgeBase, error)
	DeleteKnowledgeBase(ctx context.Context, id string) error
	CreateKnowledgeFromFile(ctx context.Context, knowledgeBaseID, filePath, customFileName string) (RemoteKnowledge, error)
	ListKnowledge(ctx context.Context, knowledgeBaseID string) ([]RemoteKnowledge, error)
	DeleteKnowledge(ctx context.Context, id string) error
	BatchDeleteKnowledge(ctx context.Context, knowledgeBaseID string, ids []string) error
	ReparseKnowledge(ctx context.Context, id string) error
	BatchReparseKnowledge(ctx context.Context, knowledgeBaseID string, ids []string) error
	WikiGraph(ctx context.Context, knowledgeBaseID string, query url.Values) (WikiGraph, error)
	WikiStats(ctx context.Context, knowledgeBaseID string) (WikiStats, error)
	WikiPages(ctx context.Context, knowledgeBaseID string, query url.Values) (WikiPageList, error)
	WikiPage(ctx context.Context, knowledgeBaseID, slug string) (WikiPage, error)
	WikiSearch(ctx context.Context, knowledgeBaseID, keyword string) (WikiPageList, error)
}

// HTTPClient calls WeKnora through the official Go SDK.
// Creating a knowledge base and reading Wiki still use the SDK raw request,
// because the typed create payload omits indexing and wiki settings, and the SDK has no Wiki methods.
type HTTPClient struct {
	sdk *sdk.Client
}

func NewHTTPClient(baseURL, apiKey string, timeout time.Duration) *HTTPClient {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &HTTPClient{
		sdk: sdk.NewClient(strings.TrimRight(baseURL, "/"), sdk.WithAPIKey(apiKey), sdk.WithTimeout(timeout)),
	}
}

type SystemInfo struct {
	Version string `json:"version"`
	Edition string `json:"edition"`
}

type DeploymentCapabilities struct {
	Edition      string                    `json:"edition"`
	Capabilities map[string]CapabilityItem `json:"capabilities"`
}

type CapabilityItem struct {
	Supported bool   `json:"supported"`
	Reason    string `json:"reason"`
}

type IndexingStrategy struct {
	VectorEnabled  bool `json:"vector_enabled"`
	KeywordEnabled bool `json:"keyword_enabled"`
	WikiEnabled    bool `json:"wiki_enabled"`
	GraphEnabled   bool `json:"graph_enabled"`
}

type WikiConfig struct {
	SynthesisModelID      string `json:"synthesis_model_id"`
	ExtractionGranularity string `json:"extraction_granularity"`
}

type CreateKnowledgeBaseInput struct {
	Name             string
	Description      string
	EmbeddingModelID string
	SummaryModelID   string
	WikiEnabled      bool
	Wiki             *WikiConfig
}

type RemoteKnowledgeBase struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type RemoteKnowledge struct {
	ID           string `json:"id"`
	FileName     string `json:"file_name"`
	FolderPath   string `json:"folder_path"`
	FileSize     int64  `json:"file_size"`
	ParseStatus  string `json:"parse_status"`
	ErrorMessage string `json:"error_message"`
}

type WikiGraph struct {
	Nodes []WikiNode `json:"nodes"`
	Edges []WikiEdge `json:"edges"`
	Meta  WikiMeta   `json:"meta"`
}

type WikiNode struct {
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	PageType  string `json:"page_type"`
	LinkCount int    `json:"link_count"`
}

type WikiEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

type WikiMeta struct {
	Mode      string `json:"mode"`
	Total     int    `json:"total"`
	Returned  int    `json:"returned"`
	Truncated bool   `json:"truncated"`
}

type WikiStats struct {
	TotalPages    int            `json:"total_pages"`
	PagesByType   map[string]int `json:"pages_by_type"`
	TotalLinks    int            `json:"total_links"`
	OrphanCount   int            `json:"orphan_count"`
	PendingTasks  int            `json:"pending_tasks"`
	PendingIssues int            `json:"pending_issues"`
	IsActive      bool           `json:"is_active"`
}

type WikiPageList struct {
	Pages     []WikiPage `json:"pages"`
	Total     int64      `json:"total"`
	Page      int        `json:"page"`
	PageSize  int        `json:"page_size"`
	TotalPage int        `json:"total_pages"`
}

type WikiPage struct {
	ID       string `json:"id"`
	Slug     string `json:"slug"`
	Title    string `json:"title"`
	PageType string `json:"page_type"`
	Summary  string `json:"summary"`
	Content  string `json:"content"`
}

func (c *HTTPClient) GetSystemInfo(ctx context.Context) (SystemInfo, error) {
	info, err := c.sdk.GetSystemInfo(ctx)
	if err != nil {
		return SystemInfo{}, wrap(err)
	}
	if info == nil {
		return SystemInfo{}, nil
	}
	return SystemInfo{Version: info.Version, Edition: info.Edition}, nil
}

func (c *HTTPClient) GetDeploymentCapabilities(ctx context.Context) (DeploymentCapabilities, error) {
	caps, err := c.sdk.GetDeploymentCapabilities(ctx)
	if err != nil {
		return DeploymentCapabilities{}, wrap(err)
	}
	if caps == nil {
		return DeploymentCapabilities{}, nil
	}
	out := DeploymentCapabilities{Edition: caps.Edition, Capabilities: map[string]CapabilityItem{}}
	for name, item := range caps.Capabilities {
		out.Capabilities[name] = CapabilityItem{Supported: item.Supported, Reason: item.Reason}
	}
	return out, nil
}

func (c *HTTPClient) CreateKnowledgeBase(ctx context.Context, input CreateKnowledgeBaseInput) (RemoteKnowledgeBase, error) {
	body := map[string]any{
		"name":               input.Name,
		"description":        input.Description,
		"type":               "document",
		"embedding_model_id": input.EmbeddingModelID,
		"summary_model_id":   input.SummaryModelID,
		"chunking_config": map[string]any{
			"chunk_size":          512,
			"chunk_overlap":       80,
			"child_chunk_size":    384,
			"parent_chunk_size":   4096,
			"enable_parent_child": true,
			"strategy":            "auto",
			"separators":          []string{"\n\n", "\n", "。", "！", "？", ";", "；"},
		},
		"indexing_strategy": IndexingStrategy{
			VectorEnabled: true, KeywordEnabled: true, WikiEnabled: input.WikiEnabled, GraphEnabled: false,
		},
	}
	if input.WikiEnabled && input.Wiki != nil {
		body["wiki_config"] = input.Wiki
	}
	var envelope struct {
		Data RemoteKnowledgeBase `json:"data"`
	}
	err := c.readRaw(ctx, http.MethodPost, "/api/v1/knowledge-bases", body, &envelope)
	return envelope.Data, err
}

func (c *HTTPClient) DeleteKnowledgeBase(ctx context.Context, id string) error {
	return wrap(c.sdk.DeleteKnowledgeBase(ctx, id))
}

func (c *HTTPClient) CreateKnowledgeFromFile(ctx context.Context, knowledgeBaseID, filePath, customFileName string) (RemoteKnowledge, error) {
	item, err := c.sdk.CreateKnowledgeFromFile(ctx, knowledgeBaseID, filePath, nil, nil, customFileName, "", nil)
	if err != nil {
		if item != nil && errors.Is(err, ErrDuplicate) {
			return remoteKnowledge(*item), ErrDuplicate
		}
		return RemoteKnowledge{}, wrap(err)
	}
	if item == nil {
		return RemoteKnowledge{}, nil
	}
	return remoteKnowledge(*item), nil
}

func (c *HTTPClient) ListKnowledge(ctx context.Context, knowledgeBaseID string) ([]RemoteKnowledge, error) {
	var all []RemoteKnowledge
	for page := 1; page < 100; page++ {
		items, total, err := c.sdk.ListKnowledge(ctx, knowledgeBaseID, page, 100, "")
		if err != nil {
			return nil, wrap(err)
		}
		for _, item := range items {
			all = append(all, remoteKnowledge(item))
		}
		if int64(len(all)) >= total || len(items) == 0 {
			break
		}
	}
	if all == nil {
		all = []RemoteKnowledge{}
	}
	return all, nil
}

func (c *HTTPClient) DeleteKnowledge(ctx context.Context, id string) error {
	return wrap(c.sdk.DeleteKnowledge(ctx, id))
}

func (c *HTTPClient) BatchDeleteKnowledge(ctx context.Context, knowledgeBaseID string, ids []string) error {
	return c.batch(ctx, "/api/v1/knowledge/batch-delete", knowledgeBaseID, ids)
}

func (c *HTTPClient) ReparseKnowledge(ctx context.Context, id string) error {
	_, err := c.sdk.ReparseKnowledge(ctx, id)
	return wrap(err)
}

func (c *HTTPClient) BatchReparseKnowledge(ctx context.Context, knowledgeBaseID string, ids []string) error {
	return c.batch(ctx, "/api/v1/knowledge/batch-reparse", knowledgeBaseID, ids)
}

func (c *HTTPClient) batch(ctx context.Context, path, knowledgeBaseID string, ids []string) error {
	for start := 0; start < len(ids); start += batchLimit {
		end := start + batchLimit
		if end > len(ids) {
			end = len(ids)
		}
		body := map[string]any{"kb_id": knowledgeBaseID, "ids": ids[start:end]}
		if err := c.readRaw(ctx, http.MethodPost, path, body, nil); err != nil {
			return err
		}
	}
	return nil
}

func (c *HTTPClient) WikiGraph(ctx context.Context, knowledgeBaseID string, query url.Values) (WikiGraph, error) {
	var graph WikiGraph
	err := c.readRaw(ctx, http.MethodGet, "/api/v1/knowledgebase/"+url.PathEscape(knowledgeBaseID)+"/wiki/graph?"+query.Encode(), nil, &graph)
	return graph, err
}

func (c *HTTPClient) WikiStats(ctx context.Context, knowledgeBaseID string) (WikiStats, error) {
	var stats WikiStats
	err := c.readRaw(ctx, http.MethodGet, "/api/v1/knowledgebase/"+url.PathEscape(knowledgeBaseID)+"/wiki/stats", nil, &stats)
	return stats, err
}

func (c *HTTPClient) WikiPages(ctx context.Context, knowledgeBaseID string, query url.Values) (WikiPageList, error) {
	var list WikiPageList
	err := c.readRaw(ctx, http.MethodGet, "/api/v1/knowledgebase/"+url.PathEscape(knowledgeBaseID)+"/wiki/pages?"+query.Encode(), nil, &list)
	return list, err
}

func (c *HTTPClient) WikiPage(ctx context.Context, knowledgeBaseID, slug string) (WikiPage, error) {
	var page WikiPage
	err := c.readRaw(ctx, http.MethodGet, "/api/v1/knowledgebase/"+url.PathEscape(knowledgeBaseID)+"/wiki/pages/"+escapeSlug(slug), nil, &page)
	return page, err
}

func (c *HTTPClient) WikiSearch(ctx context.Context, knowledgeBaseID, keyword string) (WikiPageList, error) {
	query := url.Values{}
	query.Set("q", keyword)
	var list WikiPageList
	err := c.readRaw(ctx, http.MethodGet, "/api/v1/knowledgebase/"+url.PathEscape(knowledgeBaseID)+"/wiki/search?"+query.Encode(), nil, &list)
	return list, err
}

func remoteKnowledge(item sdk.Knowledge) RemoteKnowledge {
	return RemoteKnowledge{
		ID: item.ID, FileName: item.FileName, FolderPath: item.FolderPath,
		FileSize: item.FileSize, ParseStatus: item.ParseStatus, ErrorMessage: item.ErrorMessage,
	}
}

func escapeSlug(slug string) string {
	parts := strings.Split(slug, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

func (c *HTTPClient) readRaw(ctx context.Context, method, path string, body any, target any) error {
	resp, err := c.sdk.Raw(ctx, method, path, body)
	if err != nil {
		return wrap(err)
	}
	defer resp.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{StatusCode: resp.StatusCode, Body: string(payload)}
	}
	if target == nil || len(payload) == 0 {
		return nil
	}
	return json.Unmarshal(payload, target)
}

func wrap(err error) error {
	if err == nil {
		return nil
	}
	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	return err
}

// WithRequestID returns a context that forwards X-Request-ID.
// The official SDK reads this value with the string key "RequestID".
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, "RequestID", id)
}
