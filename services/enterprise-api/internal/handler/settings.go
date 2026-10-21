package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"qingzhi/enterprise-api/internal/auth"
	"qingzhi/enterprise-api/internal/response"
	"qingzhi/enterprise-api/internal/settings"
)

// Settings serves the system configuration page.
type Settings struct {
	Logger  *slog.Logger
	Service *settings.Service
}

func (h Settings) Register(group *gin.RouterGroup, enforce bool) {
	perm := func(permission string) gin.HandlerFunc {
		if !enforce {
			return func(c *gin.Context) { c.Next() }
		}
		return auth.RequirePermission(permission)
	}
	group.GET("/system/settings", perm("system:settings:query"), h.get)
	group.PUT("/system/settings", perm("system:settings:update"), h.update)
}

func (h Settings) get(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	view, err := h.Service.Get(c.Request.Context())
	h.write(c, view, err)
}

func (h Settings) update(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	var input struct {
		BaseURL string `json:"base_url"`
		APIKey  string `json:"api_key"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.InvalidParams(c, nil)
		return
	}
	principal, _ := auth.GetPrincipal(c)
	view, err := h.Service.Update(c.Request.Context(), settings.Update{
		BaseURL: input.BaseURL,
		APIKey:  input.APIKey,
		Updater: principal.Username,
	})
	h.write(c, view, err)
}

func (h Settings) ready(c *gin.Context) bool {
	if h.Service == nil {
		response.Fail(c, http.StatusServiceUnavailable, response.CodeNotReady, "服务暂不可用", nil)
		return false
	}
	return true
}

func (h Settings) write(c *gin.Context, data any, err error) {
	if err == nil {
		response.OK(c, data)
		return
	}
	var domain *settings.Error
	if errors.As(err, &domain) {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, domain.Message, nil)
		return
	}
	if h.Logger != nil {
		h.Logger.Error("settings request failed", slog.String("component", "settings"), slog.String("error", safeSettingsError(err)))
	}
	response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误", nil)
}

func safeSettingsError(err error) string {
	text := err.Error()
	if strings.Contains(strings.ToLower(text), "api_key") || strings.Contains(strings.ToLower(text), "sk-") {
		return "settings request failed"
	}
	return text
}
