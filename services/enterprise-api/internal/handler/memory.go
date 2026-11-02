package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"qingzhi/enterprise-api/internal/auth"
	"qingzhi/enterprise-api/internal/response"
	"qingzhi/enterprise-api/internal/weknora"
)

// Memory serves workbench-facing personal memory APIs.
type Memory struct {
	Logger  *slog.Logger
	Service *weknora.MemoryService
}

func (h Memory) Register(group *gin.RouterGroup) {
	group.GET("/memory/status", h.status)
	group.GET("/memory/settings", h.getSettings)
	group.PUT("/memory/settings", h.putSettings)
	group.GET("/memory/items", h.listItems)
	group.POST("/memory/items", h.createItem)
	group.PUT("/memory/items/:id", h.updateItem)
	group.DELETE("/memory/items/:id", h.deleteItem)
	group.POST("/memory/items/:id/confirm", h.confirmItem)
	group.POST("/memory/items/:id/reject", h.rejectItem)
}

func (h Memory) status(c *gin.Context) {
	principal, ok := h.principal(c)
	if !ok {
		return
	}
	_ = principal
	if h.Service == nil {
		response.Fail(c, http.StatusServiceUnavailable, response.CodeNotReady, "记忆服务暂不可用", nil)
		return
	}
	response.OK(c, h.Service.WorkbenchStatus(c.Request.Context()))
}

func (h Memory) getSettings(c *gin.Context) {
	svc, principal, ok := h.ready(c)
	if !ok {
		return
	}
	item, err := svc.GetPersonalSettings(c.Request.Context(), principal)
	h.write(c, item, err)
}

func (h Memory) putSettings(c *gin.Context) {
	svc, principal, ok := h.ready(c)
	if !ok {
		return
	}
	var body struct {
		Enabled     *bool `json:"enabled"`
		UserEnabled *bool `json:"user_enabled"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	enabled := false
	switch {
	case body.UserEnabled != nil:
		enabled = *body.UserEnabled
	case body.Enabled != nil:
		enabled = *body.Enabled
	default:
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "请提供 enabled 或 user_enabled", nil)
		return
	}
	item, err := svc.UpdatePersonalSettings(c.Request.Context(), principal, enabled)
	h.write(c, item, err)
}

func (h Memory) listItems(c *gin.Context) {
	svc, principal, ok := h.ready(c)
	if !ok {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	kind := c.Query("type")
	if kind == "" {
		kind = c.Query("kind")
	}
	item, err := svc.ListPersonalItems(c.Request.Context(), principal, page, pageSize, kind, c.Query("status"))
	h.write(c, item, err)
}

func (h Memory) createItem(c *gin.Context) {
	svc, principal, ok := h.ready(c)
	if !ok {
		return
	}
	var body struct {
		Type       string `json:"type"`
		Kind       string `json:"kind"`
		Content    string `json:"content"`
		Importance int    `json:"importance"`
		Priority   int    `json:"priority"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	kind := body.Kind
	if kind == "" {
		kind = body.Type
	}
	importance := body.Importance
	if importance == 0 {
		importance = body.Priority
	}
	item, err := svc.CreatePersonalItem(c.Request.Context(), principal, kind, body.Content, importance)
	h.write(c, item, err)
}

func (h Memory) updateItem(c *gin.Context) {
	svc, principal, ok := h.ready(c)
	if !ok {
		return
	}
	var body struct {
		Type       string `json:"type"`
		Kind       string `json:"kind"`
		Content    string `json:"content"`
		Importance int    `json:"importance"`
		Priority   int    `json:"priority"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	kind := body.Kind
	if kind == "" {
		kind = body.Type
	}
	importance := body.Importance
	if importance == 0 {
		importance = body.Priority
	}
	item, err := svc.UpdatePersonalItem(c.Request.Context(), principal, c.Param("id"), kind, body.Content, importance)
	h.write(c, item, err)
}

func (h Memory) deleteItem(c *gin.Context) {
	svc, principal, ok := h.ready(c)
	if !ok {
		return
	}
	err := svc.DeletePersonalItem(c.Request.Context(), principal, c.Param("id"))
	if err != nil {
		h.write(c, nil, err)
		return
	}
	response.OK(c, map[string]any{"deleted": true})
}

func (h Memory) confirmItem(c *gin.Context) {
	svc, principal, ok := h.ready(c)
	if !ok {
		return
	}
	err := svc.ConfirmPersonalItem(c.Request.Context(), principal, c.Param("id"))
	if err != nil {
		h.write(c, nil, err)
		return
	}
	response.OK(c, map[string]any{"confirmed": true})
}

func (h Memory) rejectItem(c *gin.Context) {
	svc, principal, ok := h.ready(c)
	if !ok {
		return
	}
	err := svc.RejectPersonalItem(c.Request.Context(), principal, c.Param("id"))
	if err != nil {
		h.write(c, nil, err)
		return
	}
	response.OK(c, map[string]any{"rejected": true})
}

func (h Memory) principal(c *gin.Context) (auth.Principal, bool) {
	principal, ok := auth.GetPrincipal(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录或令牌无效", nil)
		return auth.Principal{}, false
	}
	return principal, true
}

func (h Memory) ready(c *gin.Context) (*weknora.MemoryService, auth.Principal, bool) {
	principal, ok := h.principal(c)
	if !ok {
		return nil, auth.Principal{}, false
	}
	if h.Service == nil {
		response.Fail(c, http.StatusServiceUnavailable, response.CodeNotReady, "记忆服务暂不可用", nil)
		return nil, auth.Principal{}, false
	}
	if !weknora.PersonalMemoryIsolation().Ready {
		response.Fail(c, http.StatusServiceUnavailable, response.CodeNotReady, weknora.PersonalMemoryIsolation().Reason, nil)
		return nil, auth.Principal{}, false
	}
	return h.Service, principal, true
}

func (h Memory) write(c *gin.Context, data any, err error) {
	if err == nil {
		response.OK(c, data)
		return
	}
	status, message := weknora.MapPersonalMemoryError(err)
	code := response.CodeInternal
	switch status {
	case http.StatusBadRequest:
		code = response.CodeInvalidParams
	case http.StatusUnauthorized:
		code = response.CodeUnauthorized
	case http.StatusForbidden:
		code = response.CodeForbidden
	case http.StatusNotFound:
		code = response.CodeNotFound
	case http.StatusConflict:
		code = response.CodeConflict
	case http.StatusServiceUnavailable:
		code = response.CodeNotReady
	default:
		if h.Logger != nil {
			h.Logger.Error("memory request failed", slog.String("component", "memory"), slog.String("error", err.Error()))
		}
		code = response.CodeInternal
		if status == http.StatusBadGateway {
			message = "记忆服务暂时不可用"
		}
	}
	response.Fail(c, status, code, message, nil)
}
