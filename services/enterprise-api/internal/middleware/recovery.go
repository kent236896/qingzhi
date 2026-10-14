package middleware

import (
	"log/slog"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"qingzhi/enterprise-api/internal/response"
)

// Recovery logs a panic once and returns a safe error envelope.
func Recovery(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}
			method := ""
			path := ""
			if c.Request != nil && c.Request.URL != nil {
				method = c.Request.Method
				path = c.Request.URL.Path
			}
			log.Error("panic recovered",
				slog.String("component", "app"),
				slog.String("request_id", response.RequestID(c)),
				slog.String("method", method),
				slog.String("path", path),
				slog.Any("error", recovered),
				slog.String("stack", string(debug.Stack())),
			)
			if !c.Writer.Written() {
				response.Fail(c, 500, response.CodeInternal, "服务器内部错误", nil)
			}
			c.Abort()
		}()
		c.Next()
	}
}
