// Package handler implements HTTP handlers.
package handler

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"qingzhi/enterprise-api/internal/response"
)

// Pinger checks one runtime dependency.
type Pinger interface {
	Ping(ctx context.Context) error
}

// Health serves process and dependency health checks.
type Health struct {
	Logger   *slog.Logger
	Database Pinger
	Redis    Pinger
	Timeout  time.Duration
}

type liveData struct {
	Status string `json:"status"`
}

type readyChecks struct {
	Database string `json:"database"`
	Redis    string `json:"redis"`
}

type readyData struct {
	Status string      `json:"status"`
	Checks readyChecks `json:"checks"`
}

// Live reports that the process can answer HTTP requests.
func (h Health) Live(c *gin.Context) {
	response.OK(c, liveData{Status: "live"})
}

// Ready reports whether PostgreSQL and Redis can be reached.
// The response never includes connection strings or credentials.
func (h Health) Ready(c *gin.Context) {
	timeout := h.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()

	checks := readyChecks{
		Database: pingState(ctx, h.Logger, "database", h.Database),
		Redis:    pingState(ctx, h.Logger, "redis", h.Redis),
	}
	data := readyData{Status: "ready", Checks: checks}
	if checks.Database != "up" || checks.Redis != "up" {
		data.Status = "not_ready"
		response.Fail(c, http.StatusServiceUnavailable, response.CodeNotReady, "服务暂不可用", data)
		return
	}
	response.OK(c, data)
}

func pingState(ctx context.Context, log *slog.Logger, name string, pinger Pinger) string {
	if pinger == nil {
		return "down"
	}
	if err := pinger.Ping(ctx); err != nil {
		if log != nil {
			log.Debug("dependency check failed",
				slog.String("component", "app"),
				slog.String("dependency", name),
				slog.String("error", err.Error()),
			)
		}
		return "down"
	}
	return "up"
}
