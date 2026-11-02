package weknora

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// ArtifactItem is one library row from WeKnora.
type ArtifactItem struct {
	SessionID    string `json:"session_id"`
	SessionTitle string `json:"session_title"`
	MessageID    string `json:"message_id"`
	Index        int    `json:"index"`
	Handle       string `json:"handle,omitempty"`
	FileName     string `json:"file_name"`
	FileType     string `json:"file_type"`
	FileSize     int64  `json:"file_size"`
	SourcePath   string `json:"source_path"`
	CreatedAt    string `json:"created_at"`
	VersionCount int    `json:"version_count"`
}

// ArtifactPage is a paginated artifact library response.
type ArtifactPage struct {
	Items    []ArtifactItem `json:"items"`
	Total    int            `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}

// ListArtifactsAdmin lists the artifact library with admin JWT (ops view).
func (c *HTTPClient) ListArtifactsAdmin(ctx context.Context, keyword string, page, pageSize int) (ArtifactPage, error) {
	return c.listArtifacts(ctx, true, keyword, page, pageSize)
}

// ListArtifactsUser lists artifacts for the terminal user in ctx (External User Token).
func (c *HTTPClient) ListArtifactsUser(ctx context.Context, keyword string, page, pageSize int) (ArtifactPage, error) {
	if ExternalUserTokenFrom(ctx) == "" {
		return ArtifactPage{}, fmt.Errorf("缺少终端用户身份令牌")
	}
	return c.listArtifacts(ctx, false, keyword, page, pageSize)
}

func (c *HTTPClient) listArtifacts(ctx context.Context, asAdmin bool, keyword string, page, pageSize int) (ArtifactPage, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	q := url.Values{}
	q.Set("page", strconv.Itoa(page))
	q.Set("page_size", strconv.Itoa(pageSize))
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		q.Set("keyword", keyword)
	}
	path := "/api/v1/artifacts?" + q.Encode()
	var status int
	var raw []byte
	var err error
	if asAdmin {
		status, raw, err = c.ExchangeAdmin(ctx, http.MethodGet, path, nil)
	} else {
		status, raw, err = c.Exchange(ctx, http.MethodGet, path, nil)
	}
	if err != nil {
		return ArtifactPage{}, err
	}
	if status != http.StatusOK {
		return ArtifactPage{}, fmt.Errorf("读取产物失败（HTTP %d）", status)
	}
	var envelope struct {
		Success  bool           `json:"success"`
		Data     []ArtifactItem `json:"data"`
		Total    int            `json:"total"`
		Page     int            `json:"page"`
		PageSize int            `json:"page_size"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return ArtifactPage{}, err
	}
	items := envelope.Data
	if items == nil {
		items = []ArtifactItem{}
	}
	total := envelope.Total
	if total == 0 {
		total = len(items)
	}
	return ArtifactPage{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

// DeleteArtifactAdmin deletes one library file (and versions) via admin JWT.
func (c *HTTPClient) DeleteArtifactAdmin(ctx context.Context, sessionID, messageID string, index int) error {
	return c.deleteArtifact(ctx, true, sessionID, messageID, index)
}

// DeleteArtifactUser deletes one library file as the terminal user.
func (c *HTTPClient) DeleteArtifactUser(ctx context.Context, sessionID, messageID string, index int) error {
	if ExternalUserTokenFrom(ctx) == "" {
		return fmt.Errorf("缺少终端用户身份令牌")
	}
	return c.deleteArtifact(ctx, false, sessionID, messageID, index)
}

func (c *HTTPClient) deleteArtifact(ctx context.Context, asAdmin bool, sessionID, messageID string, index int) error {
	sessionID = strings.TrimSpace(sessionID)
	messageID = strings.TrimSpace(messageID)
	if sessionID == "" || messageID == "" {
		return fmt.Errorf("缺少产物定位信息")
	}
	q := url.Values{}
	q.Set("session_id", sessionID)
	q.Set("message_id", messageID)
	q.Set("index", strconv.Itoa(index))
	path := "/api/v1/artifacts?" + q.Encode()
	var status int
	var err error
	if asAdmin {
		status, _, err = c.ExchangeAdmin(ctx, http.MethodDelete, path, nil)
	} else {
		status, _, err = c.Exchange(ctx, http.MethodDelete, path, nil)
	}
	if err != nil {
		return err
	}
	if status == http.StatusNotFound {
		return ErrNotFound
	}
	if status != http.StatusOK && status != http.StatusNoContent {
		return fmt.Errorf("删除产物失败（HTTP %d）", status)
	}
	return nil
}
