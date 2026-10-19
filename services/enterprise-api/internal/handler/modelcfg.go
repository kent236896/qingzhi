package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"qingzhi/enterprise-api/internal/auth"
	"qingzhi/enterprise-api/internal/modelcfg"
	"qingzhi/enterprise-api/internal/response"
	"qingzhi/enterprise-api/internal/weknora"
)

// ModelConfig serves WeKnora model administration and local scene bindings.
type ModelConfig struct {
	Logger  *slog.Logger
	Service *modelcfg.Service
}

func (h ModelConfig) Register(group *gin.RouterGroup, enforce bool) {
	perm := func(permission string) gin.HandlerFunc {
		if !enforce {
			return func(c *gin.Context) { c.Next() }
		}
		return auth.RequirePermission(permission)
	}
	query := perm("model:query")
	manage := perm("model:manage")
	group.GET("/weknora/model-providers", query, h.providers)
	group.GET("/weknora/model-providers/resolve", query, h.resolve)
	group.GET("/weknora/model-bindings", query, h.bindings)
	group.PUT("/weknora/model-bindings/:role", manage, h.saveBinding)
	group.GET("/weknora/models", query, h.list)
	group.POST("/weknora/models", manage, h.create)
	group.POST("/weknora/models/test", manage, h.test)
	group.GET("/weknora/ollama/status", query, h.ollamaStatus)
	group.GET("/weknora/ollama/models", query, h.ollamaModels)
	group.POST("/weknora/ollama/models/download", manage, h.ollamaDownload)
	group.GET("/weknora/ollama/download/progress/:taskId", query, h.ollamaProgress)
	group.GET("/weknora/models/:id", query, h.get)
	group.PUT("/weknora/models/:id", manage, h.update)
	group.DELETE("/weknora/models/:id", manage, h.delete)
	group.POST("/weknora/models/:id/copy", manage, h.copy)
	group.PUT("/weknora/models/:id/credentials", manage, h.credentials)
	group.DELETE("/weknora/models/:id/credentials/:field", manage, h.clearCredential)
	group.POST("/weknora/models/:id/debug", manage, h.debug)
}

func (h ModelConfig) providers(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	items, err := h.Service.Providers(h.ctx(c), c.Query("type"), c.Query("icons") == "1")
	h.write(c, items, err)
}

func (h ModelConfig) resolve(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	data, err := h.Service.Resolve(h.ctx(c), c.Query("provider"), c.Query("model"), c.Query("base_url"), c.Query("type"))
	h.write(c, data, err)
}

func (h ModelConfig) list(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	items, err := h.Service.List(h.ctx(c), c.Query("type"), c.Query("keyword"))
	h.write(c, items, err)
}

func (h ModelConfig) get(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	item, err := h.Service.Get(h.ctx(c), c.Param("id"))
	h.write(c, item, err)
}

func (h ModelConfig) create(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	var input modelcfg.SaveInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.InvalidParams(c, nil)
		return
	}
	item, err := h.Service.Create(h.ctx(c), input)
	h.write(c, item, err)
}

func (h ModelConfig) update(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	var input modelcfg.SaveInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.InvalidParams(c, nil)
		return
	}
	item, err := h.Service.Update(h.ctx(c), c.Param("id"), input)
	h.write(c, item, err)
}

func (h ModelConfig) delete(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	h.write(c, nil, h.Service.Delete(h.ctx(c), c.Param("id")))
}

func (h ModelConfig) copy(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	var input struct {
		DisplayName string `json:"display_name"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.InvalidParams(c, nil)
		return
	}
	item, err := h.Service.Copy(h.ctx(c), c.Param("id"), input.DisplayName)
	h.write(c, item, err)
}

func (h ModelConfig) credentials(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	var input struct {
		APIKey    string `json:"api_key"`
		AppSecret string `json:"app_secret"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.InvalidParams(c, nil)
		return
	}
	item, err := h.Service.PutCredentials(h.ctx(c), c.Param("id"), input.APIKey, input.AppSecret)
	h.write(c, item, err)
}

func (h ModelConfig) clearCredential(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	h.write(c, nil, h.Service.DeleteCredential(h.ctx(c), c.Param("id"), c.Param("field")))
}

func (h ModelConfig) test(c *gin.Context) {
	if !h.ready(c) || !h.allow(c, "test") {
		return
	}
	var input modelcfg.TestInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.InvalidParams(c, nil)
		return
	}
	item, err := h.Service.Test(h.ctx(c), input)
	h.write(c, item, err)
}

