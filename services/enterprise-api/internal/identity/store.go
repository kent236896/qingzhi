package identity

import (
	"context"
	"time"
)

// Store persists identity records. A transaction receives the same Store contract.
type Store interface {
	Transaction(ctx context.Context, fn func(Store) error) error

	InsertUser(ctx context.Context, user User) error
	UpdateUser(ctx context.Context, user User) error
	UpdateUserPassword(ctx context.Context, userID, passwordHash string) error
	UpdateUserLogin(ctx context.Context, userID, loginIP string, loginDate time.Time) error
	FindUser(ctx context.Context, id string) (User, error)
	FindUserByUsername(ctx context.Context, username string) (User, error)
	ListUsers(ctx context.Context, keyword string, page, pageSize int) ([]User, int64, error)

	InsertTenant(ctx context.Context, tenant Tenant) error
	UpdateTenant(ctx context.Context, tenant Tenant) error
	FindTenant(ctx context.Context, id string) (Tenant, error)
	ListTenants(ctx context.Context, keyword string, page, pageSize int) ([]Tenant, int64, error)

	InsertOrganization(ctx context.Context, org Organization) error
	UpdateOrganization(ctx context.Context, org Organization) error
	DeleteOrganization(ctx context.Context, id string) error
	FindOrganization(ctx context.Context, tenantID, id string) (Organization, error)
	FindOrganizationByID(ctx context.Context, id string) (Organization, error)
	ListOrganizations(ctx context.Context, tenantID string) ([]Organization, error)
	CountChildOrganizations(ctx context.Context, id string) (int64, error)
	CountOrganizationAssignments(ctx context.Context, organizationID string) (int64, error)

	InsertPost(ctx context.Context, post Post) error
	UpdatePost(ctx context.Context, post Post) error
	DeletePost(ctx context.Context, id string) error
	FindPost(ctx context.Context, tenantID, id string) (Post, error)
	ListPosts(ctx context.Context, tenantID, keyword string, page, pageSize int) ([]Post, int64, error)

	InsertMenu(ctx context.Context, menu Menu) error
	UpdateMenu(ctx context.Context, menu Menu) error
	DeleteMenu(ctx context.Context, id string) error
	FindMenu(ctx context.Context, id string) (Menu, error)
	ListMenus(ctx context.Context) ([]Menu, error)
	CountChildMenus(ctx context.Context, id string) (int64, error)

	InsertRole(ctx context.Context, role Role) error
	UpdateRole(ctx context.Context, role Role) error
	DeleteRole(ctx context.Context, id string) error
	FindRole(ctx context.Context, tenantID, id string) (Role, error)
	ListRoles(ctx context.Context, tenantID, keyword string, page, pageSize int) ([]Role, int64, error)
	ListRoleMenuIDs(ctx context.Context, tenantID, roleID string) ([]string, error)
	ReplaceRoleMenus(ctx context.Context, tenantID, roleID string, menuIDs []string) error
	ListRoleDeptIDs(ctx context.Context, tenantID, roleID string) ([]string, error)
	ReplaceRoleDepts(ctx context.Context, tenantID, roleID string, deptIDs []string) error
	ListRoleUsers(ctx context.Context, tenantID, roleID string) ([]RoleUserView, error)
	ReplaceRoleUsers(ctx context.Context, tenantID, roleID string, userIDs []string) error
	CountRoleUsers(ctx context.Context, roleID string) (int64, error)
	ListUserRoleIDs(ctx context.Context, tenantID, userID string) ([]string, error)
	ClearUserRoles(ctx context.Context, tenantID, userID string) error
	ListRolesByIDs(ctx context.Context, tenantID string, roleIDs []string) ([]Role, error)
	ListMenuIDsByRoleIDs(ctx context.Context, tenantID string, roleIDs []string) ([]string, error)
	ListMenusByIDs(ctx context.Context, menuIDs []string) ([]Menu, error)

	InsertMembership(ctx context.Context, membership TenantMembership) error
	DeleteMembership(ctx context.Context, tenantID, userID string) error
	FindMembership(ctx context.Context, tenantID, userID string) (TenantMembership, error)
	ListMembershipsByUser(ctx context.Context, userID string) ([]TenantMembership, error)
	ListMembers(ctx context.Context, tenantID, keyword string, page, pageSize int) ([]MemberRecord, int64, error)

	ReplaceAssignments(ctx context.Context, tenantID, userID string, assignments []OrganizationMembership) error
	ListAssignments(ctx context.Context, tenantID, userID string) ([]OrganizationMembership, error)
}

// MemberRecord is a membership joined to its account.
type MemberRecord struct {
	Membership TenantMembership
	Username   string
	Nickname   string
}
