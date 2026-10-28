package identity

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
)

func TestBootstrapIfEmptyCreatesTenantSuite(t *testing.T) {
	svc := NewService(newMemoryStore())
	ctx := context.Background()

	result, err := svc.BootstrapIfEmpty(ctx, BootstrapInput{})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Created {
		t.Fatal("expected bootstrap to create tenant suite")
	}
	if result.TenantID == "" || result.OrgID == "" || result.UserID == "" || result.RoleID == "" {
		t.Fatalf("missing identifiers: %+v", result)
	}
	if !result.PasswordGenerated || !result.UsernameGenerated || !result.TenantIDGenerated || !result.TenantCodeGenerated {
		t.Fatalf("expected generated fields: %+v", result)
	}
	if !strings.HasPrefix(result.Username, "admin-") {
		t.Fatalf("username = %q", result.Username)
	}
	if !strings.HasPrefix(result.TenantCode, "t-") {
		t.Fatalf("tenant code = %q", result.TenantCode)
	}

	login, err := svc.Authenticate(ctx, LoginInput{Username: result.Username, Password: result.Password, TenantID: result.TenantID})
	if err != nil {
		t.Fatal(err)
	}
	if login.TenantID != result.TenantID || login.User.ID != result.UserID {
		t.Fatalf("login = %+v", login)
	}
	if len(login.Access.Roles) != 1 || login.Access.Roles[0].Code != "super_admin" {
		t.Fatalf("roles = %+v", login.Access.Roles)
	}

	second, err := svc.BootstrapIfEmpty(ctx, BootstrapInput{AdminUsername: "other-admin", AdminPassword: "Passw0rd!"})
	if err != nil {
		t.Fatal(err)
	}
	if second.Created {
		t.Fatal("second bootstrap must be skipped")
	}
	if second.TenantID != result.TenantID {
		t.Fatalf("skipped tenant id = %q, want %q", second.TenantID, result.TenantID)
	}
}

func TestBootstrapIfEmptyUsesEnvValues(t *testing.T) {
	svc := NewService(newMemoryStore())
	ctx := context.Background()
	tenantID := newID()

	result, err := svc.BootstrapIfEmpty(ctx, BootstrapInput{
		TenantID:      tenantID,
		TenantName:    "演示租户",
		TenantCode:    "demo-org",
		AdminUsername: "demoadmin",
		AdminPassword: "Demo@123456",
		AdminNickname: "演示管理员",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Created {
		t.Fatal("expected create")
	}
	if result.TenantID != tenantID || result.TenantCode != "demo-org" || result.Username != "demoadmin" {
		t.Fatalf("result = %+v", result)
	}
	if result.PasswordGenerated || result.UsernameGenerated || result.TenantIDGenerated {
		t.Fatalf("env values should not be marked generated: %+v", result)
	}
	if result.Password != "Demo@123456" {
		t.Fatalf("password = %q", result.Password)
	}
}

func TestPrintBootstrapCredentials(t *testing.T) {
	result := BootstrapResult{
		Created:    true,
		TenantID:   "tid",
		TenantCode: "t-code",
		TenantName: "名称",
		OrgID:      "oid",
		OrgName:    "名称",
		Username:   "admin-ab12",
		Password:   "secret-password-value",
	}
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	PrintBootstrapCredentials(w, result)
	_ = w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	_ = r.Close()
	out := buf.String()
	if !strings.Contains(out, "admin-ab12") || !strings.Contains(out, "secret-password-value") || !strings.Contains(out, "tid") {
		t.Fatalf("output missing credentials: %s", out)
	}
}
