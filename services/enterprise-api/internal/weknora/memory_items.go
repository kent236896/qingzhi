package weknora

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// PersonalMemorySettings is GET/PUT /api/v1/memory/settings for the terminal user.
type PersonalMemorySettings struct {
	WorkspaceEnabled bool   `json:"workspace_enabled"`
	UserEnabled      bool   `json:"user_enabled"`
	Effective        bool   `json:"effective"`
	WriteMode        string `json:"write_mode"`
	ItemCount        int    `json:"item_count"`
	MaxItems         int    `json:"max_items"`
}

// PersonalMemoryItem is one long-term memory entry.
type PersonalMemoryItem struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Content   string    `json:"content"`
	Status    string    `json:"status"`
	Importance int      `json:"importance"`
	Origin    string    `json:"origin,omitempty"`
	SubjectID string    `json:"subject_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PersonalMemoryList is a paginated item list.
type PersonalMemoryList struct {
	Items    []PersonalMemoryItem `json:"items"`
	Total    int                  `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
}

// GetPersonalMemorySettings reads settings for the caller identified by ctx token.
func (c *HTTPClient) GetPersonalMemorySettings(ctx context.Context) (PersonalMemorySettings, error) {
	if ExternalUserTokenFrom(ctx) == "" {
		return PersonalMemorySettings{}, fmt.Errorf("缺少终端用户身份令牌")
	}
	status, raw, err := c.Exchange(ctx, http.MethodGet, "/api/v1/memory/settings", nil)
	if err != nil {
		return PersonalMemorySettings{}, err
	}
	if status != http.StatusOK {
		return PersonalMemorySettings{}, mapMemoryHTTPError(status, raw)
	}
	var out PersonalMemorySettings
	if err := json.Unmarshal(unwrapMemoryData(raw), &out); err != nil {
		return PersonalMemorySettings{}, fmt.Errorf("parse memory settings: %w", err)
	}
	return out, nil
}

// UpdatePersonalMemorySettings updates the caller's personal switch (and related fields).
func (c *HTTPClient) UpdatePersonalMemorySettings(ctx context.Context, userEnabled bool) (PersonalMemorySettings, error) {
	if ExternalUserTokenFrom(ctx) == "" {
		return PersonalMemorySettings{}, fmt.Errorf("缺少终端用户身份令牌")
	}
	body := map[string]any{"enabled": userEnabled, "user_enabled": userEnabled}
	status, raw, err := c.Exchange(ctx, http.MethodPut, "/api/v1/memory/settings", body)
	if err != nil {
		return PersonalMemorySettings{}, err
	}
	if status != http.StatusOK {
		return PersonalMemorySettings{}, mapMemoryHTTPError(status, raw)
	}
	var out PersonalMemorySettings
	if err := json.Unmarshal(unwrapMemoryData(raw), &out); err != nil {
		return PersonalMemorySettings{}, fmt.Errorf("parse memory settings: %w", err)
	}
	return out, nil
}

