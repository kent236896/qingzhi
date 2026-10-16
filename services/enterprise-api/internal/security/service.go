package security

import (
	"context"
	"crypto/rand"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Service applies security policy and writes audit logs.
type Service struct {
	store Store
}

// NewService returns a security service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// DefaultPolicy matches the historical password rule and does not lock logins.
func DefaultPolicy() Policy {
	return Policy{
		PasswordMinLength: 6,
		PasswordMaxLength: 32,
		LoginFailLimit:    0,
		LoginLockMinutes:  15,
		UpdateTime:        time.Now().UTC(),
	}
}

func (s *Service) Policy(ctx context.Context) (Policy, error) {
	item, err := s.store.FindPolicy(ctx)
	if err != nil {
		return Policy{}, err
	}
	if item.PasswordMinLength == 0 {
		return DefaultPolicy(), nil
	}
	return item, nil
}

func (s *Service) UpdatePolicy(ctx context.Context, item Policy, actor string) (Policy, error) {
	if err := validatePolicy(item); err != nil {
		return Policy{}, err
	}
	item.Updater = clip(strings.TrimSpace(actor), 64)
	item.UpdateTime = time.Now().UTC()
	if err := s.store.SavePolicy(ctx, item); err != nil {
		return Policy{}, err
	}
	return item, nil
}

func (s *Service) CheckPassword(ctx context.Context, password string) error {
	policy, err := s.Policy(ctx)
	if err != nil {
		return err
	}
	return checkPassword(password, policy)
}

func (s *Service) CheckLoginAllowed(ctx context.Context, username string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil
	}
	policy, err := s.Policy(ctx)
	if err != nil {
		return err
	}
	if policy.LoginFailLimit <= 0 {
		return nil
	}
	since := time.Now().UTC().Add(-time.Duration(policy.LoginLockMinutes) * time.Minute)
	attempts, err := s.store.ListLoginAttempts(ctx, username, since)
	if err != nil {
		return err
	}
	if credentialFailures(attempts) >= policy.LoginFailLimit {
		return locked("登录失败次数过多，请稍后再试")
	}
	return nil
}

func (s *Service) RecordLogin(ctx context.Context, item LoginLog) error {
	item.Username = clip(strings.TrimSpace(item.Username), 64)
	item.UserID = clip(item.UserID, 64)
	item.TenantID = clip(item.TenantID, 64)
	item.Message = clip(strings.TrimSpace(item.Message), 200)
	item.IP = clip(item.IP, 64)
	item.UserAgent = clip(item.UserAgent, 512)
	if item.Result != LoginSuccess && item.Result != LoginCredentialFailure && item.Result != LoginLocked {
		return invalid("登录结果不合法")
	}
	item.ID = newID()
	item.CreateTime = time.Now().UTC()
	return s.store.InsertLogin(ctx, item)
}

func (s *Service) ListLogins(ctx context.Context, query ListQuery) (Page[LoginLog], error) {
	page, pageSize, err := normalizePage(query.Page, query.PageSize)
	if err != nil {
		return Page[LoginLog]{}, err
	}
	if query.Result != nil && *query.Result != LoginSuccess && *query.Result != LoginCredentialFailure && *query.Result != LoginLocked {
		return Page[LoginLog]{}, invalid("登录结果不合法")
	}
	items, total, err := s.store.ListLogins(ctx, strings.TrimSpace(query.Keyword), query.Result, page, pageSize)
	if err != nil {
		return Page[LoginLog]{}, err
	}
	return newPage(items, page, pageSize, total), nil
}

func (s *Service) RecordOperate(ctx context.Context, item OperateLog) error {
	item.UserID = clip(item.UserID, 64)
	item.Username = clip(item.Username, 64)
	item.TenantID = clip(item.TenantID, 64)
	item.Module = clip(item.Module, 64)
	item.Action = clip(item.Action, 32)
	item.RequestMethod = clip(item.RequestMethod, 10)
	item.RequestURL = clip(item.RequestURL, 255)
	item.IP = clip(item.IP, 64)
	item.UserAgent = clip(item.UserAgent, 512)
	item.RequestID = clip(item.RequestID, 64)
	if item.DurationMS < 0 {
		item.DurationMS = 0
	}
	item.ID = newID()
	item.CreateTime = time.Now().UTC()
	return s.store.InsertOperate(ctx, item)
}

