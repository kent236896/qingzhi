package migrations

import (
	"io/fs"
	"strings"
	"testing"
)

func TestBaselineCreatesNoTables(t *testing.T) {
	for _, name := range []string{"000001_baseline.up.sql", "000001_baseline.down.sql"} {
		content, err := fs.ReadFile(FS, name)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(strings.ToUpper(string(content)), "CREATE TABLE") {
			t.Fatalf("%s creates a table; the baseline migration must stay empty", name)
		}
	}
}

func TestIdentityMigrationCreatesExpectedTables(t *testing.T) {
	content, err := fs.ReadFile(FS, "000002_identity.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	upper := strings.ToUpper(string(content))
	for _, table := range []string{
		"USERS",
		"TENANTS",
		"ORGANIZATIONS",
		"TENANT_MEMBERSHIPS",
		"ORGANIZATION_MEMBERSHIPS",
	} {
		if !strings.Contains(upper, "CREATE TABLE "+table) {
			t.Fatalf("identity migration does not create %s", table)
		}
	}
	down, err := fs.ReadFile(FS, "000002_identity.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToUpper(string(down)), "DROP TABLE") {
		t.Fatal("identity down migration does not drop tables")
	}
}

func TestSysPrefixMigrationRenamesTables(t *testing.T) {
	content, err := fs.ReadFile(FS, "000003_sys_prefix.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	upper := strings.ToUpper(string(content))
	for _, table := range []string{
		"SYS_USERS",
		"SYS_TENANTS",
		"SYS_ORGANIZATIONS",
		"SYS_TENANT_MEMBERSHIPS",
		"SYS_ORGANIZATION_MEMBERSHIPS",
	} {
		if !strings.Contains(upper, table) {
			t.Fatalf("sys prefix migration does not mention %s", table)
		}
	}
}

func TestRoleMigrationCreatesTables(t *testing.T) {
	content, err := fs.ReadFile(FS, "000007_sys_roles.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	upper := strings.ToUpper(string(content))
	for _, table := range []string{"CREATE TABLE SYS_ROLES", "CREATE TABLE SYS_ROLE_MENUS", "CREATE TABLE SYS_USER_ROLES"} {
		if !strings.Contains(upper, table) {
			t.Fatalf("role migration does not create %s", table)
		}
	}
}

func TestAuthDataScopeMigration(t *testing.T) {
	content, err := fs.ReadFile(FS, "000008_auth_data_scope.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	upper := strings.ToUpper(string(content))
	if !strings.Contains(upper, "DATA_SCOPE") {
		t.Fatal("auth migration does not add data_scope")
	}
	if !strings.Contains(upper, "CREATE TABLE SYS_ROLE_DEPTS") {
		t.Fatal("auth migration does not create sys_role_depts")
	}
}

func TestMenuCRUDButtonsMigration(t *testing.T) {
	content, err := fs.ReadFile(FS, "000009_menu_crud_buttons.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	for _, permission := range []string{
		"system:tenant:create",
		"system:organization:update",
		"system:post:delete",
		"system:role:query",
		"system:menu:create",
		"system:user:delete",
	} {
		if !strings.Contains(text, permission) {
			t.Fatalf("menu crud seed missing %s", permission)
		}
	}
	if !strings.Contains(text, "3, 1,") {
		t.Fatal("menu crud seed does not insert button type 3")
	}
}

func TestMenuSeedInsertsDefaultTree(t *testing.T) {
	content, err := fs.ReadFile(FS, "000006_sys_menus_seed.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	upper := strings.ToUpper(string(content))
	if !strings.Contains(upper, "INSERT INTO SYS_MENUS") {
		t.Fatal("menu seed does not insert sys_menus")
	}
	for _, name := range []string{"系统管理", "基础设施", "租户", "组织", "岗位", "菜单", "用户"} {
		if !strings.Contains(string(content), name) {
			t.Fatalf("menu seed does not include %s", name)
		}
	}
}

func TestMenuMigrationCreatesTable(t *testing.T) {
	content, err := fs.ReadFile(FS, "000005_sys_menus.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToUpper(string(content)), "CREATE TABLE SYS_MENUS") {
		t.Fatal("menu migration does not create sys_menus")
	}
}

func TestSecurityMigrationCreatesTables(t *testing.T) {
	content, err := fs.ReadFile(FS, "000011_sys_security.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	upper := strings.ToUpper(string(content))
	for _, table := range []string{
		"CREATE TABLE SYS_LOGIN_LOGS",
		"CREATE TABLE SYS_OPERATE_LOGS",
		"CREATE TABLE SYS_PERMISSION_LOGS",
		"CREATE TABLE SYS_SECURITY_POLICY",
	} {
		if !strings.Contains(upper, table) {
			t.Fatalf("security migration does not create %s", table)
		}
	}
	for _, permission := range []string{
		"system:operate-log:query",
		"system:login-log:query",
		"system:permission-log:query",
		"system:security-policy:update",
	} {
		if !strings.Contains(string(content), permission) {
			t.Fatalf("security migration missing %s", permission)
		}
	}
}

func TestDictMigrationCreatesTables(t *testing.T) {
	content, err := fs.ReadFile(FS, "000010_sys_dicts.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	upper := strings.ToUpper(string(content))
	for _, table := range []string{"CREATE TABLE SYS_DICT_TYPES", "CREATE TABLE SYS_DICT_DATA"} {
		if !strings.Contains(upper, table) {
			t.Fatalf("dict migration does not create %s", table)
		}
	}
	text := string(content)
	for _, permission := range []string{"system:dict:query", "system:dict:create", "system:dict:update", "system:dict:delete"} {
		if !strings.Contains(text, permission) {
			t.Fatalf("dict migration missing %s", permission)
		}
	}
	down, err := fs.ReadFile(FS, "000010_sys_dicts.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToUpper(string(down)), "DROP TABLE IF EXISTS SYS_DICT_TYPES") {
		t.Fatal("dict down migration does not drop sys_dict_types")
	}
}

func TestMonitorMigrationCreatesAlerts(t *testing.T) {
	content, err := fs.ReadFile(FS, "000012_sys_monitor.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if !strings.Contains(strings.ToUpper(text), "CREATE TABLE SYS_MONITOR_ALERTS") {
		t.Fatal("monitor migration does not create sys_monitor_alerts")
	}
	if !strings.Contains(text, "system:monitor:query") || !strings.Contains(text, "系统监控") {
		t.Fatal("monitor migration does not seed the menu")
	}
	down, err := fs.ReadFile(FS, "000012_sys_monitor.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToUpper(string(down)), "DROP TABLE IF EXISTS SYS_MONITOR_ALERTS") {
		t.Fatal("monitor down migration does not drop sys_monitor_alerts")
	}
}

func TestMenuNameMigrationRenamesPages(t *testing.T) {
	content, err := fs.ReadFile(FS, "000013_menu_names.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	for _, name := range []string{"租户管理", "组织管理", "岗位管理", "菜单管理", "用户管理", "角色管理", "字典管理"} {
		if !strings.Contains(text, name) {
			t.Fatalf("menu name migration missing %s", name)
		}
	}
	if !strings.Contains(text, "parent_id = NULL") {
		t.Fatal("menu name migration does not detach the home menu")
	}
	down, err := fs.ReadFile(FS, "000013_menu_names.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(down), "a1000000-0000-4000-8000-000000000001") {
		t.Fatal("menu name down migration does not restore the home parent")
	}
}

func TestSystemSettingsMigration(t *testing.T) {
	content, err := fs.ReadFile(FS, "000019_system_settings.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	upper := strings.ToUpper(text)
	if !strings.Contains(upper, "CREATE TABLE SYS_WEKNORA_SETTINGS") {
		t.Fatal("settings migration does not create sys_weknora_settings")
	}
	for _, permission := range []string{"system:settings:query", "system:settings:update", "系统配置"} {
		if !strings.Contains(text, permission) {
			t.Fatalf("settings migration missing %s", permission)
		}
	}
	if !strings.Contains(text, "super_admin") {
		t.Fatal("settings menu is not limited to super_admin")
	}
	down, err := fs.ReadFile(FS, "000019_system_settings.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToUpper(string(down)), "DROP TABLE IF EXISTS SYS_WEKNORA_SETTINGS") {
		t.Fatal("settings down migration does not drop the table")
	}
}

func TestModelMenuMovesIntoSettings(t *testing.T) {
	content, err := fs.ReadFile(FS, "000020_model_menu_into_settings.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	for _, piece := range []string{"model:query", "model:manage", "a1000000-0000-4000-8000-000000000020", "DELETE FROM sys_menus"} {
		if !strings.Contains(text, piece) {
			t.Fatalf("model menu migration missing %s", piece)
		}
	}
	down, err := fs.ReadFile(FS, "000020_model_menu_into_settings.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(down), "模型配置") {
		t.Fatal("model menu down migration does not restore the page")
	}
}

func TestPostMigrationCreatesTable(t *testing.T) {
	content, err := fs.ReadFile(FS, "000004_sys_posts.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToUpper(string(content)), "CREATE TABLE SYS_POSTS") {
		t.Fatal("post migration does not create sys_posts")
	}
}

func TestSSOOIDCRPMigration(t *testing.T) {
	content, err := fs.ReadFile(FS, "000023_sso_oidc_rp.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	upper := strings.ToUpper(string(content))
	for _, table := range []string{"SYS_SSO_OIDC_AUTH_REQUESTS", "SYS_SSO_LOGIN_TICKETS"} {
		if !strings.Contains(upper, "CREATE TABLE "+table) {
			t.Fatalf("oidc rp migration does not create %s", table)
		}
	}
	for _, col := range []string{"DEFAULT_TENANT_ID", "FRONTEND_BASE_URL"} {
		if !strings.Contains(upper, col) {
			t.Fatalf("oidc rp migration missing %s", col)
		}
	}
	down, err := fs.ReadFile(FS, "000023_sso_oidc_rp.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToUpper(string(down)), "DROP TABLE IF EXISTS SYS_SSO_LOGIN_TICKETS") {
		t.Fatal("oidc rp down migration incomplete")
	}
}

func TestSSOFoundationMigration(t *testing.T) {
	content, err := fs.ReadFile(FS, "000022_sso_foundation.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	upper := strings.ToUpper(string(content))
	for _, table := range []string{
		"SYS_SSO_SETTINGS",
		"SYS_SSO_SECRETS",
		"SYS_SSO_PROVIDERS",
		"SYS_SSO_OAUTH_CLIENTS",
		"SYS_SSO_SAML_SPS",
		"SYS_SSO_IDENTITY_LINKS",
		"SYS_SSO_AUTH_EVENTS",
	} {
		if !strings.Contains(upper, "CREATE TABLE "+table) {
			t.Fatalf("sso foundation migration does not create %s", table)
		}
	}
	text := string(content)
	for _, piece := range []string{"系统安全", "身份认证与单点登录", "system:sso:overview", "super_admin"} {
		if !strings.Contains(text, piece) {
			t.Fatalf("sso foundation migration missing %s", piece)
		}
	}
	down, err := fs.ReadFile(FS, "000022_sso_foundation.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToUpper(string(down)), "DROP TABLE IF EXISTS SYS_SSO_AUTH_EVENTS") {
		t.Fatal("sso foundation down migration does not drop tables")
	}
}