func (h ModelConfig) debug(c *gin.Context) {
	if !h.ready(c) || !h.allow(c, "debug") {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 32<<20+4096)
	form, err := c.MultipartForm()
	if err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "调试请求不是合法的表单", nil)
		return
	}
	input, options, documents, fileName, file, err := modelcfg.ReadDebugParts(form)
	if err != nil {
		h.write(c, nil, err)
		return
	}
	contentType, body, err := modelcfg.BuildDebugForm(input, options, documents, fileName, file)
	if err != nil {
		h.write(c, nil, err)
		return
	}
	item, err := h.Service.Debug(h.ctx(c), c.Param("id"), contentType, body)
	h.write(c, item, err)
}

func (h ModelConfig) bindings(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	items, err := h.Service.ListBindings(h.ctx(c))
	h.write(c, items, err)
}

func (h ModelConfig) saveBinding(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	var input struct {
		ModelID string `json:"model_id"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.InvalidParams(c, nil)
		return
	}
	principal, _ := auth.GetPrincipal(c)
	item, err := h.Service.SaveBinding(h.ctx(c), c.Param("role"), input.ModelID, principal.Username)
	h.write(c, item, err)
}

func (h ModelConfig) ollamaStatus(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	item, err := h.Service.OllamaStatus(h.ctx(c))
	h.write(c, item, err)
}

func (h ModelConfig) ollamaModels(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	items, err := h.Service.OllamaModels(h.ctx(c))
	h.write(c, items, err)
}

func (h ModelConfig) ollamaDownload(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	var input struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.InvalidParams(c, nil)
		return
	}
	item, err := h.Service.OllamaDownload(h.ctx(c), input.Name)
	h.write(c, item, err)
}

func (h ModelConfig) ollamaProgress(c *gin.Context) {
	if !h.ready(c) {
		return
	}
	item, err := h.Service.OllamaProgress(h.ctx(c), c.Param("taskId"))
	h.write(c, item, err)
}

func (h ModelConfig) ctx(c *gin.Context) context.Context {
	return weknora.WithRequestID(c.Request.Context(), response.RequestID(c))
}

func (h ModelConfig) ready(c *gin.Context) bool {
	if h.Service == nil {
		response.Fail(c, http.StatusServiceUnavailable, response.CodeNotReady, "服务暂不可用", nil)
		return false
	}
	return true
}

func (h ModelConfig) allow(c *gin.Context, action string) bool {
	principal, _ := auth.GetPrincipal(c)
	userID := principal.UserID
	if userID == "" {
		userID = c.ClientIP()
	}
	if h.Service.Allow(userID, action) {
		return true
	}
	response.Fail(c, http.StatusTooManyRequests, response.CodeTooManyRequests, "操作过于频繁，请稍后再试", nil)
	return false
}

func (h ModelConfig) write(c *gin.Context, data any, err error) {
	if err == nil {
		response.OK(c, data)
		return
	}
	var domain *modelcfg.Error
	if errors.As(err, &domain) {
		switch {
		case errors.Is(err, modelcfg.ErrForbidden):
			response.Fail(c, http.StatusForbidden, response.CodeForbidden, domain.Message, domain.Data)
		case errors.Is(err, modelcfg.ErrNotFound):
			response.Fail(c, http.StatusNotFound, response.CodeNotFound, domain.Message, domain.Data)
		case errors.Is(err, modelcfg.ErrConflict):
			response.Fail(c, http.StatusConflict, response.CodeConflict, domain.Message, domain.Data)
		case errors.Is(err, modelcfg.ErrNotReady):
			response.Fail(c, http.StatusServiceUnavailable, response.CodeNotReady, domain.Message, nil)
		case errors.Is(err, modelcfg.ErrTimeout):
			response.Fail(c, http.StatusGatewayTimeout, response.CodeTimeout, domain.Message, nil)
		default:
			response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, domain.Message, domain.Data)
		}
		return
	}
	if h.Logger != nil {
		h.Logger.Error("model request failed", slog.String("component", "model"), slog.String("error", safeError(err)))
	}
	response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误", nil)
}

func safeError(err error) string {
	text := err.Error()
	if strings.Contains(strings.ToLower(text), "api_key") || strings.Contains(strings.ToLower(text), "sk-") {
		return "model request failed"
	}
	return text
}
