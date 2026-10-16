package identity

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// GormStore stores identity data in PostgreSQL.
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

func (s *GormStore) InsertUser(ctx context.Context, user User) error {
	return mapWrite(s.db.WithContext(ctx).Create(userRowFrom(user)).Error)
}

func (s *GormStore) UpdateUser(ctx context.Context, user User) error {
	return mapWrite(s.db.WithContext(ctx).Model(&userRow{}).Where("id = ? AND deleted = ?", user.ID, false).Updates(map[string]any{
		"nickname":    user.Nickname,
		"remark":      nullable(user.Remark),
		"email":       user.Email,
		"mobile":      user.Mobile,
		"sex":         user.Sex,
		"avatar":      user.Avatar,
		"status":      user.Status,
		"updater":     user.Updater,
		"update_time": user.UpdateTime,
	}).Error)
}

func (s *GormStore) UpdateUserPassword(ctx context.Context, userID, passwordHash string) error {
	result := s.db.WithContext(ctx).Model(&userRow{}).Where("id = ? AND deleted = ?", userID, false).Updates(map[string]any{
		"password":    passwordHash,
		"update_time": time.Now().UTC(),
	})
	if result.Error != nil {
		return mapWrite(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *GormStore) UpdateUserLogin(ctx context.Context, userID, loginIP string, loginDate time.Time) error {
	result := s.db.WithContext(ctx).Model(&userRow{}).Where("id = ? AND deleted = ?", userID, false).Updates(map[string]any{
		"login_ip":    loginIP,
		"login_date":  loginDate.UTC(),
		"update_time": time.Now().UTC(),
	})
	if result.Error != nil {
		return mapWrite(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *GormStore) FindUser(ctx context.Context, id string) (User, error) {
	var row userRow
	err := s.db.WithContext(ctx).First(&row, "id = ? AND deleted = ?", id, false).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	return row.user(), nil
}

func (s *GormStore) FindUserByUsername(ctx context.Context, username string) (User, error) {
	var row userRow
	err := s.db.WithContext(ctx).First(&row, "username = ? AND deleted = ?", username, false).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	return row.user(), nil
}

func (s *GormStore) ListUsers(ctx context.Context, keyword string, page, pageSize int) ([]User, int64, error) {
	query := applyKeyword(s.db.WithContext(ctx).Model(&userRow{}).Where("deleted = ?", false), keyword, "username", "nickname")
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []userRow
	err := query.Session(&gorm.Session{}).Order("create_time DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&rows).Error
	return usersFrom(rows), total, err
}

func (s *GormStore) InsertTenant(ctx context.Context, tenant Tenant) error {
	return mapWrite(s.db.WithContext(ctx).Create(tenantRowFrom(tenant)).Error)
}

func (s *GormStore) UpdateTenant(ctx context.Context, tenant Tenant) error {
	return mapWrite(s.db.WithContext(ctx).Model(&tenantRow{}).Where("id = ? AND deleted = ?", tenant.ID, false).Updates(map[string]any{
		"name":            tenant.Name,
		"contact_user_id": nullable(tenant.ContactUserID),
		"contact_name":    tenant.ContactName,
		"contact_mobile":  nullable(tenant.ContactMobile),
		"status":          tenant.Status,
		"account_count":   tenant.AccountCount,
		"updater":         tenant.Updater,
		"update_time":     tenant.UpdateTime,
	}).Error)
}

func (s *GormStore) FindTenant(ctx context.Context, id string) (Tenant, error) {
	var row tenantRow
	err := s.db.WithContext(ctx).First(&row, "id = ? AND deleted = ?", id, false).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Tenant{}, ErrNotFound
	}
	if err != nil {
		return Tenant{}, err
	}
	return row.tenant(), nil
}

func (s *GormStore) ListTenants(ctx context.Context, keyword string, page, pageSize int) ([]Tenant, int64, error) {
	query := applyKeyword(s.db.WithContext(ctx).Model(&tenantRow{}).Where("deleted = ?", false), keyword, "name", "code")
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []tenantRow
	err := query.Session(&gorm.Session{}).Order("create_time DESC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&rows).Error
	return tenantsFrom(rows), total, err
}

func (s *GormStore) InsertOrganization(ctx context.Context, org Organization) error {
	return mapWrite(s.db.WithContext(ctx).Create(organizationRowFrom(org)).Error)
}

func (s *GormStore) UpdateOrganization(ctx context.Context, org Organization) error {
	return mapWrite(s.db.WithContext(ctx).Model(&organizationRow{}).Where("id = ? AND deleted = ?", org.ID, false).Updates(map[string]any{
		"name":           org.Name,
		"code":           org.Code,
		"parent_id":      nullable(org.ParentID),
		"sort":           org.Sort,
		"leader_user_id": nullable(org.LeaderUserID),
		"phone":          nullable(org.Phone),
		"email":          nullable(org.Email),
		"status":         org.Status,
		"updater":        org.Updater,
		"update_time":    org.UpdateTime,
	}).Error)
}

func (s *GormStore) DeleteOrganization(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).Model(&organizationRow{}).Where("id = ? AND deleted = ?", id, false).Updates(map[string]any{
		"deleted":     true,
		"update_time": time.Now().UTC(),
	})
	if result.Error != nil {
		return mapWrite(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *GormStore) FindOrganizationByID(ctx context.Context, id string) (Organization, error) {
	var row organizationRow
	err := s.db.WithContext(ctx).First(&row, "id = ? AND deleted = ?", id, false).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Organization{}, ErrNotFound
	}
	if err != nil {
		return Organization{}, err
	}
	return row.organization(), nil
}

func (s *GormStore) FindOrganization(ctx context.Context, tenantID, id string) (Organization, error) {
	var row organizationRow
	err := s.db.WithContext(ctx).First(&row, "id = ? AND tenant_id = ? AND deleted = ?", id, tenantID, false).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Organization{}, ErrNotFound
	}
	if err != nil {
		return Organization{}, err
	}
	return row.organization(), nil
}

func (s *GormStore) ListOrganizations(ctx context.Context, tenantID string) ([]Organization, error) {
	var rows []organizationRow
	err := s.db.WithContext(ctx).Where("tenant_id = ? AND deleted = ?", tenantID, false).Order("sort ASC, name ASC").Find(&rows).Error
	return organizationsFrom(rows), err
}

func (s *GormStore) CountChildOrganizations(ctx context.Context, id string) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&organizationRow{}).Where("parent_id = ? AND deleted = ?", id, false).Count(&count).Error
	return count, err
}

func (s *GormStore) CountOrganizationAssignments(ctx context.Context, organizationID string) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&organizationMembershipRow{}).Where("organization_id = ?", organizationID).Count(&count).Error
	return count, err
}

