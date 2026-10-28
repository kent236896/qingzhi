package identity

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"time"
)

// BootstrapInput seeds the first tenant, root organization, and super-admin account.
// Empty fields are filled with random or sensible defaults and returned for console display.
type BootstrapInput struct {
	TenantID      string
	TenantName    string
	TenantCode    string
	AdminUsername string
	AdminPassword string
	AdminNickname string
}

// BootstrapResult describes what was created or why bootstrap was skipped.
type BootstrapResult struct {
	Created               bool
	SkippedReason         string
	TenantID              string
	TenantName            string
	TenantCode            string
	OrgID                 string
	OrgName               string
	UserID                string
	Username              string
	Password              string
	AdminNickname         string
	PasswordGenerated     bool
	UsernameGenerated     bool
	TenantIDGenerated     bool
	TenantCodeGenerated   bool
	TenantNameGenerated   bool
	RoleID                string
}

// BootstrapIfEmpty creates the initial tenant suite when the database has no tenants.
// It is idempotent: existing tenants cause a no-op (SSO may still read TenantID of the oldest one).
func (s *Service) BootstrapIfEmpty(ctx context.Context, input BootstrapInput) (BootstrapResult, error) {
	_, total, err := s.store.ListTenants(ctx, "", 1, 1)
	if err != nil {
		return BootstrapResult{}, err
	}
	if total > 0 {
		tid, firstErr := s.FirstActiveTenantID(ctx)
		if firstErr != nil {
			return BootstrapResult{Created: false, SkippedReason: "tenants already exist"}, nil
		}
		return BootstrapResult{Created: false, SkippedReason: "tenants already exist", TenantID: tid}, nil
	}

	result, err := prepareBootstrap(input)
	if err != nil {
		return BootstrapResult{}, err
	}

	passwordHash, err := hashPassword(result.Password)
	if err != nil {
		return BootstrapResult{}, err
	}

	now := time.Now().UTC()
	tenant := Tenant{
		ID:           result.TenantID,
		Name:         result.TenantName,
		Code:         result.TenantCode,
		ContactName:  result.AdminNickname,
		Status:       StatusNormal,
		AccountCount: 0,
		CreateTime:   now,
		UpdateTime:   now,
	}
	org := rootOrganization(tenant, now)
	user := User{
		ID:                   newID(),
		Username:             result.Username,
		PasswordHash:         passwordHash,
		Nickname:             result.AdminNickname,
		Remark:               "系统初始化管理员",
		Status:               StatusNormal,
		LocalPasswordEnabled: true,
		CreateTime:           now,
		UpdateTime:           now,
	}
	role := Role{
		ID:         newID(),
		TenantID:   tenant.ID,
		Name:       "超级管理员",
		Code:       "super_admin",
		Sort:       0,
		Status:     StatusNormal,
		Type:       RoleTypeSystem,
		DataScope:  DataScopeAll,
		Remark:     "系统初始化角色",
		CreateTime: now,
		UpdateTime: now,
	}
	membership := TenantMembership{
		ID:         newID(),
		TenantID:   tenant.ID,
		UserID:     user.ID,
		Status:     StatusNormal,
		CreateTime: now,
		UpdateTime: now,
	}
	assignments := newAssignments(tenant.ID, user.ID, AssignmentInput{
		OrganizationIDs:       []string{org.ID},
		PrimaryOrganizationID: org.ID,
	}, now)

	menus, err := s.store.ListMenus(ctx)
	if err != nil {
		return BootstrapResult{}, err
	}
	menuIDs := make([]string, 0, len(menus))
	for _, menu := range menus {
		menuIDs = append(menuIDs, menu.ID)
	}

	err = s.store.Transaction(ctx, func(tx Store) error {
		tenant.ContactUserID = user.ID
		if err := tx.InsertTenant(ctx, tenant); err != nil {
			return err
		}
		if err := tx.InsertOrganization(ctx, org); err != nil {
			return err
		}
		if err := tx.InsertUser(ctx, user); err != nil {
			return err
		}
		if err := tx.InsertMembership(ctx, membership); err != nil {
			return err
		}
		if err := tx.ReplaceAssignments(ctx, tenant.ID, user.ID, assignments); err != nil {
			return err
		}
		if err := tx.InsertRole(ctx, role); err != nil {
			return err
		}
		if err := tx.ReplaceRoleMenus(ctx, tenant.ID, role.ID, menuIDs); err != nil {
			return err
		}
		return tx.ReplaceRoleUsers(ctx, tenant.ID, role.ID, []string{user.ID})
	})
	if err != nil {
		return BootstrapResult{}, err
	}

	result.Created = true
	result.OrgID = org.ID
	result.OrgName = org.Name
	result.UserID = user.ID
	result.RoleID = role.ID
	return result, nil
}

