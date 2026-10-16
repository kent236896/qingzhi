package identity

import "time"

const (
	// StatusNormal is an enabled account, tenant, department, membership, or menu.
	StatusNormal = 0
	// StatusDisabled is a disabled record.
	StatusDisabled = 1

	// MenuTypeDirectory is a navigation group.
	MenuTypeDirectory = 1
	// MenuTypeMenu is a routable page.
	MenuTypeMenu = 2
	// MenuTypeButton is an action identifier without its own page.
	MenuTypeButton = 3

	// RoleTypeSystem is a built-in role.
	RoleTypeSystem = 1
	// RoleTypeCustom is a tenant-defined role.
	RoleTypeCustom = 2
)

// User is a global account. PasswordHash is persisted but never serialized to clients.
type User struct {
	ID           string     `json:"id"`
	Username     string     `json:"username"`
	PasswordHash string     `json:"-"`
	Nickname     string     `json:"nickname"`
	Remark       string     `json:"remark"`
	Email        string     `json:"email"`
	Mobile       string     `json:"mobile"`
	Sex          int        `json:"sex"`
	Avatar       string     `json:"avatar"`
	Status       int        `json:"status"`
	LoginIP      string     `json:"login_ip"`
	LoginDate    *time.Time `json:"login_date"`
	Creator      string     `json:"creator"`
	CreateTime   time.Time  `json:"create_time"`
	Updater      string     `json:"updater"`
	UpdateTime   time.Time  `json:"update_time"`
	Deleted      bool       `json:"deleted"`
}

// Tenant is an isolation boundary for organizations and memberships.
type Tenant struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Code          string    `json:"code"`
	ContactUserID string    `json:"contact_user_id"`
	ContactName   string    `json:"contact_name"`
	ContactMobile string    `json:"contact_mobile"`
	Status        int       `json:"status"`
	AccountCount  int       `json:"account_count"`
	Creator       string    `json:"creator"`
	CreateTime    time.Time `json:"create_time"`
	Updater       string    `json:"updater"`
	UpdateTime    time.Time `json:"update_time"`
	Deleted       bool      `json:"deleted"`
}

// Organization is a department node in one tenant.
type Organization struct {
	ID           string    `json:"id"`
	TenantID     string    `json:"tenant_id"`
	ParentID     string    `json:"parent_id"`
	Name         string    `json:"name"`
	Code         string    `json:"code"`
	Sort         int       `json:"sort"`
	LeaderUserID string    `json:"leader_user_id"`
	Phone        string    `json:"phone"`
	Email        string    `json:"email"`
	Status       int       `json:"status"`
	Creator      string    `json:"creator"`
	CreateTime   time.Time `json:"create_time"`
	Updater      string    `json:"updater"`
	UpdateTime   time.Time `json:"update_time"`
	Deleted      bool      `json:"deleted"`
}

// Post is a job position inside one tenant.
type Post struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	Code       string    `json:"code"`
	Name       string    `json:"name"`
	Sort       int       `json:"sort"`
	Status     int       `json:"status"`
	Remark     string    `json:"remark"`
	Creator    string    `json:"creator"`
	CreateTime time.Time `json:"create_time"`
	Updater    string    `json:"updater"`
	UpdateTime time.Time `json:"update_time"`
	Deleted    bool      `json:"deleted"`
}

// Role is a tenant-scoped authorization label with menu and data scope settings.
type Role struct {
	ID         string    `json:"id"`
	TenantID   string    `json:"tenant_id"`
	Name       string    `json:"name"`
	Code       string    `json:"code"`
	Sort       int       `json:"sort"`
	Status     int       `json:"status"`
	Type       int       `json:"type"`
	DataScope  int       `json:"data_scope"`
	Remark     string    `json:"remark"`
	Creator    string    `json:"creator"`
	CreateTime time.Time `json:"create_time"`
	Updater    string    `json:"updater"`
	UpdateTime time.Time `json:"update_time"`
	Deleted    bool      `json:"deleted"`
}

// RoleDept links a custom data-scope role to an organization.
type RoleDept struct {
	ID         string
	TenantID   string
	RoleID     string
	DeptID     string
	Creator    string
	CreateTime time.Time
	Updater    string
	UpdateTime time.Time
	Deleted    bool
}

// RoleMenu links a role to a menu inside one tenant.
type RoleMenu struct {
	ID         string
	TenantID   string
	RoleID     string
	MenuID     string
	Creator    string
	CreateTime time.Time
	Updater    string
	UpdateTime time.Time
	Deleted    bool
}

// UserRole links a user to a role inside one tenant.
type UserRole struct {
	ID         string
	TenantID   string
	UserID     string
	RoleID     string
	Creator    string
	CreateTime time.Time
	Updater    string
	UpdateTime time.Time
	Deleted    bool
}

// RoleUserView is one account assigned to a role.
type RoleUserView struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Nickname string `json:"nickname"`
}

// Menu is a system navigation node. Permission is an identifier stored on the menu.
type Menu struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Permission    string    `json:"permission"`
	Type          int       `json:"type"`
	Sort          int       `json:"sort"`
	ParentID      string    `json:"parent_id"`
	Path          string    `json:"path"`
	Icon          string    `json:"icon"`
	Component     string    `json:"component"`
	ComponentName string    `json:"component_name"`
	Status        int       `json:"status"`
	Visible       bool      `json:"visible"`
	KeepAlive     bool      `json:"keep_alive"`
	AlwaysShow    bool      `json:"always_show"`
	Creator       string    `json:"creator"`
	CreateTime    time.Time `json:"create_time"`
	Updater       string    `json:"updater"`
	UpdateTime    time.Time `json:"update_time"`
	Deleted       bool      `json:"deleted"`
}

// MenuNode is one menu and its descendants.
type MenuNode struct {
	Menu
	Children []MenuNode `json:"children"`
}

// OrganizationNode is one department and its descendants.
type OrganizationNode struct {
	Organization
	Children []OrganizationNode `json:"children"`
}

// TenantMembership links an account to a tenant.
type TenantMembership struct {
	ID         string
	TenantID   string
	UserID     string
	Status     int
	CreateTime time.Time
	UpdateTime time.Time
}

// OrganizationMembership is one department assignment inside a tenant.
type OrganizationMembership struct {
	ID             string
	TenantID       string
	OrganizationID string
	UserID         string
	IsPrimary      bool
	CreateTime     time.Time
	UpdateTime     time.Time
}

// Assignment is a department shown on a membership.
type Assignment struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IsPrimary bool   `json:"is_primary"`
}

// UserMembership is the tenant association derived from organization assignments.
type UserMembership struct {
	TenantID             string       `json:"tenant_id"`
	TenantName           string       `json:"tenant_name"`
	TenantCode           string       `json:"tenant_code"`
	Status               int          `json:"status"`
	RootOrganizationID   string       `json:"root_organization_id"`
	RootOrganizationName string       `json:"root_organization_name"`
	Organizations        []Assignment `json:"organizations"`
}

// UserDetail includes the account and its tenant assignments.
type UserDetail struct {
	User
	Memberships []UserMembership `json:"memberships"`
}

// Member is an account as seen from one tenant.
type Member struct {
	UserID                string       `json:"user_id"`
	Username              string       `json:"username"`
	Nickname              string       `json:"nickname"`
	Status                int          `json:"status"`
	PrimaryOrganizationID string       `json:"primary_organization_id"`
	Organizations         []Assignment `json:"organizations"`
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
}

// AssignmentInput replaces the departments of one membership.
type AssignmentInput struct {
	OrganizationIDs       []string
	PrimaryOrganizationID string
}
