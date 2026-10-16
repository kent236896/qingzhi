package security

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
)

// GormStore stores security data in PostgreSQL.
type GormStore struct {
	db *gorm.DB
}

// NewGormStore returns a PostgreSQL store.
func NewGormStore(db *gorm.DB) *GormStore {
	return &GormStore{db: db}
}

func (s *GormStore) Transaction(ctx context.Context, fn func(Store) error) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&GormStore{db: tx})
	})
}

func (s *GormStore) InsertLogin(ctx context.Context, item LoginLog) error {
	return s.db.WithContext(ctx).Create(loginRowFrom(item)).Error
}

func (s *GormStore) ListLogins(ctx context.Context, keyword string, result *int, page, pageSize int) ([]LoginLog, int64, error) {
	query := s.db.WithContext(ctx).Model(&loginRow{})
	if result != nil {
		query = query.Where("result = ?", *result)
	}
	query = applyKeyword(query, keyword, "username", "message", "ip")
	return pageQuery(query, page, pageSize, "create_time DESC", func(rows []loginRow) []LoginLog {
		return loginsFrom(rows)
	})
}

func (s *GormStore) ListLoginAttempts(ctx context.Context, username string, since time.Time) ([]LoginLog, error) {
	var rows []loginRow
	err := s.db.WithContext(ctx).
		Where("username = ? AND create_time >= ?", username, since).
		Order("create_time ASC").
		Limit(100).
		Find(&rows).Error
	return loginsFrom(rows), err
}

func (s *GormStore) InsertOperate(ctx context.Context, item OperateLog) error {
	return s.db.WithContext(ctx).Create(operateRowFrom(item)).Error
}

func (s *GormStore) ListOperates(ctx context.Context, keyword string, page, pageSize int) ([]OperateLog, int64, error) {
	query := applyKeyword(s.db.WithContext(ctx).Model(&operateRow{}), keyword, "username", "module", "action", "request_url")
	return pageQuery(query, page, pageSize, "create_time DESC", func(rows []operateRow) []OperateLog {
		return operatesFrom(rows)
	})
}

func (s *GormStore) InsertPermission(ctx context.Context, item PermissionLog) error {
	return s.db.WithContext(ctx).Create(permissionRowFrom(item)).Error
}

func (s *GormStore) ListPermissions(ctx context.Context, keyword string, page, pageSize int) ([]PermissionLog, int64, error) {
	query := applyKeyword(s.db.WithContext(ctx).Model(&permissionRow{}), keyword, "operator_name", "role_name", "summary")
	return pageQuery(query, page, pageSize, "create_time DESC", func(rows []permissionRow) []PermissionLog {
		return permissionsFrom(rows)
	})
}

func (s *GormStore) FindPolicy(ctx context.Context) (Policy, error) {
	var row policyRow
	err := s.db.WithContext(ctx).First(&row, "id = ?", 1).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return DefaultPolicy(), nil
	}
	if err != nil {
		return Policy{}, err
	}
	return row.model(), nil
}

func (s *GormStore) SavePolicy(ctx context.Context, item Policy) error {
	row := policyRowFrom(item)
	return s.db.WithContext(ctx).Save(&row).Error
}

func pageQuery[Row any, Item any](query *gorm.DB, page, pageSize int, order string, convert func([]Row) []Item) ([]Item, int64, error) {
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []Row
	err := query.Session(&gorm.Session{}).Order(order).Limit(pageSize).Offset((page - 1) * pageSize).Find(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	return convert(rows), total, nil
}

type loginRow struct {
	ID         string    `gorm:"column:id;primaryKey"`
	Username   string    `gorm:"column:username"`
	UserID     string    `gorm:"column:user_id"`
	TenantID   string    `gorm:"column:tenant_id"`
	Result     int       `gorm:"column:result"`
	Message    string    `gorm:"column:message"`
	IP         string    `gorm:"column:ip"`
	UserAgent  string    `gorm:"column:user_agent"`
	CreateTime time.Time `gorm:"column:create_time"`
}

func (loginRow) TableName() string { return "sys_login_logs" }

func loginRowFrom(item LoginLog) loginRow {
	return loginRow{
		ID: item.ID, Username: item.Username, UserID: item.UserID, TenantID: item.TenantID,
		Result: item.Result, Message: item.Message, IP: item.IP, UserAgent: item.UserAgent, CreateTime: item.CreateTime,
	}
}

func (row loginRow) model() LoginLog {
	return LoginLog{
		ID: row.ID, Username: row.Username, UserID: row.UserID, TenantID: row.TenantID,
		Result: row.Result, Message: row.Message, IP: row.IP, UserAgent: row.UserAgent, CreateTime: row.CreateTime,
	}
}

func loginsFrom(rows []loginRow) []LoginLog {
	items := make([]LoginLog, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.model())
	}
	return items
}

type operateRow struct {
	ID            string    `gorm:"column:id;primaryKey"`
	UserID        string    `gorm:"column:user_id"`
	Username      string    `gorm:"column:username"`
	TenantID      string    `gorm:"column:tenant_id"`
	Module        string    `gorm:"column:module"`
	Action        string    `gorm:"column:action"`
	RequestMethod string    `gorm:"column:request_method"`
	RequestURL    string    `gorm:"column:request_url"`
	IP            string    `gorm:"column:ip"`
	UserAgent     string    `gorm:"column:user_agent"`
	StatusCode    int       `gorm:"column:status_code"`
	DurationMS    int       `gorm:"column:duration_ms"`
	RequestID     string    `gorm:"column:request_id"`
	CreateTime    time.Time `gorm:"column:create_time"`
}

