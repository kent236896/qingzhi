package monitor

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"qingzhi/enterprise-api/internal/config"
)

// AlertStore persists firing and resolved alerts.
type AlertStore interface {
	Sync(ctx context.Context, desired []Draft) ([]Alert, error)
	List(ctx context.Context, page, pageSize int, status string) (Page[Alert], error)
}

// Pinger checks one runtime dependency.
type Pinger interface {
	Ping(ctx context.Context) error
}

func buildAlerts(cfg config.MonitorConfig, state ServiceState, traffic Traffic, resources Resources, resourceErr bool) []Draft {
	var drafts []Draft
	if state.Database != "up" {
		drafts = append(drafts, Draft{Kind: "service", Target: "database", Level: "critical", Title: "数据库不可用", Message: "PostgreSQL 连通性检查失败"})
	}
	if state.Redis != "up" {
		drafts = append(drafts, Draft{Kind: "service", Target: "redis", Level: "critical", Title: "Redis 不可用", Message: "Redis 连通性检查失败"})
	}
	if cfg.ErrorCount > 0 && traffic.ServerErrors >= cfg.ErrorCount {
		drafts = append(drafts, Draft{
			Kind: "traffic", Target: "errors", Level: "warning", Title: "接口错误过多",
			Message: fmt.Sprintf("窗口内服务端错误 %d 次，阈值 %d", traffic.ServerErrors, cfg.ErrorCount),
		})
	}
	if traffic.Requests > 0 && traffic.P95MS >= int(cfg.P95/time.Millisecond) {
		drafts = append(drafts, Draft{
			Kind: "traffic", Target: "latency", Level: "warning", Title: "响应时间过长",
			Message: fmt.Sprintf("P95 %d 毫秒，阈值 %d 毫秒", traffic.P95MS, cfg.P95/time.Millisecond),
		})
	}
	if cfg.TimeoutCount > 0 && traffic.Timeouts >= cfg.TimeoutCount {
		drafts = append(drafts, Draft{
			Kind: "traffic", Target: "timeout", Level: "warning", Title: "请求超时",
			Message: fmt.Sprintf("窗口内超时 %d 次，阈值 %d", traffic.Timeouts, cfg.TimeoutCount),
		})
	}
	if resourceErr {
		drafts = append(drafts, Draft{Kind: "task", Target: "collector", Level: "warning", Title: "资源采集失败", Message: "无法读取 CPU、内存或磁盘"})
	}
	if resources.Available {
		if cfg.CPUPercent > 0 && resources.CPUPercent >= float64(cfg.CPUPercent) {
			drafts = append(drafts, capacityDraft("cpu", "CPU 使用率过高", resources.CPUPercent, cfg.CPUPercent))
		}
		if cfg.MemoryPercent > 0 && resources.MemoryPercent >= float64(cfg.MemoryPercent) {
			drafts = append(drafts, capacityDraft("memory", "内存使用率过高", resources.MemoryPercent, cfg.MemoryPercent))
		}
		if cfg.DiskPercent > 0 && resources.DiskPercent >= float64(cfg.DiskPercent) {
			drafts = append(drafts, capacityDraft("disk", "磁盘使用率过高", resources.DiskPercent, cfg.DiskPercent))
		}
	}
	return drafts
}

func capacityDraft(target, title string, current float64, threshold int) Draft {
	return Draft{
		Kind: "capacity", Target: target, Level: "warning", Title: title,
		Message: fmt.Sprintf("当前 %.1f%%，阈值 %d%%", current, threshold),
	}
}

func draftsAsAlerts(drafts []Draft, now time.Time) []Alert {
	alerts := make([]Alert, 0, len(drafts))
	for _, draft := range drafts {
		alerts = append(alerts, Alert{
			Kind: draft.Kind, Target: draft.Target, Level: draft.Level,
			Title: draft.Title, Message: draft.Message, Status: "firing",
			FirstTime: now, LastTime: now,
		})
	}
	return alerts
}

// MemoryStore is an in-process alert history used by tests and when no database is configured.
type MemoryStore struct {
	items []Alert
}

// Sync opens, updates, and resolves alerts, then returns those still firing.
func (s *MemoryStore) Sync(_ context.Context, desired []Draft) ([]Alert, error) {
	now := time.Now()
	seen := map[string]bool{}
	for _, draft := range desired {
		key := draft.Kind + "\x00" + draft.Target
		seen[key] = true
		found := false
		for i := range s.items {
			if s.items[i].Status == "firing" && s.items[i].Kind == draft.Kind && s.items[i].Target == draft.Target {
				s.items[i].Level = draft.Level
				s.items[i].Title = draft.Title
				s.items[i].Message = draft.Message
				s.items[i].LastTime = now
				found = true
				break
			}
		}
		if found {
			continue
		}
		s.items = append(s.items, Alert{
			ID: newID(), Kind: draft.Kind, Target: draft.Target, Level: draft.Level,
			Title: draft.Title, Message: draft.Message, Status: "firing",
			FirstTime: now, LastTime: now,
		})
	}
	for i := range s.items {
		if s.items[i].Status != "firing" || seen[s.items[i].Kind+"\x00"+s.items[i].Target] {
			continue
		}
		s.items[i].Status = "resolved"
		resolved := now
		s.items[i].ResolvedTime = &resolved
		s.items[i].LastTime = now
	}
	return s.firing(), nil
}

// List returns alert history, newest first.
func (s *MemoryStore) List(_ context.Context, page, pageSize int, status string) (Page[Alert], error) {
	matched := make([]Alert, 0)
	for i := len(s.items) - 1; i >= 0; i-- {
		item := s.items[i]
		if status != "" && item.Status != status {
			continue
		}
		matched = append(matched, item)
	}
	return pageOf(matched, page, pageSize), nil
}

func (s *MemoryStore) firing() []Alert {
	out := make([]Alert, 0)
	for _, item := range s.items {
		if item.Status == "firing" {
			out = append(out, item)
		}
	}
	return out
}

func pageOf(items []Alert, page, pageSize int) Page[Alert] {
	total := int64(len(items))
	start := (page - 1) * pageSize
	if start > len(items) {
		start = len(items)
	}
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}
	slice := append([]Alert(nil), items[start:end]...)
	if slice == nil {
		slice = []Alert{}
	}
	return Page[Alert]{Items: slice, Page: page, PageSize: pageSize, Total: total}
}

func newID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return fmt.Sprintf("alert-%d", time.Now().UnixNano())
	}
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", raw[0:4], raw[4:6], raw[6:8], raw[8:10], raw[10:])
}

func normalizePage(page, pageSize int) (int, int, error) {
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 20
	}
	if page < 1 || pageSize < 1 || pageSize > 100 {
		return 0, 0, &Error{Message: "分页参数不合法"}
	}
	return page, pageSize, nil
}

func normalizeStatus(status string) (string, error) {
	status = strings.TrimSpace(status)
	switch status {
	case "", "firing", "resolved":
		return status, nil
	default:
		return "", &Error{Message: "告警状态不合法"}
	}
}

// Error is a client-safe monitor error.
type Error struct {
	Message string
}

func (e *Error) Error() string { return e.Message }
