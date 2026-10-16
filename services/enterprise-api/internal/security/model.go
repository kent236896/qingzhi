package security

import "time"

const (
	// LoginSuccess is a successful login.
	LoginSuccess = 0
	// LoginCredentialFailure is a rejected username or password.
	LoginCredentialFailure = 1
	// LoginLocked is a rejection because the account is temporarily locked.
	LoginLocked = 2

	// ChangeMenus records a role menu assignment.
	ChangeMenus = "menus"
	// ChangeUsers records a role user assignment.
	ChangeUsers = "users"
	// ChangeDepts records a custom data-scope department assignment.
	ChangeDepts = "depts"
	// ChangeDataScope records a role data-scope change.
	ChangeDataScope = "data_scope"
)

// LoginLog is one login attempt. The password is never stored.
type LoginLog struct {
	ID         string    `json:"id"`
	Username   string    `json:"username"`
	UserID     string    `json:"user_id"`
	TenantID   string    `json:"tenant_id"`
	Result     int       `json:"result"`
	Message    string    `json:"message"`
	IP         string    `json:"ip"`
	UserAgent  string    `json:"user_agent"`
	CreateTime time.Time `json:"create_time"`
}

// OperateLog is one authenticated write request. The body is not stored.
type OperateLog struct {
	ID            string    `json:"id"`
	UserID        string    `json:"user_id"`
	Username      string    `json:"username"`
	TenantID      string    `json:"tenant_id"`
	Module        string    `json:"module"`
	Action        string    `json:"action"`
	RequestMethod string    `json:"request_method"`
	RequestURL    string    `json:"request_url"`
	IP            string    `json:"ip"`
	UserAgent     string    `json:"user_agent"`
	StatusCode    int       `json:"status_code"`
	DurationMS    int       `json:"duration_ms"`
	RequestID     string    `json:"request_id"`
	CreateTime    time.Time `json:"create_time"`
}

// PermissionLog is one role authorization change.
type PermissionLog struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	OperatorID   string    `json:"operator_id"`
	OperatorName string    `json:"operator_name"`
	RoleID       string    `json:"role_id"`
	RoleName     string    `json:"role_name"`
	ChangeType   string    `json:"change_type"`
	Summary      string    `json:"summary"`
	BeforeText   string    `json:"before_text"`
	AfterText    string    `json:"after_text"`
	CreateTime   time.Time `json:"create_time"`
}

// Policy is the single system security policy.
type Policy struct {
	PasswordMinLength     int       `json:"password_min_length"`
	PasswordMaxLength     int       `json:"password_max_length"`
	PasswordRequireLetter bool      `json:"password_require_letter"`
	PasswordRequireNumber bool      `json:"password_require_number"`
	PasswordRequireSymbol bool      `json:"password_require_symbol"`
	LoginFailLimit        int       `json:"login_fail_limit"`
	LoginLockMinutes      int       `json:"login_lock_minutes"`
	Updater               string    `json:"updater"`
	UpdateTime            time.Time `json:"update_time"`
}

// Page is a paginated collection.
type Page[T any] struct {
	Items    []T   `json:"items"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

// ListQuery is the shared list request.
type ListQuery struct {
	Page     int
	PageSize int
	Keyword  string
	Result   *int
}