func (operateRow) TableName() string { return "sys_operate_logs" }

func operateRowFrom(item OperateLog) operateRow {
	return operateRow{
		ID: item.ID, UserID: item.UserID, Username: item.Username, TenantID: item.TenantID,
		Module: item.Module, Action: item.Action, RequestMethod: item.RequestMethod, RequestURL: item.RequestURL,
		IP: item.IP, UserAgent: item.UserAgent, StatusCode: item.StatusCode, DurationMS: item.DurationMS,
		RequestID: item.RequestID, CreateTime: item.CreateTime,
	}
}

func (row operateRow) model() OperateLog {
	return OperateLog{
		ID: row.ID, UserID: row.UserID, Username: row.Username, TenantID: row.TenantID,
		Module: row.Module, Action: row.Action, RequestMethod: row.RequestMethod, RequestURL: row.RequestURL,
		IP: row.IP, UserAgent: row.UserAgent, StatusCode: row.StatusCode, DurationMS: row.DurationMS,
		RequestID: row.RequestID, CreateTime: row.CreateTime,
	}
}

func operatesFrom(rows []operateRow) []OperateLog {
	items := make([]OperateLog, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.model())
	}
	return items
}

type permissionRow struct {
	ID           string    `gorm:"column:id;primaryKey"`
	TenantID     string    `gorm:"column:tenant_id"`
	OperatorID   string    `gorm:"column:operator_id"`
	OperatorName string    `gorm:"column:operator_name"`
	RoleID       string    `gorm:"column:role_id"`
	RoleName     string    `gorm:"column:role_name"`
	ChangeType   string    `gorm:"column:change_type"`
	Summary      string    `gorm:"column:summary"`
	BeforeText   string    `gorm:"column:before_text"`
	AfterText    string    `gorm:"column:after_text"`
	CreateTime   time.Time `gorm:"column:create_time"`
}

func (permissionRow) TableName() string { return "sys_permission_logs" }

func permissionRowFrom(item PermissionLog) permissionRow {
	return permissionRow{
		ID: item.ID, TenantID: item.TenantID, OperatorID: item.OperatorID, OperatorName: item.OperatorName,
		RoleID: item.RoleID, RoleName: item.RoleName, ChangeType: item.ChangeType, Summary: item.Summary,
		BeforeText: item.BeforeText, AfterText: item.AfterText, CreateTime: item.CreateTime,
	}
}

func (row permissionRow) model() PermissionLog {
	return PermissionLog{
		ID: row.ID, TenantID: row.TenantID, OperatorID: row.OperatorID, OperatorName: row.OperatorName,
		RoleID: row.RoleID, RoleName: row.RoleName, ChangeType: row.ChangeType, Summary: row.Summary,
		BeforeText: row.BeforeText, AfterText: row.AfterText, CreateTime: row.CreateTime,
	}
}

func permissionsFrom(rows []permissionRow) []PermissionLog {
	items := make([]PermissionLog, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.model())
	}
	return items
}

type policyRow struct {
	ID                    int       `gorm:"column:id;primaryKey"`
	PasswordMinLength     int       `gorm:"column:password_min_length"`
	PasswordMaxLength     int       `gorm:"column:password_max_length"`
	PasswordRequireLetter bool      `gorm:"column:password_require_letter"`
	PasswordRequireNumber bool      `gorm:"column:password_require_number"`
	PasswordRequireSymbol bool      `gorm:"column:password_require_symbol"`
	LoginFailLimit        int       `gorm:"column:login_fail_limit"`
	LoginLockMinutes      int       `gorm:"column:login_lock_minutes"`
	Updater               string    `gorm:"column:updater"`
	UpdateTime            time.Time `gorm:"column:update_time"`
}

func (policyRow) TableName() string { return "sys_security_policy" }

func policyRowFrom(item Policy) policyRow {
	return policyRow{
		ID: 1, PasswordMinLength: item.PasswordMinLength, PasswordMaxLength: item.PasswordMaxLength,
		PasswordRequireLetter: item.PasswordRequireLetter, PasswordRequireNumber: item.PasswordRequireNumber,
		PasswordRequireSymbol: item.PasswordRequireSymbol, LoginFailLimit: item.LoginFailLimit,
		LoginLockMinutes: item.LoginLockMinutes, Updater: item.Updater, UpdateTime: item.UpdateTime,
	}
}

func (row policyRow) model() Policy {
	return Policy{
		PasswordMinLength: row.PasswordMinLength, PasswordMaxLength: row.PasswordMaxLength,
		PasswordRequireLetter: row.PasswordRequireLetter, PasswordRequireNumber: row.PasswordRequireNumber,
		PasswordRequireSymbol: row.PasswordRequireSymbol, LoginFailLimit: row.LoginFailLimit,
		LoginLockMinutes: row.LoginLockMinutes, Updater: row.Updater, UpdateTime: row.UpdateTime,
	}
}

func applyKeyword(query *gorm.DB, keyword string, columns ...string) *gorm.DB {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return query
	}
	escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(keyword)
	pattern := "%" + escaped + "%"
	parts := make([]string, 0, len(columns))
	args := make([]any, 0, len(columns))
	for _, column := range columns {
		parts = append(parts, column+" ILIKE ? ESCAPE '\\'")
		args = append(args, pattern)
	}
	return query.Where("("+strings.Join(parts, " OR ")+")", args...)
}
