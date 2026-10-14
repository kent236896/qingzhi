// Package logger writes structured JSON logs to standard output.
package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"
	"time"
)

// New builds a JSON logger. component distinguishes access logs from application logs.
func New(level, service string) *slog.Logger {
	return NewWriter(os.Stdout, level, service)
}

// NewWriter builds a logger that writes to w. Tests use it to capture output.
func NewWriter(w io.Writer, level, service string) *slog.Logger {
	slogLevel := parseLevel(level)
	handler := slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level:     slogLevel,
		AddSource: slogLevel == slog.LevelDebug,
		ReplaceAttr: func(_ []string, attr slog.Attr) slog.Attr {
			if attr.Key == slog.TimeKey && attr.Value.Kind() == slog.KindTime {
				attr.Value = slog.StringValue(attr.Value.Time().UTC().Format(time.RFC3339Nano))
			}
			return attr
		},
	})
	log := slog.New(handler).With(slog.String("service", service))
	if w == os.Stdout {
		slog.SetDefault(log)
	}
	return log
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