// ListPersonalMemoryItems lists memory items for the caller.
func (c *HTTPClient) ListPersonalMemoryItems(ctx context.Context, page, pageSize int, kind, statusFilter string) (PersonalMemoryList, error) {
	if ExternalUserTokenFrom(ctx) == "" {
		return PersonalMemoryList{}, fmt.Errorf("缺少终端用户身份令牌")
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	q := url.Values{}
	q.Set("page", strconv.Itoa(page))
	q.Set("page_size", strconv.Itoa(pageSize))
	if kind = strings.TrimSpace(kind); kind != "" {
		q.Set("type", kind)
		q.Set("kind", kind)
	}
	if statusFilter = strings.TrimSpace(statusFilter); statusFilter != "" {
		q.Set("status", statusFilter)
	}
	path := "/api/v1/memory/items?" + q.Encode()
	httpStatus, raw, err := c.Exchange(ctx, http.MethodGet, path, nil)
	if err != nil {
		return PersonalMemoryList{}, err
	}
	if httpStatus != http.StatusOK {
		return PersonalMemoryList{}, mapMemoryHTTPError(httpStatus, raw)
	}
	var envelope struct {
		Success bool                 `json:"success"`
		Total   int                  `json:"total"`
		Data    []PersonalMemoryItem `json:"data"`
		Items   []PersonalMemoryItem `json:"items"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return PersonalMemoryList{}, fmt.Errorf("parse memory items: %w", err)
	}
	items := envelope.Data
	if items == nil {
		items = envelope.Items
	}
	if items == nil {
		items = []PersonalMemoryItem{}
	}
	total := envelope.Total
	if total == 0 {
		total = len(items)
	}
	return PersonalMemoryList{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

// CreatePersonalMemoryItem creates one memory entry for the caller.
func (c *HTTPClient) CreatePersonalMemoryItem(ctx context.Context, kind, content string, importance int) (PersonalMemoryItem, error) {
	if ExternalUserTokenFrom(ctx) == "" {
		return PersonalMemoryItem{}, fmt.Errorf("缺少终端用户身份令牌")
	}
	body := map[string]any{
		"type":     kind,
		"kind":     kind,
		"content":  content,
		"priority": importance,
		"importance": importance,
	}
	status, raw, err := c.Exchange(ctx, http.MethodPost, "/api/v1/memory/items", body)
	if err != nil {
		return PersonalMemoryItem{}, err
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return PersonalMemoryItem{}, mapMemoryHTTPError(status, raw)
	}
	var out PersonalMemoryItem
	if err := json.Unmarshal(unwrapMemoryData(raw), &out); err != nil {
		return PersonalMemoryItem{}, fmt.Errorf("parse memory item: %w", err)
	}
	return out, nil
}

// UpdatePersonalMemoryItem updates one memory entry owned by the caller.
func (c *HTTPClient) UpdatePersonalMemoryItem(ctx context.Context, id, kind, content string, importance int) (PersonalMemoryItem, error) {
	if ExternalUserTokenFrom(ctx) == "" {
		return PersonalMemoryItem{}, fmt.Errorf("缺少终端用户身份令牌")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return PersonalMemoryItem{}, fmt.Errorf("缺少记忆标识")
	}
	body := map[string]any{
		"type":       kind,
		"kind":       kind,
		"content":    content,
		"priority":   importance,
		"importance": importance,
	}
	status, raw, err := c.Exchange(ctx, http.MethodPut, "/api/v1/memory/items/"+url.PathEscape(id), body)
	if err != nil {
		return PersonalMemoryItem{}, err
	}
	if status != http.StatusOK {
		return PersonalMemoryItem{}, mapMemoryHTTPError(status, raw)
	}
	var out PersonalMemoryItem
	if err := json.Unmarshal(unwrapMemoryData(raw), &out); err != nil {
		return PersonalMemoryItem{}, fmt.Errorf("parse memory item: %w", err)
	}
	return out, nil
}

// DeletePersonalMemoryItem deletes one memory entry owned by the caller.
func (c *HTTPClient) DeletePersonalMemoryItem(ctx context.Context, id string) error {
	if ExternalUserTokenFrom(ctx) == "" {
		return fmt.Errorf("缺少终端用户身份令牌")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("缺少记忆标识")
	}
	status, raw, err := c.Exchange(ctx, http.MethodDelete, "/api/v1/memory/items/"+url.PathEscape(id), nil)
	if err != nil {
		return err
	}
	if status != http.StatusOK && status != http.StatusNoContent {
		return mapMemoryHTTPError(status, raw)
	}
	return nil
}

// ConfirmPersonalMemoryItem confirms a pending item.
func (c *HTTPClient) ConfirmPersonalMemoryItem(ctx context.Context, id string) error {
	return c.personalMemoryAction(ctx, id, "confirm")
}

// RejectPersonalMemoryItem rejects a pending item.
func (c *HTTPClient) RejectPersonalMemoryItem(ctx context.Context, id string) error {
	return c.personalMemoryAction(ctx, id, "reject")
}

func (c *HTTPClient) personalMemoryAction(ctx context.Context, id, action string) error {
	if ExternalUserTokenFrom(ctx) == "" {
		return fmt.Errorf("缺少终端用户身份令牌")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("缺少记忆标识")
	}
	status, raw, err := c.Exchange(ctx, http.MethodPost, "/api/v1/memory/items/"+url.PathEscape(id)+"/"+action, map[string]any{})
	if err != nil {
		return err
	}
	if status == http.StatusConflict {
		return &MemoryAPIError{StatusCode: status, Message: "记忆状态已变化，请刷新后重试"}
	}
	if status != http.StatusOK && status != http.StatusNoContent {
		return mapMemoryHTTPError(status, raw)
	}
	return nil
}

// MemoryAPIError is a WeKnora personal-memory HTTP error.
type MemoryAPIError struct {
	StatusCode int
	Message    string
}

func (e *MemoryAPIError) Error() string {
	if e == nil {
		return "memory api error"
	}
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("memory api http %d", e.StatusCode)
}

func unwrapMemoryData(raw []byte) []byte {
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return raw
	}
	if data, ok := envelope["data"]; ok && len(data) > 0 && string(data) != "null" {
		// Prefer unwrapping object payloads; list endpoints keep the outer envelope.
		trim := strings.TrimSpace(string(data))
		if strings.HasPrefix(trim, "{") {
			return data
		}
	}
	return raw
}

func mapMemoryHTTPError(status int, raw []byte) error {
	msg := strings.TrimSpace(string(raw))
	var envelope struct {
		Error   string `json:"error"`
		Message string `json:"message"`
	}
	_ = json.Unmarshal(raw, &envelope)
	if envelope.Message != "" {
		msg = envelope.Message
	} else if envelope.Error != "" {
		msg = envelope.Error
	}
	if msg == "" {
		msg = fmt.Sprintf("WeKnora 记忆接口返回 HTTP %d", status)
	}
	return &MemoryAPIError{StatusCode: status, Message: msg}
}