func (s *GormStore) InsertPost(ctx context.Context, post Post) error {
	return mapWrite(s.db.WithContext(ctx).Create(postRowFrom(post)).Error)
}

func (s *GormStore) UpdatePost(ctx context.Context, post Post) error {
	return mapWrite(s.db.WithContext(ctx).Model(&postRow{}).Where("id = ? AND deleted = ?", post.ID, false).Updates(map[string]any{
		"code":        post.Code,
		"name":        post.Name,
		"sort":        post.Sort,
		"status":      post.Status,
		"remark":      nullable(post.Remark),
		"updater":     post.Updater,
		"update_time": post.UpdateTime,
	}).Error)
}

func (s *GormStore) DeletePost(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).Model(&postRow{}).Where("id = ? AND deleted = ?", id, false).Updates(map[string]any{
		"deleted":     true,
		"update_time": time.Now().UTC(),
	})
	if result.Error != nil {
		return mapWrite(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *GormStore) FindPost(ctx context.Context, tenantID, id string) (Post, error) {
	var row postRow
	err := s.db.WithContext(ctx).First(&row, "id = ? AND tenant_id = ? AND deleted = ?", id, tenantID, false).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Post{}, ErrNotFound
	}
	if err != nil {
		return Post{}, err
	}
	return row.post(), nil
}

func (s *GormStore) ListPosts(ctx context.Context, tenantID, keyword string, page, pageSize int) ([]Post, int64, error) {
	query := applyKeyword(s.db.WithContext(ctx).Model(&postRow{}).Where("tenant_id = ? AND deleted = ?", tenantID, false), keyword, "name", "code")
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []postRow
	err := query.Session(&gorm.Session{}).Order("sort ASC, name ASC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&rows).Error
	return postsFrom(rows), total, err
}

func (s *GormStore) InsertMembership(ctx context.Context, membership TenantMembership) error {
	return mapWrite(s.db.WithContext(ctx).Create(membershipRowFrom(membership)).Error)
}

func (s *GormStore) DeleteMembership(ctx context.Context, tenantID, userID string) error {
	result := s.db.WithContext(ctx).Delete(&tenantMembershipRow{}, "tenant_id = ? AND user_id = ?", tenantID, userID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *GormStore) FindMembership(ctx context.Context, tenantID, userID string) (TenantMembership, error) {
	var row tenantMembershipRow
	err := s.db.WithContext(ctx).First(&row, "tenant_id = ? AND user_id = ?", tenantID, userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return TenantMembership{}, ErrNotFound
	}
	if err != nil {
		return TenantMembership{}, err
	}
	return row.membership(), nil
}

func (s *GormStore) ListMembershipsByUser(ctx context.Context, userID string) ([]TenantMembership, error) {
	var rows []tenantMembershipRow
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).Order("create_time ASC").Find(&rows).Error
	items := make([]TenantMembership, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.membership())
	}
	return items, err
}

