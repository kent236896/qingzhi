package security

import (
	"log/slog"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"qingzhi/enterprise-api/internal/auth"
	"qingzhi/enterprise-api/internal/response"
)

// OperateAudit records authenticated write requests after the handler finishes.
// It stores the path without the query string and never reads the body.
func OperateAudit(svc *Service, log *slog.Logger) gin.HandlerFunc {
	if svc == nil {
		return func(c *gin.Context) { c.Next() }
	}
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		if c.Request == nil || !isWriteMethod(c.Request.Method) {
			return
		}
		principal, ok := auth.GetPrincipal(c)
		if !ok {
			return
		}
		path := ""
		if c.Request.URL != nil {
			path = c.Request.URL.Path
		}
		agent := ""
		if c.Request != nil {
			agent = c.Request.UserAgent()
		}
		item := OperateLog{
			UserID:        principal.UserID,
			Username:      principal.Username,
			TenantID:      principal.TenantID,
			Module:        moduleOf(path),
			Action:        actionOf(c.Request.Method, path),
			RequestMethod: c.Request.Method,
			RequestURL:    path,
			IP:            c.ClientIP(),
			UserAgent:     agent,
			StatusCode:    c.Writer.Status(),
			DurationMS:    int(time.Since(start).Milliseconds()),
			RequestID:     response.RequestID(c),
		}
		if err := svc.RecordOperate(c.Request.Context(), item); err != nil && log != nil {
			log.Warn("operate log failed", slog.String("component", "app"), slog.String("error", err.Error()))
		}
	}
}

func isWriteMethod(method string) bool {
	switch method {
	case "POST", "PUT", "PATCH", "DELETE":
		return true
	default:
		return false
	}
}

func moduleOf(path string) string {
	switch {
	case strings.Contains(path, "/security/policy"):
		return "安全策略"
	case strings.Contains(path, "/dict"):
		return "字典"
	case strings.Contains(path, "/roles"):
		return "角色"
	case strings.Contains(path, "/menus"):
		return "菜单"
	case strings.Contains(path, "/posts"):
		return "岗位"
	case strings.Contains(path, "/organizations"):
		return "组织"
	case strings.Contains(path, "/members"):
		return "成员"
	case strings.Contains(path, "/users"):
		return "用户"
	case strings.Contains(path, "/tenants"):
		return "租户"
	case strings.Contains(path, "/weknora"):
		return "模型配置"
	default:
		return "其他"
	}
}

func actionOf(method, path string) string {
	switch {
	case strings.Contains(path, "/menus") && method == "PUT":
		return "分配菜单"
	case strings.Contains(path, "/users") && strings.Contains(path, "/roles") && method == "PUT":
		return "分配用户"
	case strings.Contains(path, "/depts"):
		return "分配部门"
	case strings.Contains(path, "/debug"):
		return "调试"
	case strings.Contains(path, "/models/test"):
		return "测试"
	case strings.Contains(path, "/copy"):
		return "复制"
	case strings.Contains(path, "/credentials"):
		return "密钥"
	case method == "POST":
		return "新增"
	case method == "PATCH" || method == "PUT":
		return "修改"
	case method == "DELETE":
		return "删除"
	default:
		return method
	}
}
