package handler

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"qingzhi/enterprise-api/internal/auth"
	"qingzhi/enterprise-api/internal/chat"
	"qingzhi/enterprise-api/internal/knowledge"
	"qingzhi/enterprise-api/internal/response"
	"qingzhi/enterprise-api/internal/weknora"
)

// Chat serves workbench knowledge Q&A proxies.
type Chat struct {
	Logger  *slog.Logger
	Service *chat.Service
}

// Register mounts authenticated chat routes (no menu permission required).
func (h Chat) Register(group *gin.RouterGroup) {
	group.GET("/chat/knowledge-bases", h.listKnowledgeBases)
	group.GET("/chat/sessions", h.listSessions)
	group.POST("/chat/sessions", h.createSession)
	group.PUT("/chat/sessions/:id", h.renameSession)
	group.DELETE("/chat/sessions/:id", h.deleteSession)
	group.GET("/chat/sessions/:id/messages", h.loadMessages)
	group.POST("/chat/sessions/:id/ask", h.ask)
	group.POST("/chat/sessions/:id/stop", h.stop)
}

func (h Chat) listKnowledgeBases(c *gin.Context) {
	svc, principal, ok := h.ready(c)
	if !ok {
		return
	}
	items, err := svc.ListKnowledgeBases(c.Request.Context(), principal, c.Query("keyword"))
	h.write(c, items, err)
}

func (h Chat) listSessions(c *gin.Context) {
	svc, principal, ok := h.ready(c)
	if !ok {
		return
	}
	page, pageSize := weknora.ParsePageQuery(c.DefaultQuery("page", "1"), c.DefaultQuery("page_size", "20"))
	items, total, err := svc.ListSessions(c.Request.Context(), principal, page, pageSize)
	if err != nil {
		h.write(c, nil, err)
		return
	}
	response.OK(c, map[string]any{
		"items":     items,
		"page":      page,
		"page_size": pageSize,
		"total":     total,
	})
}

func (h Chat) createSession(c *gin.Context) {
	svc, principal, ok := h.ready(c)
	if !ok {
		return
	}
	var body struct {
		Title string `json:"title"`
	}
	if c.Request.ContentLength != 0 && !response.BindJSON(c, &body) {
		return
	}
	item, err := svc.CreateSession(c.Request.Context(), principal, body.Title)
	h.write(c, item, err)
}

func (h Chat) renameSession(c *gin.Context) {
	svc, principal, ok := h.ready(c)
	if !ok {
		return
	}
	var body struct {
		Title string `json:"title"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	item, err := svc.RenameSession(c.Request.Context(), principal, c.Param("id"), body.Title)
	h.write(c, item, err)
}

func (h Chat) deleteSession(c *gin.Context) {
	svc, principal, ok := h.ready(c)
	if !ok {
		return
	}
	err := svc.DeleteSession(c.Request.Context(), principal, c.Param("id"))
	h.write(c, map[string]any{"deleted": err == nil}, err)
}

func (h Chat) loadMessages(c *gin.Context) {
	svc, principal, ok := h.ready(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	var before *time.Time
	if raw := strings.TrimSpace(c.Query("before_time")); raw != "" {
		t, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			t, err = time.Parse(time.RFC3339, raw)
		}
		if err != nil {
			response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "before_time 格式不合法", nil)
			return
		}
		utc := t.UTC()
		before = &utc
	}
	items, err := svc.LoadMessages(c.Request.Context(), principal, c.Param("id"), limit, before)
	h.write(c, items, err)
}

func (h Chat) ask(c *gin.Context) {
	svc, principal, ok := h.ready(c)
	if !ok {
		return
	}
	var body struct {
		Query             string   `json:"query"`
		KnowledgeBaseIDs  []string `json:"knowledge_base_ids"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	ctx := weknora.WithRequestID(c.Request.Context(), response.RequestID(c))
	resp, err := svc.Ask(ctx, principal, c.Param("id"), body.Query, body.KnowledgeBaseIDs)
	if err != nil {
		h.write(c, nil, err)
		return
	}
	defer resp.Body.Close()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache, no-transform")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	flusher, canFlush := c.Writer.(http.Flusher)
	buf := make([]byte, 32*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := c.Writer.Write(buf[:n]); writeErr != nil {
				if h.Logger != nil {
					h.Logger.Info("chat sse client disconnected",
						slog.String("component", "chat"),
						slog.String("request_id", response.RequestID(c)),
					)
				}
				return
			}
			if canFlush {
				flusher.Flush()
			}
		}
		if readErr != nil {
			if !errors.Is(readErr, io.EOF) && h.Logger != nil {
				h.Logger.Warn("chat sse upstream closed",
					slog.String("component", "chat"),
					slog.String("request_id", response.RequestID(c)),
					slog.String("error", readErr.Error()),
				)
			}
			return
		}
	}
}

