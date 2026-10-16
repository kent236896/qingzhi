package security

import (
	"context"
	"errors"
	"testing"
)

func TestPasswordPolicyAndLoginLock(t *testing.T) {
	svc := NewService(newMemoryStore())
	ctx := context.Background()

	if err := svc.CheckPassword(ctx, "short"); err == nil {
		t.Fatal("expected short password to fail the default minimum")
	}
	updated, err := svc.UpdatePolicy(ctx, Policy{
		PasswordMinLength: 8, PasswordMaxLength: 32,
		PasswordRequireLetter: true, PasswordRequireNumber: true, PasswordRequireSymbol: true,
		LoginFailLimit: 2, LoginLockMinutes: 15,
	}, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Updater != "admin" {
		t.Fatalf("updater = %s", updated.Updater)
	}
	if err := svc.CheckPassword(ctx, "password"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("weak password error = %v", err)
	}
	if err := svc.CheckPassword(ctx, "Passw0rd!"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.UpdatePolicy(ctx, Policy{PasswordMinLength: 4, PasswordMaxLength: 32, LoginLockMinutes: 15}, "admin"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid policy error = %v", err)
	}

	for i := 0; i < 2; i++ {
		if err := svc.RecordLogin(ctx, LoginLog{Username: "ada", Result: LoginCredentialFailure, Message: "用户名或密码错误"}); err != nil {
			t.Fatal(err)
		}
	}
	if err := svc.CheckLoginAllowed(ctx, "ada"); !errors.Is(err, ErrLocked) {
		t.Fatalf("lock error = %v", err)
	}
	if err := svc.RecordLogin(ctx, LoginLog{Username: "ada", Result: LoginSuccess, Message: "登录成功"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.CheckLoginAllowed(ctx, "ada"); err != nil {
		t.Fatalf("unlocked error = %v", err)
	}
}

func TestAuditLists(t *testing.T) {
	svc := NewService(newMemoryStore())
	ctx := context.Background()
	if err := svc.RecordOperate(ctx, OperateLog{Username: "admin", Module: "角色", Action: "修改", RequestMethod: "PATCH", RequestURL: "/api/v1/roles"}); err != nil {
		t.Fatal(err)
	}
	operates, err := svc.ListOperates(ctx, ListQuery{Keyword: "角色"})
	if err != nil || operates.Total != 1 {
		t.Fatalf("operates = %+v err = %v", operates, err)
	}
	if err := svc.RecordPermission(ctx, PermissionLog{
		OperatorName: "admin", RoleName: "超级管理员", ChangeType: ChangeMenus, Summary: "菜单权限由 1 项调整为 2 项",
	}); err != nil {
		t.Fatal(err)
	}
	if err := svc.RecordPermission(ctx, PermissionLog{ChangeType: "other"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid change error = %v", err)
	}
	logs, err := svc.ListPermissions(ctx, ListQuery{})
	if err != nil || logs.Total != 1 || logs.Items[0].ChangeType != ChangeMenus {
		t.Fatalf("permissions = %+v err = %v", logs, err)
	}

	if err := svc.RecordLogin(ctx, LoginLog{Username: "ada", Result: LoginCredentialFailure, Message: "用户名或密码错误"}); err != nil {
		t.Fatal(err)
	}
	page, err := svc.ListLogins(ctx, ListQuery{Keyword: "ada"})
	if err != nil || page.Total != 1 {
		t.Fatalf("logins = %+v err = %v", page, err)
	}
}
