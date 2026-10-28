package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"qingzhi/enterprise-api/internal/auth"
	"qingzhi/enterprise-api/internal/response"
	"qingzhi/enterprise-api/internal/weknora"
)

// WeKnoraMemory serves workspace long-term memory configuration.
type WeKnoraMemory struct {
	Logger  *slog.Logger
	Service *weknora.MemoryService
}

func (h WeKnoraMemory) Register(group *gin.RouterGroup, enforce bool) {
	perm := func(permission string) gin.HandlerFunc {
		if !enforce {
			return func(c *gin.Context) { c.Next() }
		}
		return auth.RequirePermission(permission)
	}
	query := perm("model:query")
	manage := perm("model:manage")
	group.GET("/weknora/memory-config", query, h.getConfig)
	group.PUT("/weknora/memory-config", manage, h.updateConfig)
	group.GET("/weknora/memory/personal-status", query, h.personalStatus)
	group.POST("/weknora/external-user-token", manage, h.issueToken)
}

func (h WeKnoraMemory) getConfig(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	cfg, err := h.Service.GetConfig(c.Request.Context())
	h.write(c, cfg, err)
}

func (h WeKnoraMemory) updateConfig(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	var body weknora.MemoryConfig
	if !response.BindJSON(c, &body) {
		return
	}
	cfg, err := h.Service.UpdateConfig(c.Request.Context(), body)
	h.write(c, cfg, err)
}

func (h WeKnoraMemory) personalStatus(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	response.OK(c, h.Service.PersonalStatus())
}

func (h WeKnoraMemory) issueToken(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	principal, ok := auth.GetPrincipal(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录或令牌无效", nil)
		return
	}
	token, exp, identity, err := h.Service.IssueExternalUserToken(c.Request.Context(), principal)
	if err != nil {
		h.write(c, nil, err)
		return
	}
	response.OK(c, map[string]any{
		"expires_at":         exp.UTC(),
		"platform_user_id":   identity.PlatformUserID,
		"platform_tenant_id": identity.PlatformTenantID,
		"weknora_tenant_id":  identity.WeKnoraTenantID,
		"auth_method":        identity.AuthMethod,
		// Token is returned only to trusted admin tooling / server-side callers;
		// browsers must not store WeKnora API keys. Prefer attaching the token
		// only on server→WeKnora hops.
		"token": token,
	})
}

func (h WeKnoraMemory) ready(c *gin.Context) bool {
	if h.Service == nil {
		response.Fail(c, http.StatusServiceUnavailable, response.CodeNotReady, "服务暂不可用", nil)
		return false
	}
	return true
}

func (h WeKnoraMemory) write(c *gin.Context, data any, err error) {
	if err == nil {
		response.OK(c, data)
		return
	}
	msg := err.Error()
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "write_mode") || strings.Contains(lower, "max_items") ||
		strings.Contains(lower, "须在") || strings.Contains(lower, "未配置") {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, msg, nil)
		return
	}
	var apiErr *weknora.APIError
	if errors.As(err, &apiErr) {
		if apiErr.StatusCode == http.StatusForbidden || apiErr.StatusCode == http.StatusUnauthorized {
			response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "WeKnora 拒绝记忆配置访问，请检查 API Key 是否具备 manage_tenant_settings 或 full-access", nil)
			return
		}
	}
	if h.Logger != nil {
		h.Logger.Error("weknora memory request failed", slog.String("component", "weknora"), slog.String("error", safeWeknoraMemError(err)))
	}
	response.Fail(c, http.StatusBadGateway, response.CodeInternal, "无法访问 WeKnora 长期记忆配置", nil)
}

func safeWeknoraMemError(err error) string {
	text := err.Error()
	lower := strings.ToLower(text)
	if strings.Contains(lower, "api_key") || strings.Contains(lower, "sk-") || strings.Contains(lower, "hmac") {
		return "weknora memory request failed"
	}
	return text
}
