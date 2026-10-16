package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"qingzhi/enterprise-api/internal/auth"
	"qingzhi/enterprise-api/internal/response"
	"qingzhi/enterprise-api/internal/security"
)

// Security serves audit logs and the security policy.
type Security struct {
	Logger   *slog.Logger
	Database Pinger
	Service  *security.Service
	Timeout  time.Duration
}

// Register adds security routes.
func (h Security) Register(group *gin.RouterGroup, enforce bool) {
	perm := func(permission string) gin.HandlerFunc {
		if !enforce {
			return func(c *gin.Context) { c.Next() }
		}
		return auth.RequirePermission(permission)
	}
	group.GET("/security/login-logs", perm("system:login-log:query"), h.listLogins)
	group.GET("/security/operate-logs", perm("system:operate-log:query"), h.listOperates)
	group.GET("/security/permission-logs", perm("system:permission-log:query"), h.listPermissions)
	group.GET("/security/policy", perm("system:security-policy:query"), h.getPolicy)
	group.PATCH("/security/policy", perm("system:security-policy:update"), h.updatePolicy)
}

func (h Security) listLogins(c *gin.Context) {
	if !h.available(c) {
		return
	}
	query, ok := securityListQuery(c)
	if !ok {
		return
	}
	if raw := strings.TrimSpace(c.Query("result")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil {
			response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "登录结果不合法", nil)
			return
		}
		query.Result = &value
	}
	page, err := h.Service.ListLogins(c.Request.Context(), query)
	h.write(c, page, err)
}

func (h Security) listOperates(c *gin.Context) {
	if !h.available(c) {
		return
	}
	query, ok := securityListQuery(c)
	if !ok {
		return
	}
	page, err := h.Service.ListOperates(c.Request.Context(), query)
	h.write(c, page, err)
}

func (h Security) listPermissions(c *gin.Context) {
	if !h.available(c) {
		return
	}
	query, ok := securityListQuery(c)
	if !ok {
		return
	}
	page, err := h.Service.ListPermissions(c.Request.Context(), query)
	h.write(c, page, err)
}

func (h Security) getPolicy(c *gin.Context) {
	if !h.available(c) {
		return
	}
	item, err := h.Service.Policy(c.Request.Context())
	h.write(c, item, err)
}

func (h Security) updatePolicy(c *gin.Context) {
	if !h.available(c) {
		return
	}
	var body security.Policy
	if !response.BindJSON(c, &body) {
		return
	}
	item, err := h.Service.UpdatePolicy(c.Request.Context(), body, securityActor(c))
	h.write(c, item, err)
}

func securityActor(c *gin.Context) string {
	principal, ok := auth.GetPrincipal(c)
	if !ok {
		return ""
	}
	return principal.Username
}

func (h Security) available(c *gin.Context) bool {
	if h.Service == nil {
		response.Fail(c, http.StatusServiceUnavailable, response.CodeNotReady, "服务暂不可用", nil)
		return false
	}
	if h.Database == nil {
		return true
	}
	timeout := h.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()
	if err := h.Database.Ping(ctx); err != nil {
		if h.Logger != nil {
			h.Logger.Warn("security database unavailable", slog.String("component", "app"), slog.String("error", err.Error()))
		}
		response.Fail(c, http.StatusServiceUnavailable, response.CodeNotReady, "服务暂不可用", nil)
		return false
	}
	return true
}

func (h Security) write(c *gin.Context, data any, err error) {
	if err == nil {
		response.OK(c, data)
		return
	}
	var domain *security.Error
	if errors.As(err, &domain) {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, domain.Message, nil)
		return
	}
	if h.Logger != nil {
		h.Logger.Error("security request failed", slog.String("component", "app"), slog.String("error", err.Error()))
	}
	response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误", nil)
}

func securityListQuery(c *gin.Context) (security.ListQuery, bool) {
	page, err := securityOptionalInt(c.Query("page"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "分页参数不合法", nil)
		return security.ListQuery{}, false
	}
	pageSize, err := securityOptionalInt(c.Query("page_size"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "分页参数不合法", nil)
		return security.ListQuery{}, false
	}
	keyword := strings.TrimSpace(c.Query("keyword"))
	if len(keyword) > 64 {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "请求参数不合法", nil)
		return security.ListQuery{}, false
	}
	return security.ListQuery{Page: page, PageSize: pageSize, Keyword: keyword}, true
}

func securityOptionalInt(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	return strconv.Atoi(raw)
}