func (s *GormStore) ListMembers(ctx context.Context, tenantID, keyword string, page, pageSize int) ([]MemberRecord, int64, error) {
	query := applyKeyword(s.db.WithContext(ctx).Table("sys_tenant_memberships").
		Joins("JOIN sys_users ON sys_users.id = sys_tenant_memberships.user_id").
		Where("sys_tenant_memberships.tenant_id = ? AND sys_users.deleted = ?", tenantID, false), keyword, "sys_users.username", "sys_users.nickname")
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []memberJoinRow
	err := query.Session(&gorm.Session{}).
		Select("sys_tenant_memberships.id, sys_tenant_memberships.tenant_id, sys_tenant_memberships.user_id, sys_tenant_memberships.status, sys_tenant_memberships.create_time, sys_tenant_memberships.update_time, sys_users.username, sys_users.nickname").
		Order("sys_tenant_memberships.create_time DESC").
		Limit(pageSize).Offset((page - 1) * pageSize).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}
	items := make([]MemberRecord, 0, len(rows))
	for _, row := range rows {
		items = append(items, MemberRecord{
			Membership: TenantMembership{
				ID: row.ID, TenantID: row.TenantID, UserID: row.UserID, Status: row.Status,
				CreateTime: row.CreateTime.UTC(), UpdateTime: row.UpdateTime.UTC(),
			},
			Username: row.Username,
			Nickname: row.Nickname,
		})
	}
	return items, total, nil
}

func (s *GormStore) ReplaceAssignments(ctx context.Context, tenantID, userID string, assignments []OrganizationMembership) error {
	if err := s.db.WithContext(ctx).Delete(&organizationMembershipRow{}, "tenant_id = ? AND user_id = ?", tenantID, userID).Error; err != nil {
		return err
	}
	if len(assignments) == 0 {
		return nil
	}
	rows := make([]organizationMembershipRow, 0, len(assignments))
	for _, item := range assignments {
		rows = append(rows, organizationMembershipRowFrom(item))
	}
	return mapWrite(s.db.WithContext(ctx).Create(&rows).Error)
}

func (s *GormStore) ListAssignments(ctx context.Context, tenantID, userID string) ([]OrganizationMembership, error) {
	var rows []organizationMembershipRow
	err := s.db.WithContext(ctx).Where("tenant_id = ? AND user_id = ?", tenantID, userID).Order("create_time ASC").Find(&rows).Error
	items := make([]OrganizationMembership, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.membership())
	}
	return items, err
}

type userRow struct {
	ID         string     `gorm:"type:uuid;primaryKey"`
	Username   string     `gorm:"column:username"`
	Password   string     `gorm:"column:password"`
	Nickname   string     `gorm:"column:nickname"`
	Remark     *string    `gorm:"column:remark"`
	Email      string     `gorm:"column:email"`
	Mobile     string     `gorm:"column:mobile"`
	Sex        int        `gorm:"column:sex"`
	Avatar     string     `gorm:"column:avatar"`
	Status     int        `gorm:"column:status"`
	LoginIP    string     `gorm:"column:login_ip"`
	LoginDate  *time.Time `gorm:"column:login_date"`
	Creator    string     `gorm:"column:creator"`
	CreateTime time.Time  `gorm:"column:create_time"`
	Updater    string     `gorm:"column:updater"`
	UpdateTime time.Time  `gorm:"column:update_time"`
	Deleted    bool       `gorm:"column:deleted"`
}

func (userRow) TableName() string { return "sys_users" }

func userRowFrom(user User) userRow {
	return userRow{
		ID: user.ID, Username: user.Username, Password: user.PasswordHash, Nickname: user.Nickname,
		Remark: nullable(user.Remark), Email: user.Email, Mobile: user.Mobile, Sex: user.Sex, Avatar: user.Avatar,
		Status: user.Status, LoginIP: user.LoginIP, LoginDate: user.LoginDate, Creator: user.Creator,
		Updater: user.Updater, CreateTime: user.CreateTime, UpdateTime: user.UpdateTime, Deleted: user.Deleted,
	}
}

func (row userRow) user() User {
	return User{
		ID: row.ID, Username: row.Username, PasswordHash: row.Password, Nickname: row.Nickname,
		Remark: valueOf(row.Remark), Email: row.Email, Mobile: row.Mobile, Sex: row.Sex, Avatar: row.Avatar,
		Status: row.Status, LoginIP: row.LoginIP, LoginDate: utcPtr(row.LoginDate), Creator: row.Creator,
		Updater: row.Updater, CreateTime: row.CreateTime.UTC(), UpdateTime: row.UpdateTime.UTC(), Deleted: row.Deleted,
	}
}

type tenantRow struct {
	ID            string    `gorm:"type:uuid;primaryKey"`
	Name          string    `gorm:"column:name"`
	Code          string    `gorm:"column:code"`
	ContactUserID *string   `gorm:"column:contact_user_id"`
	ContactName   string    `gorm:"column:contact_name"`
	ContactMobile *string   `gorm:"column:contact_mobile"`
	Status        int       `gorm:"column:status"`
	AccountCount  int       `gorm:"column:account_count"`
	Creator       string    `gorm:"column:creator"`
	CreateTime    time.Time `gorm:"column:create_time"`
	Updater       string    `gorm:"column:updater"`
	UpdateTime    time.Time `gorm:"column:update_time"`
	Deleted       bool      `gorm:"column:deleted"`
}

func (tenantRow) TableName() string { return "sys_tenants" }

func tenantRowFrom(tenant Tenant) tenantRow {
	return tenantRow{
		ID: tenant.ID, Name: tenant.Name, Code: tenant.Code, ContactUserID: nullable(tenant.ContactUserID),
		ContactName: tenant.ContactName, ContactMobile: nullable(tenant.ContactMobile), Status: tenant.Status,
		AccountCount: tenant.AccountCount, Creator: tenant.Creator, Updater: tenant.Updater,
		CreateTime: tenant.CreateTime, UpdateTime: tenant.UpdateTime, Deleted: tenant.Deleted,
	}
}