func prepareBootstrap(input BootstrapInput) (BootstrapResult, error) {
	result := BootstrapResult{
		TenantID:      strings.TrimSpace(input.TenantID),
		TenantName:    strings.TrimSpace(input.TenantName),
		TenantCode:    strings.ToLower(strings.TrimSpace(input.TenantCode)),
		Username:      strings.TrimSpace(input.AdminUsername),
		Password:      input.AdminPassword,
		AdminNickname: strings.TrimSpace(input.AdminNickname),
	}

	if result.TenantID == "" {
		result.TenantID = newID()
		result.TenantIDGenerated = true
	}
	if result.TenantName == "" {
		result.TenantName = "默认组织"
		result.TenantNameGenerated = true
	}
	if result.TenantCode == "" {
		suffix, err := randomHex(8)
		if err != nil {
			return BootstrapResult{}, err
		}
		result.TenantCode = "t-" + suffix
		result.TenantCodeGenerated = true
	}
	if result.Username == "" {
		suffix, err := randomHex(4)
		if err != nil {
			return BootstrapResult{}, err
		}
		result.Username = "admin-" + suffix
		result.UsernameGenerated = true
	}
	if result.Password == "" {
		password, err := randomPassword()
		if err != nil {
			return BootstrapResult{}, err
		}
		result.Password = password
		result.PasswordGenerated = true
	}
	if result.AdminNickname == "" {
		result.AdminNickname = "管理员"
	}

	if err := validateName(result.TenantName); err != nil {
		return BootstrapResult{}, err
	}
	if err := validateTenantCode(result.TenantCode); err != nil {
		return BootstrapResult{}, err
	}
	if err := validateUsername(result.Username); err != nil {
		return BootstrapResult{}, err
	}
	if err := validatePassword(result.Password); err != nil {
		return BootstrapResult{}, err
	}
	if err := validateName(result.AdminNickname); err != nil {
		return BootstrapResult{}, err
	}
	return result, nil
}

func randomHex(n int) (string, error) {
	if n <= 0 {
		return "", fmt.Errorf("random hex length must be positive")
	}
	buf := make([]byte, (n+1)/2)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf)[:n], nil
}

// PrintBootstrapCredentials writes the initial login set to stdout once.
func PrintBootstrapCredentials(out *os.File, result BootstrapResult) {
	if out == nil || !result.Created {
		return
	}
	_, _ = fmt.Fprintln(out, "")
	_, _ = fmt.Fprintln(out, "========== 系统初始账号已创建 ==========")
	_, _ = fmt.Fprintf(out, "租户 ID:   %s\n", result.TenantID)
	_, _ = fmt.Fprintf(out, "租户编码: %s\n", result.TenantCode)
	_, _ = fmt.Fprintf(out, "租户名称: %s\n", result.TenantName)
	_, _ = fmt.Fprintf(out, "组织 ID:   %s\n", result.OrgID)
	_, _ = fmt.Fprintf(out, "组织名称: %s\n", result.OrgName)
	_, _ = fmt.Fprintf(out, "管理员:   %s\n", result.Username)
	_, _ = fmt.Fprintf(out, "密码:     %s\n", result.Password)
	_, _ = fmt.Fprintln(out, "请尽快登录并修改密码。此信息仅在首次初始化时打印。")
	_, _ = fmt.Fprintln(out, "======================================")
	_, _ = fmt.Fprintln(out, "")
}
