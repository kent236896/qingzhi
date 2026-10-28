package weknora

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	sdk "github.com/Tencent/WeKnora/client"
)

// ChatSession is a WeKnora conversation container.
type ChatSession struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// ChatMessage is one turn in a session.
type ChatMessage struct {
	ID                  string              `json:"id"`
	SessionID           string              `json:"session_id"`
	RequestID           string              `json:"request_id"`
	Content             string              `json:"content"`
	Role                string              `json:"role"`
	KnowledgeReferences []ChatReference     `json:"knowledge_references"`
	IsCompleted         bool                `json:"is_completed"`
	CreatedAt           time.Time           `json:"created_at"`
	UpdatedAt           time.Time           `json:"updated_at"`
}

// ChatReference is a knowledge citation fragment.
type ChatReference struct {
	ID                string  `json:"id"`
	Content           string  `json:"content"`
	KnowledgeID       string  `json:"knowledge_id"`
	KnowledgeTitle    string  `json:"knowledge_title"`
	KnowledgeFilename string  `json:"knowledge_filename"`
	Score             float64 `json:"score"`
	KnowledgeBaseID   string  `json:"knowledge_base_id"`
	ChunkIndex        int     `json:"chunk_index"`
}

// KnowledgeQARequest is the body for POST /api/v1/knowledge-chat/{session_id}.
type KnowledgeQARequest struct {
	Query            string   `json:"query"`
	KnowledgeBaseIDs []string `json:"knowledge_base_ids"`
	AgentEnabled     bool     `json:"agent_enabled"`
}

// CreateSession creates a WeKnora session as the terminal user in ctx.
func (c *HTTPClient) CreateSession(ctx context.Context, title, description string) (ChatSession, error) {
	client, _, _ := c.session()
	if client == nil {
		return ChatSession{}, fmt.Errorf("weknora client is not configured")
	}
	item, err := client.CreateSession(ctx, &sdk.CreateSessionRequest{
		Title:       title,
		Description: description,
	})
	if err != nil {
		return ChatSession{}, wrap(err)
	}
	return mapSession(item), nil
}

// GetSession loads one session.
func (c *HTTPClient) GetSession(ctx context.Context, sessionID string) (ChatSession, error) {
	client, _, _ := c.session()
	if client == nil {
		return ChatSession{}, fmt.Errorf("weknora client is not configured")
	}
	item, err := client.GetSession(ctx, sessionID)
	if err != nil {
		return ChatSession{}, wrap(err)
	}
	return mapSession(item), nil
}