func (row tenantRow) tenant() Tenant {
	return Tenant{
		ID: row.ID, Name: row.Name, Code: row.Code, ContactUserID: valueOf(row.ContactUserID),
		ContactName: row.ContactName, ContactMobile: valueOf(row.ContactMobile), Status: row.Status,
		AccountCount: row.AccountCount, Creator: row.Creator, Updater: row.Updater,
		CreateTime: row.CreateTime.UTC(), UpdateTime: row.UpdateTime.UTC(), Deleted: row.Deleted,
	}
}

type organizationRow struct {
	ID           string    `gorm:"type:uuid;primaryKey"`
	TenantID     string    `gorm:"column:tenant_id"`
	ParentID     *string   `gorm:"column:parent_id"`
	IsTenantRoot bool      `gorm:"column:is_tenant_root"`
	Name         string    `gorm:"column:name"`
	Code         string    `gorm:"column:code"`
	Sort         int       `gorm:"column:sort"`
	LeaderUserID *string   `gorm:"column:leader_user_id"`
	Phone        *string   `gorm:"column:phone"`
	Email        *string   `gorm:"column:email"`
	Status       int       `gorm:"column:status"`
	Creator      string    `gorm:"column:creator"`
	CreateTime   time.Time `gorm:"column:create_time"`
	Updater      string    `gorm:"column:updater"`
	UpdateTime   time.Time `gorm:"column:update_time"`
	Deleted      bool      `gorm:"column:deleted"`
}

func (organizationRow) TableName() string { return "sys_organizations" }

func organizationRowFrom(org Organization) organizationRow {
	return organizationRow{
		ID: org.ID, TenantID: org.TenantID, ParentID: nullable(org.ParentID), IsTenantRoot: org.IsTenantRoot, Name: org.Name,
		Code: org.Code, Sort: org.Sort, LeaderUserID: nullable(org.LeaderUserID), Phone: nullable(org.Phone),
		Email: nullable(org.Email), Status: org.Status, Creator: org.Creator, Updater: org.Updater,
		CreateTime: org.CreateTime, UpdateTime: org.UpdateTime, Deleted: org.Deleted,
	}
}

func (row organizationRow) organization() Organization {
	return Organization{
		ID: row.ID, TenantID: row.TenantID, ParentID: valueOf(row.ParentID), IsTenantRoot: row.IsTenantRoot, Name: row.Name,
		Code: row.Code, Sort: row.Sort, LeaderUserID: valueOf(row.LeaderUserID), Phone: valueOf(row.Phone),
		Email: valueOf(row.Email), Status: row.Status, Creator: row.Creator, Updater: row.Updater,
		CreateTime: row.CreateTime.UTC(), UpdateTime: row.UpdateTime.UTC(), Deleted: row.Deleted,
	}
}

type tenantMembershipRow struct {
	ID         string    `gorm:"type:uuid;primaryKey"`
	TenantID   string    `gorm:"column:tenant_id"`
	UserID     string    `gorm:"column:user_id"`
	Status     int       `gorm:"column:status"`
	CreateTime time.Time `gorm:"column:create_time"`
	UpdateTime time.Time `gorm:"column:update_time"`
}

func (tenantMembershipRow) TableName() string { return "sys_tenant_memberships" }

func membershipRowFrom(item TenantMembership) tenantMembershipRow {
	return tenantMembershipRow{ID: item.ID, TenantID: item.TenantID, UserID: item.UserID, Status: item.Status, CreateTime: item.CreateTime, UpdateTime: item.UpdateTime}
}

func (row tenantMembershipRow) membership() TenantMembership {
	return TenantMembership{ID: row.ID, TenantID: row.TenantID, UserID: row.UserID, Status: row.Status, CreateTime: row.CreateTime.UTC(), UpdateTime: row.UpdateTime.UTC()}
}

type organizationMembershipRow struct {
	ID             string    `gorm:"type:uuid;primaryKey"`
	TenantID       string    `gorm:"column:tenant_id"`
	OrganizationID string    `gorm:"column:organization_id"`
	UserID         string    `gorm:"column:user_id"`
	IsPrimary      bool      `gorm:"column:is_primary"`
	CreateTime     time.Time `gorm:"column:create_time"`
	UpdateTime     time.Time `gorm:"column:update_time"`
}

func (organizationMembershipRow) TableName() string { return "sys_organization_memberships" }

type postRow struct {
	ID         string    `gorm:"type:uuid;primaryKey"`
	TenantID   string    `gorm:"column:tenant_id"`
	Code       string    `gorm:"column:code"`
	Name       string    `gorm:"column:name"`
	Sort       int       `gorm:"column:sort"`
	Status     int       `gorm:"column:status"`
	Remark     *string   `gorm:"column:remark"`
	Creator    string    `gorm:"column:creator"`
	CreateTime time.Time `gorm:"column:create_time"`
	Updater    string    `gorm:"column:updater"`
	UpdateTime time.Time `gorm:"column:update_time"`
	Deleted    bool      `gorm:"column:deleted"`
}

