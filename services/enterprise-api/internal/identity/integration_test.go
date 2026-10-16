package identity

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gorm.io/gorm"
	"qingzhi/enterprise-api/internal/config"
	"qingzhi/enterprise-api/internal/platform/database"
)

var errRollback = errors.New("rollback identity integration")

func TestPostgresIdentityFlowRollsBack(t *testing.T) {
	if os.Getenv("ENV_FILE") == "" {
		if path := findRepoEnv(t); path != "" {
			t.Setenv("ENV_FILE", path)
		}
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	db, err := database.OpenPostgres(cfg.Postgres)
	if err != nil {
		t.Fatalf("open postgres: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := db.Close(); closeErr != nil {
			t.Errorf("close postgres: %v", closeErr)
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.Ping(ctx); err != nil {
		t.Skipf("PostgreSQL 不可达，集成测试已跳过，不记为通过: %v", err)
	}

	var present bool
	if err := db.GORM().Raw(`SELECT to_regclass('public.sys_users') IS NOT NULL`).Scan(&present).Error; err != nil {
		t.Fatalf("check users table: %v", err)
	}
	if !present {
		t.Fatal("sys_users 表不存在，请先执行迁移 000003_sys_prefix 和 000004_sys_posts")
	}

	var flowErr error
	err = db.GORM().Transaction(func(tx *gorm.DB) error {
		flowErr = runIdentityFlow(NewService(NewGormStore(tx)), tx)
		return errRollback
	})
	if flowErr != nil {
		t.Fatal(flowErr)
	}
	if !errors.Is(err, errRollback) {
		t.Fatalf("transaction error = %v", err)
	}
}

func runIdentityFlow(svc *Service, tx *gorm.DB) error {
	ctx := context.Background()
	tenant, err := svc.CreateTenant(ctx, CreateTenantInput{Name: "集成租户", Code: "integration-tenant", ContactName: "联系人"})
	if err != nil {
		return err
	}
	err = withinSavepoint(tx, func() error {
		_, createErr := svc.CreateTenant(ctx, CreateTenantInput{Name: "重复租户", Code: "integration-tenant", ContactName: "联系人"})
		return createErr
	})
	if !errors.Is(err, ErrConflict) {
		return errors.New("duplicate tenant code was not rejected")
	}
	roots, err := svc.ListOrganizations(ctx, tenant.ID)
	if err != nil {
		return err
	}
	if len(roots) != 1 || roots[0].ParentID != "" {
		return errors.New("tenant was not created with exactly one root organization")
	}
	root := roots[0].Organization
	err = withinSavepoint(tx, func() error {
		_, createErr := svc.CreateOrganization(ctx, tenant.ID, CreateOrganizationInput{Name: "第二个主体", Code: "HQ"})
		return createErr
	})
	if !errors.Is(err, ErrInvalid) {
		return errors.New("second root organization was not rejected")
	}
	child, err := svc.CreateOrganization(ctx, tenant.ID, CreateOrganizationInput{Name: "研发", Code: "RD", ParentID: root.ID})
	if err != nil {
		return err
	}
	parentID := child.ID
	if _, err := svc.UpdateOrganization(ctx, tenant.ID, root.ID, UpdateOrganizationInput{ParentSet: true, ParentID: &parentID}); !errors.Is(err, ErrInvalid) {
		return errors.New("organization cycle was not rejected")
	}
	user, err := svc.CreateUser(ctx, CreateUserInput{Username: "integration.user", Password: "Passw0rd!", Nickname: "集成用户", Email: "integration@example.com"})
	if err != nil {
		return err
	}
	member, err := svc.CreateMember(ctx, tenant.ID, CreateMemberInput{
		UserID: user.ID,
		AssignmentInput: AssignmentInput{
			OrganizationIDs:       []string{root.ID, child.ID},
			PrimaryOrganizationID: child.ID,
		},
	})
	if err != nil {
		return err
	}
	if member.PrimaryOrganizationID != child.ID || len(member.Organizations) != 2 {
		return errors.New("primary and concurrent departments were not saved")
	}
	err = withinSavepoint(tx, func() error {
		_, createErr := svc.CreateMember(ctx, tenant.ID, CreateMemberInput{
			UserID: user.ID,
			AssignmentInput: AssignmentInput{
				OrganizationIDs:       []string{root.ID},
				PrimaryOrganizationID: root.ID,
			},
		})
		return createErr
	})
	if !errors.Is(err, ErrConflict) {
		return errors.New("duplicate membership was not rejected")
	}
	if err := svc.DeleteOrganization(ctx, tenant.ID, child.ID); !errors.Is(err, ErrConflict) {
		return errors.New("assigned organization delete was not rejected")
	}
	post, err := svc.CreatePost(ctx, tenant.ID, CreatePostInput{Code: "PM", Name: "产品经理", Sort: 1})
	if err != nil {
		return err
	}
	err = withinSavepoint(tx, func() error {
		_, createErr := svc.CreatePost(ctx, tenant.ID, CreatePostInput{Code: "PM", Name: "重复岗位"})
		return createErr
	})
	if !errors.Is(err, ErrConflict) {
		return errors.New("duplicate post code was not rejected")
	}
	if err := svc.DeletePost(ctx, tenant.ID, post.ID); err != nil {
		return err
	}
	directory, err := svc.CreateMenu(ctx, CreateMenuInput{Name: "系统管理", Type: MenuTypeDirectory})
	if err != nil {
		return err
	}
	childMenu, err := svc.CreateMenu(ctx, CreateMenuInput{Name: "用户管理", Type: MenuTypeMenu, ParentID: directory.ID, Path: "users"})
	if err != nil {
		return err
	}
	menuParentID := childMenu.ID
	if _, err := svc.UpdateMenu(ctx, directory.ID, UpdateMenuInput{ParentSet: true, ParentID: &menuParentID}); !errors.Is(err, ErrInvalid) {
		return errors.New("menu cycle was not rejected")
	}
	if err := svc.DeleteMenu(ctx, directory.ID); !errors.Is(err, ErrConflict) {
		return errors.New("parent menu delete was not rejected")
	}
	if err := svc.DeleteMenu(ctx, childMenu.ID); err != nil {
		return err
	}
	if err := svc.DeleteMenu(ctx, directory.ID); err != nil {
		return err
	}
	role, err := svc.CreateRole(ctx, tenant.ID, CreateRoleInput{Name: "管理员", Code: "tenant-admin"})
	if err != nil {
		return err
	}
	err = withinSavepoint(tx, func() error {
		_, createErr := svc.CreateRole(ctx, tenant.ID, CreateRoleInput{Name: "重复", Code: "tenant-admin"})
		return createErr
	})
	if !errors.Is(err, ErrConflict) {
		return errors.New("duplicate role code was not rejected")
	}
	menu, err := svc.CreateMenu(ctx, CreateMenuInput{Name: "角色页", Type: MenuTypeMenu, Path: "roles"})
	if err != nil {
		return err
	}
	if _, err := svc.ReplaceRoleMenus(ctx, tenant.ID, role.ID, []string{menu.ID}); err != nil {
		return err
	}
	roleUser, err := svc.CreateUser(ctx, CreateUserInput{Username: "role.user", Password: "Passw0rd!", Nickname: "角色用户"})
	if err != nil {
		return err
	}
	if _, err := svc.CreateMember(ctx, tenant.ID, CreateMemberInput{
		UserID: roleUser.ID,
		AssignmentInput: AssignmentInput{
			OrganizationIDs:       []string{root.ID},
			PrimaryOrganizationID: root.ID,
		},
	}); err != nil {
		return err
	}
	if _, err := svc.ReplaceRoleUsers(ctx, tenant.ID, role.ID, []string{roleUser.ID}); err != nil {
		return err
	}
	if err := svc.DeleteRole(ctx, tenant.ID, role.ID); !errors.Is(err, ErrConflict) {
		return errors.New("assigned role delete was not rejected")
	}
	if _, err := svc.ReplaceRoleUsers(ctx, tenant.ID, role.ID, nil); err != nil {
		return err
	}
	if err := svc.DeleteRole(ctx, tenant.ID, role.ID); err != nil {
		return err
	}
	return svc.DeleteMenu(ctx, menu.ID)
}

func withinSavepoint(tx *gorm.DB, fn func() error) error {
	if err := tx.SavePoint("identity_expect").Error; err != nil {
		return err
	}
	err := fn()
	if rbErr := tx.RollbackTo("identity_expect").Error; rbErr != nil {
		return rbErr
	}
	return err
}

func findRepoEnv(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(dir, ".env")
		info, statErr := os.Stat(candidate)
		if statErr == nil && !info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}