func (s *Service) ListOperates(ctx context.Context, query ListQuery) (Page[OperateLog], error) {
	page, pageSize, err := normalizePage(query.Page, query.PageSize)
	if err != nil {
		return Page[OperateLog]{}, err
	}
	items, total, err := s.store.ListOperates(ctx, strings.TrimSpace(query.Keyword), page, pageSize)
	if err != nil {
		return Page[OperateLog]{}, err
	}
	return newPage(items, page, pageSize, total), nil
}

func (s *Service) RecordPermission(ctx context.Context, item PermissionLog) error {
	switch item.ChangeType {
	case ChangeMenus, ChangeUsers, ChangeDepts, ChangeDataScope:
	default:
		return invalid("权限变更类型不合法")
	}
	item.TenantID = clip(item.TenantID, 64)
	item.OperatorID = clip(item.OperatorID, 64)
	item.OperatorName = clip(item.OperatorName, 64)
	item.RoleID = clip(item.RoleID, 64)
	item.RoleName = clip(item.RoleName, 64)
	item.Summary = clip(strings.TrimSpace(item.Summary), 500)
	item.ID = newID()
	item.CreateTime = time.Now().UTC()
	return s.store.InsertPermission(ctx, item)
}

func (s *Service) ListPermissions(ctx context.Context, query ListQuery) (Page[PermissionLog], error) {
	page, pageSize, err := normalizePage(query.Page, query.PageSize)
	if err != nil {
		return Page[PermissionLog]{}, err
	}
	items, total, err := s.store.ListPermissions(ctx, strings.TrimSpace(query.Keyword), page, pageSize)
	if err != nil {
		return Page[PermissionLog]{}, err
	}
	return newPage(items, page, pageSize, total), nil
}

func checkPassword(password string, policy Policy) error {
	length := utf8.RuneCountInString(password)
	if length < policy.PasswordMinLength || length > policy.PasswordMaxLength {
		return invalid(fmt.Sprintf("密码长度必须在 %d 到 %d 个字符之间", policy.PasswordMinLength, policy.PasswordMaxLength))
	}
	var letter, number, symbol bool
	for _, r := range password {
		switch {
		case unicode.IsLetter(r):
			letter = true
		case unicode.IsDigit(r):
			number = true
		default:
			symbol = true
		}
	}
	if policy.PasswordRequireLetter && !letter {
		return invalid("密码必须包含字母")
	}
	if policy.PasswordRequireNumber && !number {
		return invalid("密码必须包含数字")
	}
	if policy.PasswordRequireSymbol && !symbol {
		return invalid("密码必须包含符号")
	}
	return nil
}

func validatePolicy(item Policy) error {
	if item.PasswordMinLength < 6 || item.PasswordMinLength > 32 {
		return invalid("密码最小长度必须在 6 到 32 之间")
	}
	if item.PasswordMaxLength < item.PasswordMinLength || item.PasswordMaxLength > 32 {
		return invalid("密码最大长度必须不小于最小长度，且不超过 32")
	}
	if item.LoginFailLimit < 0 || item.LoginFailLimit > 20 {
		return invalid("登录失败锁定次数必须在 0 到 20 之间")
	}
	if item.LoginLockMinutes < 1 || item.LoginLockMinutes > 1440 {
		return invalid("锁定窗口必须在 1 到 1440 分钟之间")
	}
	return nil
}

func credentialFailures(items []LoginLog) int {
	var lastSuccess time.Time
	for _, item := range items {
		if item.Result == LoginSuccess && item.CreateTime.After(lastSuccess) {
			lastSuccess = item.CreateTime
		}
	}
	count := 0
	for _, item := range items {
		if item.Result != LoginCredentialFailure {
			continue
		}
		if lastSuccess.IsZero() || item.CreateTime.After(lastSuccess) {
			count++
		}
	}
	return count
}

func normalizePage(page, pageSize int) (int, int, error) {
	if page == 0 {
		page = 1
	}
	if pageSize == 0 {
		pageSize = 20
	}
	if page < 1 || pageSize < 1 || pageSize > 100 {
		return 0, 0, invalid("分页参数不合法")
	}
	return page, pageSize, nil
}

func newPage[T any](items []T, page, pageSize int, total int64) Page[T] {
	if items == nil {
		items = []T{}
	}
	return Page[T]{Items: items, Page: page, PageSize: pageSize, Total: total}
}

func clip(value string, limit int) string {
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	return string(runes[:limit])
}

func newID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		panic(err)
	}
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16])
}