func (postRow) TableName() string { return "sys_posts" }

type menuRow struct {
	ID            string    `gorm:"type:uuid;primaryKey"`
	Name          string    `gorm:"column:name"`
	Permission    string    `gorm:"column:permission"`
	Type          int       `gorm:"column:type"`
	Sort          int       `gorm:"column:sort"`
	ParentID      *string   `gorm:"column:parent_id"`
	Path          string    `gorm:"column:path"`
	Icon          string    `gorm:"column:icon"`
	Component     *string   `gorm:"column:component"`
	ComponentName *string   `gorm:"column:component_name"`
	Status        int       `gorm:"column:status"`
	Visible       bool      `gorm:"column:visible"`
	KeepAlive     bool      `gorm:"column:keep_alive"`
	AlwaysShow    bool      `gorm:"column:always_show"`
	Creator       string    `gorm:"column:creator"`
	CreateTime    time.Time `gorm:"column:create_time"`
	Updater       string    `gorm:"column:updater"`
	UpdateTime    time.Time `gorm:"column:update_time"`
	Deleted       bool      `gorm:"column:deleted"`
}

func (menuRow) TableName() string { return "sys_menus" }

func menuRowFrom(menu Menu) menuRow {
	return menuRow{
		ID: menu.ID, Name: menu.Name, Permission: menu.Permission, Type: menu.Type, Sort: menu.Sort,
		ParentID: nullable(menu.ParentID), Path: menu.Path, Icon: menu.Icon,
		Component: nullable(menu.Component), ComponentName: nullable(menu.ComponentName),
		Status: menu.Status, Visible: menu.Visible, KeepAlive: menu.KeepAlive, AlwaysShow: menu.AlwaysShow,
		Creator: menu.Creator, Updater: menu.Updater, CreateTime: menu.CreateTime, UpdateTime: menu.UpdateTime,
		Deleted: menu.Deleted,
	}
}

func (row menuRow) menu() Menu {
	return Menu{
		ID: row.ID, Name: row.Name, Permission: row.Permission, Type: row.Type, Sort: row.Sort,
		ParentID: valueOf(row.ParentID), Path: row.Path, Icon: row.Icon,
		Component: valueOf(row.Component), ComponentName: valueOf(row.ComponentName),
		Status: row.Status, Visible: row.Visible, KeepAlive: row.KeepAlive, AlwaysShow: row.AlwaysShow,
		Creator: row.Creator, Updater: row.Updater,
		CreateTime: row.CreateTime.UTC(), UpdateTime: row.UpdateTime.UTC(), Deleted: row.Deleted,
	}
}

func menusFrom(rows []menuRow) []Menu {
	items := make([]Menu, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.menu())
	}
	return items
}

func postRowFrom(post Post) postRow {
	return postRow{
		ID: post.ID, TenantID: post.TenantID, Code: post.Code, Name: post.Name, Sort: post.Sort,
		Status: post.Status, Remark: nullable(post.Remark), Creator: post.Creator, Updater: post.Updater,
		CreateTime: post.CreateTime, UpdateTime: post.UpdateTime, Deleted: post.Deleted,
	}
}

func (row postRow) post() Post {
	return Post{
		ID: row.ID, TenantID: row.TenantID, Code: row.Code, Name: row.Name, Sort: row.Sort, Status: row.Status,
		Remark: valueOf(row.Remark), Creator: row.Creator, Updater: row.Updater,
		CreateTime: row.CreateTime.UTC(), UpdateTime: row.UpdateTime.UTC(), Deleted: row.Deleted,
	}
}

func postsFrom(rows []postRow) []Post {
	items := make([]Post, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.post())
	}
	return items
}

func (s *GormStore) InsertMenu(ctx context.Context, menu Menu) error {
	return mapWrite(s.db.WithContext(ctx).Create(menuRowFrom(menu)).Error)
}

func (s *GormStore) UpdateMenu(ctx context.Context, menu Menu) error {
	return mapWrite(s.db.WithContext(ctx).Model(&menuRow{}).Where("id = ? AND deleted = ?", menu.ID, false).Updates(map[string]any{
		"name":           menu.Name,
		"permission":     menu.Permission,
		"type":           menu.Type,
		"sort":           menu.Sort,
		"parent_id":      nullable(menu.ParentID),
		"path":           menu.Path,
		"icon":           menu.Icon,
		"component":      nullable(menu.Component),
		"component_name": nullable(menu.ComponentName),
		"status":         menu.Status,
		"visible":        menu.Visible,
		"keep_alive":     menu.KeepAlive,
		"always_show":    menu.AlwaysShow,
		"updater":        menu.Updater,
		"update_time":    menu.UpdateTime,
	}).Error)
}

func (s *GormStore) DeleteMenu(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).Model(&menuRow{}).Where("id = ? AND deleted = ?", id, false).Updates(map[string]any{
		"deleted":     true,
		"update_time": time.Now().UTC(),
	})
	if result.Error != nil {
		return mapWrite(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *GormStore) FindMenu(ctx context.Context, id string) (Menu, error) {
	var row menuRow
	err := s.db.WithContext(ctx).First(&row, "id = ? AND deleted = ?", id, false).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Menu{}, ErrNotFound
	}
	if err != nil {
		return Menu{}, err
	}
	return row.menu(), nil
}

