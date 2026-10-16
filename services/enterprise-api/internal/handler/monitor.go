package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"qingzhi/enterprise-api/internal/auth"
	"qingzhi/enterprise-api/internal/monitor"
	"qingzhi/enterprise-api/internal/response"
)

// Monitor serves live service, traffic, resource, and alert data.
type Monitor struct {
	Logger  *slog.Logger
	Service *monitor.Service
}

// Register adds monitoring routes.
func (h Monitor) Register(group *gin.RouterGroup, enforce bool) {
	perm := func(permission string) gin.HandlerFunc {
		if !enforce {
			return func(c *gin.Context) { c.Next() }
		}
		return auth.RequirePermission(permission)
	}
	group.GET("/monitor/overview", perm("system:monitor:query"), h.overview)
	group.GET("/monitor/alerts", perm("system:monitor:query"), h.alerts)
}

func (h Monitor) overview(c *gin.Context) {
	if h.Service == nil {
		response.Fail(c, http.StatusServiceUnavailable, response.CodeNotReady, "服务暂不可用", nil)
		return
	}
	response.OK(c, h.Service.Overview(c.Request.Context()))
}

func (h Monitor) alerts(c *gin.Context) {
	if h.Service == nil {
		response.Fail(c, http.StatusServiceUnavailable, response.CodeNotReady, "服务暂不可用", nil)
		return
	}
	page, err := monitorOptionalInt(c.Query("page"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "分页参数不合法", nil)
		return
	}
	pageSize, err := monitorOptionalInt(c.Query("page_size"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "分页参数不合法", nil)
		return
	}
	result, err := h.Service.ListAlerts(c.Request.Context(), page, pageSize, strings.TrimSpace(c.Query("status")))
	if err != nil {
		var domain *monitor.Error
		if errors.As(err, &domain) {
			response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, domain.Message, nil)
			return
		}
		if h.Logger != nil {
			h.Logger.Error("monitor request failed", slog.String("component", "app"), slog.String("error", err.Error()))
		}
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误", nil)
		return
	}
	response.OK(c, result)
}

func monitorOptionalInt(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	return strconv.Atoi(raw)
}
