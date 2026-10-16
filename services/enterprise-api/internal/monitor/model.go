package monitor

import "time"

// ServiceState is the live connectivity of one dependency.
type ServiceState struct {
	API      string `json:"api"`
	Database string `json:"database"`
	Redis    string `json:"redis"`
}

// Traffic is aggregated from real requests inside the monitor window.
type Traffic struct {
	Requests     int `json:"requests"`
	ServerErrors int `json:"server_errors"`
	ClientErrors int `json:"client_errors"`
	Timeouts     int `json:"timeouts"`
	Rejected     int `json:"rejected"`
	AvgMS        int `json:"avg_ms"`
	P95MS        int `json:"p95_ms"`
}

// ErrorStat groups failures by type inside the window.
type ErrorStat struct {
	Type     string    `json:"type"`
	Count    int       `json:"count"`
	LastTime time.Time `json:"last_time"`
}

// RecentError is one failed or rejected request. The path has no query string.
type RecentError struct {
	Time       time.Time `json:"time"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	Status     int       `json:"status"`
	DurationMS int       `json:"duration_ms"`
	Type       string    `json:"type"`
	RequestID  string    `json:"request_id"`
}

// Resources is a host sample. Mount is the volume root, not a user path.
type Resources struct {
	Available     bool    `json:"available"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryUsed    uint64  `json:"memory_used"`
	MemoryTotal   uint64  `json:"memory_total"`
	MemoryPercent float64 `json:"memory_percent"`
	DiskUsed      uint64  `json:"disk_used"`
	DiskTotal     uint64  `json:"disk_total"`
	DiskPercent   float64 `json:"disk_percent"`
	DiskMount     string  `json:"disk_mount"`
}

// Protection is the active request guard. A zero rate means the limiter is off.
type Protection struct {
	RateLimitRPS     int   `json:"rate_limit_rps"`
	RateLimitBurst   int   `json:"rate_limit_burst"`
	MaxBodyBytes     int64 `json:"max_body_bytes"`
	RequestTimeoutMS int   `json:"request_timeout_ms"`
}

// Alert is a firing or resolved condition.
type Alert struct {
	ID           string     `json:"id" gorm:"primaryKey;type:uuid"`
	Kind         string     `json:"kind" gorm:"size:32"`
	Target       string     `json:"target" gorm:"size:64"`
	Level        string     `json:"level" gorm:"size:16"`
	Title        string     `json:"title" gorm:"size:100"`
	Message      string     `json:"message" gorm:"size:500"`
	Status       string     `json:"status" gorm:"size:16"`
	FirstTime    time.Time  `json:"first_time"`
	LastTime     time.Time  `json:"last_time"`
	ResolvedTime *time.Time `json:"resolved_time"`
}

// TableName keeps the monitor alert table explicit.
func (Alert) TableName() string { return "sys_monitor_alerts" }

// Overview is the monitoring page payload.
type Overview struct {
	CollectedAt   time.Time     `json:"collected_at"`
	WindowSeconds int           `json:"window_seconds"`
	Service       ServiceState  `json:"service"`
	Traffic       Traffic       `json:"traffic"`
	Errors        []ErrorStat   `json:"errors"`
	RecentErrors  []RecentError `json:"recent_errors"`
	Resources     Resources     `json:"resources"`
	Protection    Protection    `json:"protection"`
	Alerts        []Alert       `json:"alerts"`
}

// Page is a paginated alert history.
type Page[T any] struct {
	Items    []T   `json:"items"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

// Draft is an alert condition before it is stored.
type Draft struct {
	Kind    string
	Target  string
	Level   string
	Title   string
	Message string
}