func (s *GormStore) ListMenus(ctx context.Context) ([]Menu, error) {
	var rows []menuRow
	err := s.db.WithContext(ctx).Where("deleted = ?", false).Order("sort ASC, name ASC").Find(&rows).Error
	return menusFrom(rows), err
}

func (s *GormStore) CountChildMenus(ctx context.Context, id string) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&menuRow{}).Where("parent_id = ? AND deleted = ?", id, false).Count(&count).Error
	return count, err
}

func (s *GormStore) InsertRole(ctx context.Context, role Role) error {
	return mapWrite(s.db.WithContext(ctx).Create(roleRowFrom(role)).Error)
}

func (s *GormStore) UpdateRole(ctx context.Context, role Role) error {
	return mapWrite(s.db.WithContext(ctx).Model(&roleRow{}).Where("id = ? AND deleted = ?", role.ID, false).Updates(map[string]any{
		"name":        role.Name,
		"code":        role.Code,
		"sort":        role.Sort,
		"status":      role.Status,
		"type":        role.Type,
		"data_scope":  role.DataScope,
		"remark":      nullable(role.Remark),
		"updater":     role.Updater,
		"update_time": role.UpdateTime,
	}).Error)
}

func (s *GormStore) DeleteRole(ctx context.Context, id string) error {
	result := s.db.WithContext(ctx).Model(&roleRow{}).Where("id = ? AND deleted = ?", id, false).Updates(map[string]any{
		"deleted":     true,
		"update_time": time.Now().UTC(),
	})
	if result.Error != nil {
		return mapWrite(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *GormStore) FindRole(ctx context.Context, tenantID, id string) (Role, error) {
	var row roleRow
	err := s.db.WithContext(ctx).First(&row, "id = ? AND tenant_id = ? AND deleted = ?", id, tenantID, false).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Role{}, ErrNotFound
	}
	if err != nil {
		return Role{}, err
	}
	return row.role(), nil
}

func (s *GormStore) ListRoles(ctx context.Context, tenantID, keyword string, page, pageSize int) ([]Role, int64, error) {
	query := applyKeyword(s.db.WithContext(ctx).Model(&roleRow{}).Where("tenant_id = ? AND deleted = ?", tenantID, false), keyword, "name", "code")
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []roleRow
	err := query.Session(&gorm.Session{}).Order("sort ASC, name ASC").Limit(pageSize).Offset((page - 1) * pageSize).Find(&rows).Error
	return rolesFrom(rows), total, err
}

func (s *GormStore) ListRoleMenuIDs(ctx context.Context, tenantID, roleID string) ([]string, error) {
	var ids []string
	err := s.db.WithContext(ctx).Model(&roleMenuRow{}).
		Where("tenant_id = ? AND role_id = ? AND deleted = ?", tenantID, roleID, false).
		Order("create_time ASC").Pluck("menu_id", &ids).Error
	return ids, err
}

func (s *GormStore) ReplaceRoleMenus(ctx context.Context, tenantID, roleID string, menuIDs []string) error {
	now := time.Now().UTC()
	if err := s.db.WithContext(ctx).Model(&roleMenuRow{}).
		Where("tenant_id = ? AND role_id = ? AND deleted = ?", tenantID, roleID, false).
		Updates(map[string]any{"deleted": true, "update_time": now}).Error; err != nil {
		return mapWrite(err)
	}
	for _, menuID := range menuIDs {
		row := roleMenuRow{
			ID: newID(), TenantID: tenantID, RoleID: roleID, MenuID: menuID,
			CreateTime: now, UpdateTime: now,
		}
		if err := mapWrite(s.db.WithContext(ctx).Create(&row).Error); err != nil {
			return err
		}
	}
	return nil
}

func (s *GormStore) ListRoleDeptIDs(ctx context.Context, tenantID, roleID string) ([]string, error) {
	var ids []string
	err := s.db.WithContext(ctx).Model(&roleDeptRow{}).
		Where("tenant_id = ? AND role_id = ? AND deleted = ?", tenantID, roleID, false).
		Order("create_time ASC").Pluck("dept_id", &ids).Error
	return ids, err
}

func (s *GormStore) ReplaceRoleDepts(ctx context.Context, tenantID, roleID string, deptIDs []string) error {
	now := time.Now().UTC()
	if err := s.db.WithContext(ctx).Model(&roleDeptRow{}).
		Where("tenant_id = ? AND role_id = ? AND deleted = ?", tenantID, roleID, false).
		Updates(map[string]any{"deleted": true, "update_time": now}).Error; err != nil {
		return mapWrite(err)
	}
	for _, deptID := range deptIDs {
		row := roleDeptRow{
			ID: newID(), TenantID: tenantID, RoleID: roleID, DeptID: deptID,
			CreateTime: now, UpdateTime: now,
		}
		if err := mapWrite(s.db.WithContext(ctx).Create(&row).Error); err != nil {
			return err
		}
	}
	return nil
}

func (s *GormStore) ListUserRoleIDs(ctx context.Context, tenantID, userID string) ([]string, error) {
	var ids []string
	err := s.db.WithContext(ctx).Model(&userRoleRow{}).
		Where("tenant_id = ? AND user_id = ? AND deleted = ?", tenantID, userID, false).
		Order("create_time ASC").Pluck("role_id", &ids).Error
	return ids, err
}

func (s *GormStore) ListRolesByIDs(ctx context.Context, tenantID string, roleIDs []string) ([]Role, error) {
	if len(roleIDs) == 0 {
		return nil, nil
	}
	var rows []roleRow
	err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND id IN ? AND deleted = ? AND status = ?", tenantID, roleIDs, false, StatusNormal).
		Order("sort ASC, name ASC").Find(&rows).Error
	return rolesFrom(rows), err
}

func (s *GormStore) ListMenuIDsByRoleIDs(ctx context.Context, tenantID string, roleIDs []string) ([]string, error) {
	if len(roleIDs) == 0 {
		return nil, nil
	}
	var ids []string
	err := s.db.WithContext(ctx).Model(&roleMenuRow{}).
		Where("tenant_id = ? AND role_id IN ? AND deleted = ?", tenantID, roleIDs, false).
		Distinct("menu_id").Pluck("menu_id", &ids).Error
	return ids, err
}

func (s *GormStore) ListMenusByIDs(ctx context.Context, menuIDs []string) ([]Menu, error) {
	if len(menuIDs) == 0 {
		return nil, nil
	}
	var rows []menuRow
	err := s.db.WithContext(ctx).
		Where("id IN ? AND deleted = ?", menuIDs, false).
		Order("sort ASC, name ASC").Find(&rows).Error
	return menusFrom(rows), err
}

func (s *GormStore) ListRoleUsers(ctx context.Context, tenantID, roleID string) ([]RoleUserView, error) {
	var rows []roleUserJoinRow
	err := s.db.WithContext(ctx).Table("sys_user_roles AS ur").
		Select("ur.user_id, u.username, u.nickname").
		Joins("JOIN sys_users AS u ON u.id = ur.user_id AND u.deleted = ?", false).
		Where("ur.tenant_id = ? AND ur.role_id = ? AND ur.deleted = ?", tenantID, roleID, false).
		Order("u.username ASC").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	items := make([]RoleUserView, 0, len(rows))
	for _, row := range rows {
		items = append(items, RoleUserView{UserID: row.UserID, Username: row.Username, Nickname: row.Nickname})
	}
	return items, nil
}

func (s *GormStore) ReplaceRoleUsers(ctx context.Context, tenantID, roleID string, userIDs []string) error {
	now := time.Now().UTC()
	if err := s.db.WithContext(ctx).Model(&userRoleRow{}).
		Where("tenant_id = ? AND role_id = ? AND deleted = ?", tenantID, roleID, false).
		Updates(map[string]any{"deleted": true, "update_time": now}).Error; err != nil {
		return mapWrite(err)
	}
	for _, userID := range userIDs {
		row := userRoleRow{
			ID: newID(), TenantID: tenantID, UserID: userID, RoleID: roleID,
			CreateTime: now, UpdateTime: now,
		}
		if err := mapWrite(s.db.WithContext(ctx).Create(&row).Error); err != nil {
			return err
		}
	}
	return nil
}

func (s *GormStore) ClearUserRoles(ctx context.Context, tenantID, userID string) error {
	now := time.Now().UTC()
	return mapWrite(s.db.WithContext(ctx).Model(&userRoleRow{}).
		Where("tenant_id = ? AND user_id = ? AND deleted = ?", tenantID, userID, false).
		Updates(map[string]any{"deleted": true, "update_time": now}).Error)
}

func (s *GormStore) CountRoleUsers(ctx context.Context, roleID string) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&userRoleRow{}).Where("role_id = ? AND deleted = ?", roleID, false).Count(&count).Error
	return count, err
}