func (h Chat) stop(c *gin.Context) {
	svc, principal, ok := h.ready(c)
	if !ok {
		return
	}
	var body struct {
		MessageID string `json:"message_id"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	err := svc.Stop(c.Request.Context(), principal, c.Param("id"), body.MessageID)
	h.write(c, map[string]any{"stopped": err == nil}, err)
}

func (h Chat) ready(c *gin.Context) (*chat.Service, auth.Principal, bool) {
	principal, ok := auth.GetPrincipal(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录或令牌无效", nil)
		return nil, auth.Principal{}, false
	}
	if h.Service == nil {
		response.Fail(c, http.StatusServiceUnavailable, response.CodeNotReady, "对话服务暂不可用", nil)
		return nil, auth.Principal{}, false
	}
	if err := h.Service.Ready(); err != nil {
		var domain *chat.Error
		if errors.As(err, &domain) {
			response.Fail(c, http.StatusServiceUnavailable, response.CodeNotReady, domain.Message, nil)
		} else {
			response.Fail(c, http.StatusServiceUnavailable, response.CodeNotReady, "对话服务暂不可用", nil)
		}
		return nil, auth.Principal{}, false
	}
	return h.Service, principal, true
}

func (h Chat) write(c *gin.Context, data any, err error) {
	if err == nil {
		response.OK(c, data)
		return
	}
	var domain *chat.Error
	if errors.As(err, &domain) {
		switch {
		case errors.Is(err, chat.ErrUnauthorized):
			response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, domain.Message, nil)
		case errors.Is(err, chat.ErrForbidden):
			response.Fail(c, http.StatusForbidden, response.CodeForbidden, domain.Message, nil)
		case errors.Is(err, chat.ErrInvalid):
			response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, domain.Message, nil)
		case errors.Is(err, chat.ErrNotFound):
			response.Fail(c, http.StatusNotFound, response.CodeNotFound, domain.Message, nil)
		case errors.Is(err, chat.ErrNotReady):
			response.Fail(c, http.StatusServiceUnavailable, response.CodeNotReady, domain.Message, nil)
		default:
			response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, domain.Message, nil)
		}
		return
	}
	var knowledgeErr *knowledge.Error
	if errors.As(err, &knowledgeErr) {
		switch {
		case errors.Is(err, knowledge.ErrForbidden):
			response.Fail(c, http.StatusForbidden, response.CodeForbidden, knowledgeErr.Message, nil)
		case errors.Is(err, knowledge.ErrNotFound):
			response.Fail(c, http.StatusNotFound, response.CodeNotFound, knowledgeErr.Message, nil)
		case errors.Is(err, knowledge.ErrInvalid):
			response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, knowledgeErr.Message, nil)
		default:
			response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, knowledgeErr.Message, nil)
		}
		return
	}
	if h.Logger != nil {
		h.Logger.Error("chat request failed", slog.String("component", "chat"), slog.String("error", err.Error()))
	}
	response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误", nil)
}
