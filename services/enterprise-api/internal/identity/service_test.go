package identity

import (
	"context"
	"errors"
	"testing"
)

func TestTenantOrganizationAndMembershipRules(t *testing.T) {
	svc := NewService(newMemoryStore())
	ctx := context.Background()

	tenant, err := svc.CreateTenant(ctx, CreateTenantInput{Name: "擎知", Code: "Qing-Zhi", ContactName: "联系人"})
	if err != nil {
		t.Fatal(err)
	}
	if tenant.Code != "qing-zhi" {
		t.Fatalf("code = %s", tenant.Code)
	}
	if _, err := svc.CreateTenant(ctx, CreateTenantInput{Name: "重复", Code: "qing-zhi", ContactName: "联系人"}); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate tenant error = %v", err)
	}

	root, err := onlyRoot(t, svc, tenant.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateOrganization(ctx, tenant.ID, CreateOrganizationInput{Name: "第二个主体", Code: "HQ"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("second root error = %v", err)
	}
	child, err := svc.CreateOrganization(ctx, tenant.ID, CreateOrganizationInput{Name: "研发", Code: "RD", ParentID: root.ID})
	if err != nil {
		t.Fatal(err)
	}
	parentID := child.ID
	if _, err := svc.UpdateOrganization(ctx, tenant.ID, root.ID, UpdateOrganizationInput{ParentSet: true, ParentID: &parentID}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("cycle error = %v", err)
	}

	user, err := svc.CreateUser(ctx, CreateUserInput{Username: "ada", Password: "Passw0rd!", Nickname: "艾达", Email: "ada@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	member, err := svc.CreateMember(ctx, tenant.ID, CreateMemberInput{
		UserID: user.ID,
		AssignmentInput: AssignmentInput{
			OrganizationIDs:       []string{root.ID, child.ID},
			PrimaryOrganizationID: child.ID,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if member.PrimaryOrganizationID != child.ID || len(member.Organizations) != 2 {
		t.Fatalf("member = %+v", member)
	}
	if _, err := svc.CreateMember(ctx, tenant.ID, CreateMemberInput{
		UserID: user.ID,
		AssignmentInput: AssignmentInput{
			OrganizationIDs:       []string{root.ID},
			PrimaryOrganizationID: root.ID,
		},
	}); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate member error = %v", err)
	}
	if err := svc.DeleteOrganization(ctx, tenant.ID, child.ID); !errors.Is(err, ErrConflict) {
		t.Fatalf("delete assigned organization error = %v", err)
	}

	replaced, err := svc.ReplaceMemberAssignments(ctx, tenant.ID, user.ID, AssignmentInput{
		OrganizationIDs:       []string{root.ID},
		PrimaryOrganizationID: root.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if replaced.PrimaryOrganizationID != root.ID || len(replaced.Organizations) != 1 {
		t.Fatalf("replaced = %+v", replaced)
	}
	if err := svc.RemoveMember(ctx, tenant.ID, user.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteOrganization(ctx, tenant.ID, child.ID); err != nil {
		t.Fatal(err)
	}

	detail, err := svc.GetUser(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Memberships) != 0 {
		t.Fatalf("memberships = %+v", detail.Memberships)
	}
	post, err := svc.CreatePost(ctx, tenant.ID, CreatePostInput{Code: "PM", Name: "产品经理", Sort: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreatePost(ctx, tenant.ID, CreatePostInput{Code: "PM", Name: "重复岗位"}); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate post error = %v", err)
	}
	if err := svc.DeletePost(ctx, tenant.ID, post.ID); err != nil {
		t.Fatal(err)
	}
	directory, err := svc.CreateMenu(ctx, CreateMenuInput{Name: "系统管理", Type: MenuTypeDirectory, Sort: 1})
	if err != nil {
		t.Fatal(err)
	}
	if directory.Icon != "#" || !directory.Visible || !directory.KeepAlive || !directory.AlwaysShow {
		t.Fatalf("menu defaults = %+v", directory)
	}
	childMenu, err := svc.CreateMenu(ctx, CreateMenuInput{Name: "用户管理", Type: MenuTypeMenu, ParentID: directory.ID, Path: "users"})
	if err != nil {
		t.Fatal(err)
	}
	menuParentID := childMenu.ID
	if _, err := svc.UpdateMenu(ctx, directory.ID, UpdateMenuInput{ParentSet: true, ParentID: &menuParentID}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("menu cycle error = %v", err)
	}
	if err := svc.DeleteMenu(ctx, directory.ID); !errors.Is(err, ErrConflict) {
		t.Fatalf("delete parent menu error = %v", err)
	}
	if err := svc.DeleteMenu(ctx, childMenu.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteMenu(ctx, directory.ID); err != nil {
		t.Fatal(err)
	}
	role, err := svc.CreateRole(ctx, tenant.ID, CreateRoleInput{Name: "管理员", Code: "tenant-admin", Sort: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateRole(ctx, tenant.ID, CreateRoleInput{Name: "重复", Code: "tenant-admin"}); !errors.Is(err, ErrConflict) {
		t.Fatalf("duplicate role error = %v", err)
	}
	menu, err := svc.CreateMenu(ctx, CreateMenuInput{Name: "角色页", Type: MenuTypeMenu, Path: "roles"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ReplaceRoleMenus(ctx, tenant.ID, role.ID, []string{menu.ID}); err != nil {
		t.Fatal(err)
	}
	memberUser, err := svc.CreateUser(ctx, CreateUserInput{Username: "role.user", Password: "Passw0rd!", Nickname: "角色用户"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateMember(ctx, tenant.ID, CreateMemberInput{
		UserID: memberUser.ID,
		AssignmentInput: AssignmentInput{
			OrganizationIDs:       []string{root.ID},
			PrimaryOrganizationID: root.ID,
		},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ReplaceRoleUsers(ctx, tenant.ID, role.ID, []string{memberUser.ID}); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteRole(ctx, tenant.ID, role.ID); !errors.Is(err, ErrConflict) {
		t.Fatalf("delete assigned role error = %v", err)
	}
	if _, err := svc.ReplaceRoleUsers(ctx, tenant.ID, role.ID, nil); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteRole(ctx, tenant.ID, role.ID); err != nil {
		t.Fatal(err)
	}
	tree, err := svc.ListOrganizations(ctx, tenant.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tree) != 1 || tree[0].ID != root.ID || len(tree[0].Children) != 0 {
		t.Fatalf("tree = %+v", tree)
	}
}

func TestCrossTenantOrganizationAssignment(t *testing.T) {
	svc := NewService(newMemoryStore())
	ctx := context.Background()
	tenantA, err := svc.CreateTenant(ctx, CreateTenantInput{Name: "甲公司", Code: "tenant-a", ContactName: "甲"})
	if err != nil {
		t.Fatal(err)
	}
	tenantB, err := svc.CreateTenant(ctx, CreateTenantInput{Name: "乙公司", Code: "tenant-b", ContactName: "乙"})
	if err != nil {
		t.Fatal(err)
	}
	rootA, err := onlyRoot(t, svc, tenantA.ID)
	if err != nil {
		t.Fatal(err)
	}
	rootB, err := onlyRoot(t, svc, tenantB.ID)
	if err != nil {
		t.Fatal(err)
	}
	if rootA.Name != tenantA.Name || rootB.Name != tenantB.Name {
		t.Fatalf("roots = %s %s", rootA.Name, rootB.Name)
	}
	dept, err := svc.CreateOrganization(ctx, tenantA.ID, CreateOrganizationInput{Name: "研发", Code: "RD", ParentID: rootA.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateOrganization(ctx, tenantA.ID, CreateOrganizationInput{Name: "跨租户", Code: "X", ParentID: rootB.ID}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("cross-tenant department error = %v", err)
	}
	placed, err := svc.UpdateOrganization(ctx, tenantB.ID, rootB.ID, UpdateOrganizationInput{ParentSet: true, ParentID: &rootA.ID})
	if err != nil {
		t.Fatal(err)
	}
	if placed.ParentID != rootA.ID || !placed.IsTenantRoot || placed.TenantID != tenantB.ID {
		t.Fatalf("subsidiary = %+v", placed)
	}
	if _, err := svc.UpdateOrganization(ctx, tenantA.ID, rootA.ID, UpdateOrganizationInput{ParentSet: true, ParentID: &rootB.ID}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("organization cycle error = %v", err)
	}
	if _, err := svc.UpdateOrganization(ctx, tenantB.ID, rootB.ID, UpdateOrganizationInput{ParentSet: true, ParentID: &dept.ID}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("subsidiary under department error = %v", err)
	}
	clearedParent := ""
	lifted, err := svc.UpdateOrganization(ctx, tenantB.ID, rootB.ID, UpdateOrganizationInput{ParentSet: true, ParentID: &clearedParent})
	if err != nil || lifted.ParentID != "" || !lifted.IsTenantRoot {
		t.Fatalf("lifted = %+v err=%v", lifted, err)
	}
	cleared := ""
	if _, err := svc.UpdateOrganization(ctx, tenantA.ID, dept.ID, UpdateOrganizationInput{ParentSet: true, ParentID: &cleared}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("promote department error = %v", err)
	}
	if err := svc.DeleteOrganization(ctx, tenantA.ID, rootA.ID); !errors.Is(err, ErrConflict) {
		t.Fatalf("delete root error = %v", err)
	}

	user, err := svc.CreateUser(ctx, CreateUserInput{Username: "cross.user", Password: "Passw0rd!", Nickname: "跨租户用户"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateMember(ctx, tenantA.ID, CreateMemberInput{
		UserID: user.ID,
		AssignmentInput: AssignmentInput{
			OrganizationIDs:       []string{rootA.ID, dept.ID},
			PrimaryOrganizationID: dept.ID,
		},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateMember(ctx, tenantB.ID, CreateMemberInput{
		UserID: user.ID,
		AssignmentInput: AssignmentInput{
			OrganizationIDs:       []string{rootB.ID},
			PrimaryOrganizationID: rootB.ID,
		},
	}); err != nil {
		t.Fatal(err)
	}
	role, err := svc.CreateRole(ctx, tenantA.ID, CreateRoleInput{Name: "甲角色", Code: "role-a"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ReplaceRoleUsers(ctx, tenantA.ID, role.ID, []string{user.ID}); err != nil {
		t.Fatal(err)
	}

	detail, err := svc.GetUser(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Memberships) != 2 {
		t.Fatalf("memberships = %+v", detail.Memberships)
	}

	replaced, err := svc.ReplaceMemberAssignments(ctx, tenantA.ID, user.ID, AssignmentInput{
		OrganizationIDs:       []string{dept.ID},
		PrimaryOrganizationID: dept.ID,
	})
	if err != nil || len(replaced.Organizations) != 1 {
		t.Fatalf("replaced = %+v err=%v", replaced, err)
	}
	if _, err := svc.ReplaceMemberAssignments(ctx, tenantA.ID, user.ID, AssignmentInput{}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("empty assignment error = %v", err)
	}
	if err := svc.RemoveMember(ctx, tenantA.ID, user.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteRole(ctx, tenantA.ID, role.ID); err != nil {
		t.Fatalf("role should be removable after the last assignment is removed: %v", err)
	}

	detail, err = svc.GetUser(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Username != user.Username || len(detail.Memberships) != 1 || detail.Memberships[0].TenantID != tenantB.ID {
		t.Fatalf("remaining identity = %+v", detail)
	}
	if detail.Memberships[0].RootOrganizationID != rootB.ID || len(detail.Memberships[0].Organizations) != 1 {
		t.Fatalf("tenant B assignment = %+v", detail.Memberships[0])
	}
}

func TestOwnProfileAndPassword(t *testing.T) {
	svc := NewService(newMemoryStore())
	ctx := context.Background()
	user, err := svc.CreateUser(ctx, CreateUserInput{Username: "ada", Password: "Passw0rd!", Nickname: "艾达"})
	if err != nil {
		t.Fatal(err)
	}
	nickname := "艾达·洛夫莱斯"
	email := "ada@example.com"
	updated, err := svc.UpdateOwnProfile(ctx, user.ID, UpdateUserInput{Nickname: &nickname, Email: &email})
	if err != nil || updated.Nickname != nickname || updated.Email != email || updated.Status != user.Status {
		t.Fatalf("profile = %+v err=%v", updated, err)
	}
	if err := svc.ChangeOwnPassword(ctx, user.ID, "wrong", "NewPass1!"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("wrong password error = %v", err)
	}
	if err := svc.ChangeOwnPassword(ctx, user.ID, "Passw0rd!", "NewPass1!"); err != nil {
		t.Fatal(err)
	}
	if err := svc.ChangeOwnPassword(ctx, user.ID, "NewPass1!", "Newer1!"); err != nil {
		t.Fatal(err)
	}
	if err := svc.ChangeOwnPassword(ctx, user.ID, "Passw0rd!", "Nope1!"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("old password error = %v", err)
	}
	if !updated.LocalPasswordEnabled {
		t.Fatal("local users must have local password enabled")
	}
}

func TestFederatedUserCannotUseLocalPassword(t *testing.T) {
	svc := NewService(newMemoryStore())
	ctx := context.Background()
	user, err := svc.CreateFederatedUser(ctx, "fed.user", "联邦用户", "fed@example.com", "")
	if err != nil {
		t.Fatal(err)
	}
	profile, err := svc.GetOwnProfile(ctx, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if profile.LocalPasswordEnabled {
		t.Fatal("federated users must not enable local password")
	}
	if err := svc.ChangeOwnPassword(ctx, user.ID, "anything", "NewPass1!"); !errors.Is(err, ErrForbidden) {
		t.Fatalf("change password error = %v", err)
	}
	if _, err := svc.Authenticate(ctx, LoginInput{Username: "fed.user", Password: "anything"}); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("authenticate error = %v", err)
	}
}

func onlyRoot(t *testing.T, svc *Service, tenantID string) (OrganizationNode, error) {
	t.Helper()
	tree, err := svc.ListOrganizations(context.Background(), tenantID)
	if err != nil {
		return OrganizationNode{}, err
	}
	if len(tree) != 1 || !tree[0].IsTenantRoot {
		t.Fatalf("root tree = %+v", tree)
	}
	return tree[0], nil
}