type roleRow struct {
	ID         string    `gorm:"type:uuid;primaryKey"`
	TenantID   string    `gorm:"column:tenant_id"`
	Name       string    `gorm:"column:name"`
	Code       string    `gorm:"column:code"`
	Sort       int       `gorm:"column:sort"`
	Status     int       `gorm:"column:status"`
	Type       int       `gorm:"column:type"`
	DataScope  int       `gorm:"column:data_scope"`
	Remark     *string   `gorm:"column:remark"`
	Creator    string    `gorm:"column:creator"`
	CreateTime time.Time `gorm:"column:create_time"`
	Updater    string    `gorm:"column:updater"`
	UpdateTime time.Time `gorm:"column:update_time"`
	Deleted    bool      `gorm:"column:deleted"`
}

func (roleRow) TableName() string { return "sys_roles" }

func roleRowFrom(role Role) roleRow {
	return roleRow{
		ID: role.ID, TenantID: role.TenantID, Name: role.Name, Code: role.Code, Sort: role.Sort,
		Status: role.Status, Type: role.Type, DataScope: role.DataScope, Remark: nullable(role.Remark),
		Creator: role.Creator, Updater: role.Updater, CreateTime: role.CreateTime, UpdateTime: role.UpdateTime,
		Deleted: role.Deleted,
	}
}

