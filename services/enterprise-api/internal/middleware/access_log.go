package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"qingzhi/enterprise-api/internal/response"
)

// AccessLog writes one structured access log after the request finishes.
// It records the path without the query string and never reads the body.
func AccessLog(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		path := ""
		method := ""
		if c.Request != nil && c.Request.URL != nil {
			path = c.Request.URL.Path
			method = c.Request.Method
		}
		log.Info("http request",
			slog.String("component", "access"),
			slog.String("request_id", response.RequestID(c)),
			slog.String("method", method),
			slog.String("path", path),
			slog.Int("status", c.Writer.Status()),
			slog.Float64("duration_ms", float64(time.Since(start).Microseconds())/1000),
			slog.String("client_ip", c.ClientIP()),
			slog.Int("bytes", c.Writer.Size()),
		)
	}
}