// ListSessions lists sessions for the current WeKnora principal.
func (c *HTTPClient) ListSessions(ctx context.Context, page, pageSize int) ([]ChatSession, int, error) {
	client, _, _ := c.session()
	if client == nil {
		return nil, 0, fmt.Errorf("weknora client is not configured")
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	items, total, err := client.GetSessionsByTenant(ctx, page, pageSize)
	if err != nil {
		return nil, 0, wrap(err)
	}
	out := make([]ChatSession, 0, len(items))
	for i := range items {
		out = append(out, mapSession(&items[i]))
	}
	return out, total, nil
}

// UpdateSession renames a session.
func (c *HTTPClient) UpdateSession(ctx context.Context, sessionID, title, description string) (ChatSession, error) {
	client, _, _ := c.session()
	if client == nil {
		return ChatSession{}, fmt.Errorf("weknora client is not configured")
	}
	item, err := client.UpdateSession(ctx, sessionID, &sdk.CreateSessionRequest{
		Title:       title,
		Description: description,
	})
	if err != nil {
		return ChatSession{}, wrap(err)
	}
	return mapSession(item), nil
}

// DeleteSession removes a session.
func (c *HTTPClient) DeleteSession(ctx context.Context, sessionID string) error {
	client, _, _ := c.session()
	if client == nil {
		return fmt.Errorf("weknora client is not configured")
	}
	return wrap(client.DeleteSession(ctx, sessionID))
}

// LoadMessages loads history for a session.
func (c *HTTPClient) LoadMessages(ctx context.Context, sessionID string, limit int, before *time.Time) ([]ChatMessage, error) {
	client, _, _ := c.session()
	if client == nil {
		return nil, fmt.Errorf("weknora client is not configured")
	}
	if limit < 1 || limit > 200 {
		limit = 50
	}
	items, err := client.LoadMessages(ctx, sessionID, limit, before)
	if err != nil {
		return nil, wrap(err)
	}
	out := make([]ChatMessage, 0, len(items))
	for i := range items {
		out = append(out, mapMessage(&items[i]))
	}
	return out, nil
}

// StopSession stops generation for an assistant message.
func (c *HTTPClient) StopSession(ctx context.Context, sessionID, messageID string) error {
	client, _, _ := c.session()
	if client == nil {
		return fmt.Errorf("weknora client is not configured")
	}
	return wrap(client.StopSession(ctx, sessionID, messageID))
}

// OpenKnowledgeChatStream POSTs knowledge-chat and returns the raw SSE response body.
// Caller must close the body. Uses a client without blanket Timeout so streams can run.
func (c *HTTPClient) OpenKnowledgeChatStream(ctx context.Context, sessionID string, req KnowledgeQARequest) (*http.Response, error) {
	_, baseURL, apiKey := c.session()
	if baseURL == "" || apiKey == "" {
		return nil, fmt.Errorf("weknora client is not configured")
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, fmt.Errorf("session id is required")
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	path := "/api/v1/knowledge-chat/" + url.PathEscape(sessionID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("X-API-Key", apiKey)
	if token := ExternalUserTokenFrom(ctx); token != "" {
		httpReq.Header.Set(ExternalUserTokenHeader, token)
	}
	if reqID := ctx.Value("RequestID"); reqID != nil {
		if s, ok := reqID.(string); ok && s != "" {
			httpReq.Header.Set("X-Request-ID", s)
		}
	}
	streamHTTP := &http.Client{
		Timeout:   0,
		Transport: wrapExternalTokenTransport(http.DefaultTransport),
	}
	resp, err := streamHTTP.Do(httpReq)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		_ = resp.Body.Close()
		return nil, &APIError{StatusCode: resp.StatusCode, Body: strings.TrimSpace(string(body))}
	}
	return resp, nil
}

func mapSession(item *sdk.Session) ChatSession {
	if item == nil {
		return ChatSession{}
	}
	return ChatSession{
		ID:          item.ID,
		Title:       item.Title,
		Description: item.Description,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}
}

func mapMessage(item *sdk.Message) ChatMessage {
	if item == nil {
		return ChatMessage{}
	}
	refs := make([]ChatReference, 0, len(item.KnowledgeReferences))
	for _, ref := range item.KnowledgeReferences {
		if ref == nil {
			continue
		}
		refs = append(refs, ChatReference{
			ID:                ref.ID,
			Content:           ref.Content,
			KnowledgeID:       ref.KnowledgeID,
			KnowledgeTitle:    ref.KnowledgeTitle,
			KnowledgeFilename: ref.KnowledgeFilename,
			Score:             ref.Score,
			KnowledgeBaseID:   ref.KnowledgeBaseID,
			ChunkIndex:        ref.ChunkIndex,
		})
	}
	return ChatMessage{
		ID:                  item.ID,
		SessionID:           item.SessionID,
		RequestID:           item.RequestID,
		Content:             item.Content,
		Role:                item.Role,
		KnowledgeReferences: refs,
		IsCompleted:         item.IsCompleted,
		CreatedAt:           item.CreatedAt,
		UpdatedAt:           item.UpdatedAt,
	}
}

// ParsePageQuery helpers for handlers.
func ParsePageQuery(pageStr, sizeStr string) (int, int) {
	page, _ := strconv.Atoi(pageStr)
	size, _ := strconv.Atoi(sizeStr)
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	return page, size
}