func (row roleRow) role() Role {
	return Role{
		ID: row.ID, TenantID: row.TenantID, Name: row.Name, Code: row.Code, Sort: row.Sort,
		Status: row.Status, Type: row.Type, DataScope: row.DataScope, Remark: valueOf(row.Remark),
		Creator: row.Creator, Updater: row.Updater, CreateTime: row.CreateTime.UTC(),
		UpdateTime: row.UpdateTime.UTC(), Deleted: row.Deleted,
	}
}

type roleDeptRow struct {
	ID         string    `gorm:"type:uuid;primaryKey"`
	TenantID   string    `gorm:"column:tenant_id"`
	RoleID     string    `gorm:"column:role_id"`
	DeptID     string    `gorm:"column:dept_id"`
	Creator    string    `gorm:"column:creator"`
	CreateTime time.Time `gorm:"column:create_time"`
	Updater    string    `gorm:"column:updater"`
	UpdateTime time.Time `gorm:"column:update_time"`
	Deleted    bool      `gorm:"column:deleted"`
}

func (roleDeptRow) TableName() string { return "sys_role_depts" }

func rolesFrom(rows []roleRow) []Role {
	items := make([]Role, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.role())
	}
	return items
}

type roleMenuRow struct {
	ID         string    `gorm:"type:uuid;primaryKey"`
	TenantID   string    `gorm:"column:tenant_id"`
	RoleID     string    `gorm:"column:role_id"`
	MenuID     string    `gorm:"column:menu_id"`
	Creator    string    `gorm:"column:creator"`
	CreateTime time.Time `gorm:"column:create_time"`
	Updater    string    `gorm:"column:updater"`
	UpdateTime time.Time `gorm:"column:update_time"`
	Deleted    bool      `gorm:"column:deleted"`
}

func (roleMenuRow) TableName() string { return "sys_role_menus" }

type userRoleRow struct {
	ID         string    `gorm:"type:uuid;primaryKey"`
	TenantID   string    `gorm:"column:tenant_id"`
	UserID     string    `gorm:"column:user_id"`
	RoleID     string    `gorm:"column:role_id"`
	Creator    string    `gorm:"column:creator"`
	CreateTime time.Time `gorm:"column:create_time"`
	Updater    string    `gorm:"column:updater"`
	UpdateTime time.Time `gorm:"column:update_time"`
	Deleted    bool      `gorm:"column:deleted"`
}

func (userRoleRow) TableName() string { return "sys_user_roles" }

type roleUserJoinRow struct {
	UserID   string
	Username string
	Nickname string
}

func organizationMembershipRowFrom(item OrganizationMembership) organizationMembershipRow {
	return organizationMembershipRow{
		ID: item.ID, TenantID: item.TenantID, OrganizationID: item.OrganizationID, UserID: item.UserID,
		IsPrimary: item.IsPrimary, CreateTime: item.CreateTime, UpdateTime: item.UpdateTime,
	}
}

func (row organizationMembershipRow) membership() OrganizationMembership {
	return OrganizationMembership{
		ID: row.ID, TenantID: row.TenantID, OrganizationID: row.OrganizationID, UserID: row.UserID,
		IsPrimary: row.IsPrimary, CreateTime: row.CreateTime.UTC(), UpdateTime: row.UpdateTime.UTC(),
	}
}

type memberJoinRow struct {
	ID         string
	TenantID   string
	UserID     string
	Status     int
	CreateTime time.Time
	UpdateTime time.Time
	Username   string
	Nickname   string
}

func utcPtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	utc := value.UTC()
	return &utc
}

func nullable(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func valueOf(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func usersFrom(rows []userRow) []User {
	items := make([]User, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.user())
	}
	return items
}

func tenantsFrom(rows []tenantRow) []Tenant {
	items := make([]Tenant, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.tenant())
	}
	return items
}

func organizationsFrom(rows []organizationRow) []Organization {
	items := make([]Organization, 0, len(rows))
	for _, row := range rows {
		items = append(items, row.organization())
	}
	return items
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

func mapWrite(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return conflict(uniqueMessage(pgErr.ConstraintName))
	}
	return err
}

func uniqueMessage(constraint string) string {
	switch constraint {
	case "sys_users_username_unique":
		return "用户名已存在"
	case "sys_users_email_unique":
		return "邮箱已存在"
	case "sys_users_mobile_unique":
		return "手机号已存在"
	case "sys_tenants_code_unique":
		return "租户编码已存在"
	case "sys_posts_code_unique":
		return "岗位编码已存在"
	case "sys_roles_code_unique":
		return "角色权限字符串已存在"
	case "sys_role_menus_unique":
		return "角色菜单已存在"
	case "sys_user_roles_unique":
		return "角色用户已存在"
	case "sys_organizations_code_unique":
		return "部门编码已存在"
	case "sys_organizations_root_name", "sys_organizations_sibling_name":
		return "同一上级下部门名称已存在"
	case "tenant_memberships_unique":
		return "账号已在该租户中"
	case "organization_memberships_unique":
		return "任职部门重复"
	case "organization_memberships_one_primary":
		return "同一租户只能有一个主部门"
	default:
		return "数据已存在"
	}
}
