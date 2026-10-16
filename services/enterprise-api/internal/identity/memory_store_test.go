package identity

import (
	"context"
	"sort"
	"strings"
	"time"
)

type memoryStore struct {
	users       map[string]User
	tenants     map[string]Tenant
	orgs        map[string]Organization
	posts       map[string]Post
	menus       map[string]Menu
	roles       map[string]Role
	roleMenus   map[string]RoleMenu
	roleDepts   map[string]RoleDept
	userRoles   map[string]UserRole
	members     map[string]TenantMembership
	assignments map[string]OrganizationMembership
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		users:       map[string]User{},
		tenants:     map[string]Tenant{},
		orgs:        map[string]Organization{},
		posts:       map[string]Post{},
		menus:       map[string]Menu{},
		roles:       map[string]Role{},
		roleMenus:   map[string]RoleMenu{},
		roleDepts:   map[string]RoleDept{},
		userRoles:   map[string]UserRole{},
		members:     map[string]TenantMembership{},
		assignments: map[string]OrganizationMembership{},
	}
}

func (s *memoryStore) Transaction(_ context.Context, fn func(Store) error) error {
	snap := s.clone()
	if err := fn(s); err != nil {
		*s = *snap
		return err
	}
	return nil
}

func (s *memoryStore) clone() *memoryStore {
	return &memoryStore{
		users: cloneMap(s.users), tenants: cloneMap(s.tenants), orgs: cloneMap(s.orgs), posts: cloneMap(s.posts), menus: cloneMap(s.menus),
		roles: cloneMap(s.roles), roleMenus: cloneMap(s.roleMenus), roleDepts: cloneMap(s.roleDepts), userRoles: cloneMap(s.userRoles),
		members: cloneMap(s.members), assignments: cloneMap(s.assignments),
	}
}

func (s *memoryStore) InsertUser(_ context.Context, user User) error {
	for _, existing := range s.users {
		if existing.Deleted {
			continue
		}
		if existing.Username == user.Username {
			return conflict("用户名已存在")
		}
		if user.Email != "" && existing.Email == user.Email {
			return conflict("邮箱已存在")
		}
		if user.Mobile != "" && existing.Mobile == user.Mobile {
			return conflict("手机号已存在")
		}
	}
	s.users[user.ID] = user
	return nil
}

func (s *memoryStore) UpdateUser(_ context.Context, user User) error {
	if _, ok := s.users[user.ID]; !ok {
		return ErrNotFound
	}
	for _, existing := range s.users {
		if existing.ID == user.ID || existing.Deleted {
			continue
		}
		if user.Email != "" && existing.Email == user.Email {
			return conflict("邮箱已存在")
		}
		if user.Mobile != "" && existing.Mobile == user.Mobile {
			return conflict("手机号已存在")
		}
	}
	s.users[user.ID] = user
	return nil
}

func (s *memoryStore) FindUser(_ context.Context, id string) (User, error) {
	user, ok := s.users[id]
	if !ok || user.Deleted {
		return User{}, ErrNotFound
	}
	return user, nil
}

func (s *memoryStore) FindUserByUsername(_ context.Context, username string) (User, error) {
	for _, user := range s.users {
		if !user.Deleted && user.Username == username {
			return user, nil
		}
	}
	return User{}, ErrNotFound
}

func (s *memoryStore) UpdateUserPassword(_ context.Context, userID, passwordHash string) error {
	user, ok := s.users[userID]
	if !ok || user.Deleted {
		return ErrNotFound
	}
	user.PasswordHash = passwordHash
	user.UpdateTime = time.Now().UTC()
	s.users[userID] = user
	return nil
}

func (s *memoryStore) UpdateUserLogin(_ context.Context, userID, loginIP string, loginDate time.Time) error {
	user, ok := s.users[userID]
	if !ok || user.Deleted {
		return ErrNotFound
	}
	user.LoginIP = loginIP
	at := loginDate.UTC()
	user.LoginDate = &at
	user.UpdateTime = time.Now().UTC()
	s.users[userID] = user
	return nil
}

func (s *memoryStore) ListUsers(_ context.Context, keyword string, page, pageSize int) ([]User, int64, error) {
	items := make([]User, 0)
	for _, user := range s.users {
		if user.Deleted {
			continue
		}
		if keyword == "" || strings.Contains(user.Username, keyword) || strings.Contains(user.Nickname, keyword) {
			items = append(items, user)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreateTime.After(items[j].CreateTime) })
	return pageSlice(items, page, pageSize), int64(len(items)), nil
}

func (s *memoryStore) InsertTenant(_ context.Context, tenant Tenant) error {
	for _, existing := range s.tenants {
		if existing.Deleted {
			continue
		}
		if existing.Code == tenant.Code {
			return conflict("租户编码已存在")
		}
	}
	s.tenants[tenant.ID] = tenant
	return nil
}

func (s *memoryStore) UpdateTenant(_ context.Context, tenant Tenant) error {
	if _, ok := s.tenants[tenant.ID]; !ok {
		return ErrNotFound
	}
	s.tenants[tenant.ID] = tenant
	return nil
}

func (s *memoryStore) FindTenant(_ context.Context, id string) (Tenant, error) {
	tenant, ok := s.tenants[id]
	if !ok || tenant.Deleted {
		return Tenant{}, ErrNotFound
	}
	return tenant, nil
}

func (s *memoryStore) ListTenants(_ context.Context, keyword string, page, pageSize int) ([]Tenant, int64, error) {
	items := make([]Tenant, 0)
	for _, tenant := range s.tenants {
		if tenant.Deleted {
			continue
		}
		if keyword == "" || strings.Contains(tenant.Name, keyword) || strings.Contains(tenant.Code, keyword) {
			items = append(items, tenant)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreateTime.After(items[j].CreateTime) })
	return pageSlice(items, page, pageSize), int64(len(items)), nil
}

func (s *memoryStore) InsertOrganization(_ context.Context, org Organization) error {
	if err := s.ensureOrganizationUnique(org); err != nil {
		return err
	}
	s.orgs[org.ID] = org
	return nil
}

func (s *memoryStore) UpdateOrganization(_ context.Context, org Organization) error {
	if _, ok := s.orgs[org.ID]; !ok {
		return ErrNotFound
	}
	if err := s.ensureOrganizationUnique(org); err != nil {
		return err
	}
	s.orgs[org.ID] = org
	return nil
}

func (s *memoryStore) ensureOrganizationUnique(org Organization) error {
	for _, existing := range s.orgs {
		if existing.ID == org.ID || existing.TenantID != org.TenantID || existing.Deleted {
			continue
		}
		if existing.Code == org.Code {
			return conflict("部门编码已存在")
		}
		if existing.ParentID == org.ParentID && existing.Name == org.Name {
			return conflict("同一上级下部门名称已存在")
		}
	}
	return nil
}

func (s *memoryStore) DeleteOrganization(_ context.Context, id string) error {
	org, ok := s.orgs[id]
	if !ok || org.Deleted {
		return ErrNotFound
	}
	org.Deleted = true
	s.orgs[id] = org
	return nil
}

func (s *memoryStore) FindOrganization(_ context.Context, tenantID, id string) (Organization, error) {
	org, ok := s.orgs[id]
	if !ok || org.Deleted || org.TenantID != tenantID {
		return Organization{}, ErrNotFound
	}
	return org, nil
}

func (s *memoryStore) ListOrganizations(_ context.Context, tenantID string) ([]Organization, error) {
	items := make([]Organization, 0)
	for _, org := range s.orgs {
		if org.TenantID == tenantID && !org.Deleted {
			items = append(items, org)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Sort == items[j].Sort {
			return items[i].Name < items[j].Name
		}
		return items[i].Sort < items[j].Sort
	})
	return items, nil
}

func (s *memoryStore) CountChildOrganizations(_ context.Context, id string) (int64, error) {
	var count int64
	for _, org := range s.orgs {
		if org.ParentID == id && !org.Deleted {
			count++
		}
	}
	return count, nil
}

func (s *memoryStore) CountOrganizationAssignments(_ context.Context, organizationID string) (int64, error) {
	var count int64
	for _, item := range s.assignments {
		if item.OrganizationID == organizationID {
			count++
		}
	}
	return count, nil
}

func (s *memoryStore) InsertPost(_ context.Context, post Post) error {
	for _, existing := range s.posts {
		if existing.Deleted || existing.TenantID != post.TenantID {
			continue
		}
		if existing.Code == post.Code {
			return conflict("岗位编码已存在")
		}
	}
	s.posts[post.ID] = post
	return nil
}

func (s *memoryStore) UpdatePost(_ context.Context, post Post) error {
	current, ok := s.posts[post.ID]
	if !ok || current.Deleted {
		return ErrNotFound
	}
	for _, existing := range s.posts {
		if existing.ID == post.ID || existing.Deleted || existing.TenantID != post.TenantID {
			continue
		}
		if existing.Code == post.Code {
			return conflict("岗位编码已存在")
		}
	}
	s.posts[post.ID] = post
	return nil
}

func (s *memoryStore) DeletePost(_ context.Context, id string) error {
	post, ok := s.posts[id]
	if !ok || post.Deleted {
		return ErrNotFound
	}
	post.Deleted = true
	s.posts[id] = post
	return nil
}

func (s *memoryStore) FindPost(_ context.Context, tenantID, id string) (Post, error) {
	post, ok := s.posts[id]
	if !ok || post.Deleted || post.TenantID != tenantID {
		return Post{}, ErrNotFound
	}
	return post, nil
}

func (s *memoryStore) ListPosts(_ context.Context, tenantID, keyword string, page, pageSize int) ([]Post, int64, error) {
	items := make([]Post, 0)
	for _, post := range s.posts {
		if post.Deleted || post.TenantID != tenantID {
			continue
		}
		if keyword != "" && !strings.Contains(post.Name, keyword) && !strings.Contains(post.Code, keyword) {
			continue
		}
		items = append(items, post)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Sort == items[j].Sort {
			return items[i].Name < items[j].Name
		}
		return items[i].Sort < items[j].Sort
	})
	return pageSlice(items, page, pageSize), int64(len(items)), nil
}

func (s *memoryStore) InsertMenu(_ context.Context, menu Menu) error {
	s.menus[menu.ID] = menu
	return nil
}

func (s *memoryStore) UpdateMenu(_ context.Context, menu Menu) error {
	current, ok := s.menus[menu.ID]
	if !ok || current.Deleted {
		return ErrNotFound
	}
	s.menus[menu.ID] = menu
	return nil
}

func (s *memoryStore) DeleteMenu(_ context.Context, id string) error {
	menu, ok := s.menus[id]
	if !ok || menu.Deleted {
		return ErrNotFound
	}
	menu.Deleted = true
	s.menus[id] = menu
	return nil
}

func (s *memoryStore) FindMenu(_ context.Context, id string) (Menu, error) {
	menu, ok := s.menus[id]
	if !ok || menu.Deleted {
		return Menu{}, ErrNotFound
	}
	return menu, nil
}

func (s *memoryStore) ListMenus(_ context.Context) ([]Menu, error) {
	items := make([]Menu, 0)
	for _, menu := range s.menus {
		if !menu.Deleted {
			items = append(items, menu)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Sort == items[j].Sort {
			return items[i].Name < items[j].Name
		}
		return items[i].Sort < items[j].Sort
	})
	return items, nil
}

func (s *memoryStore) CountChildMenus(_ context.Context, id string) (int64, error) {
	var count int64
	for _, menu := range s.menus {
		if menu.ParentID == id && !menu.Deleted {
			count++
		}
	}
	return count, nil
}

func (s *memoryStore) InsertRole(_ context.Context, role Role) error {
	for _, existing := range s.roles {
		if existing.Deleted || existing.TenantID != role.TenantID {
			continue
		}
		if existing.Code == role.Code {
			return conflict("角色权限字符串已存在")
		}
	}
	s.roles[role.ID] = role
	return nil
}

func (s *memoryStore) UpdateRole(_ context.Context, role Role) error {
	current, ok := s.roles[role.ID]
	if !ok || current.Deleted {
		return ErrNotFound
	}
	for _, existing := range s.roles {
		if existing.ID == role.ID || existing.Deleted || existing.TenantID != role.TenantID {
			continue
		}
		if existing.Code == role.Code {
			return conflict("角色权限字符串已存在")
		}
	}
	s.roles[role.ID] = role
	return nil
}

func (s *memoryStore) DeleteRole(_ context.Context, id string) error {
	role, ok := s.roles[id]
	if !ok || role.Deleted {
		return ErrNotFound
	}
	role.Deleted = true
	s.roles[id] = role
	return nil
}

func (s *memoryStore) FindRole(_ context.Context, tenantID, id string) (Role, error) {
	role, ok := s.roles[id]
	if !ok || role.Deleted || role.TenantID != tenantID {
		return Role{}, ErrNotFound
	}
	return role, nil
}

func (s *memoryStore) ListRoles(_ context.Context, tenantID, keyword string, page, pageSize int) ([]Role, int64, error) {
	items := make([]Role, 0)
	for _, role := range s.roles {
		if role.Deleted || role.TenantID != tenantID {
			continue
		}
		if keyword != "" && !strings.Contains(role.Name, keyword) && !strings.Contains(role.Code, keyword) {
			continue
		}
		items = append(items, role)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Sort == items[j].Sort {
			return items[i].Name < items[j].Name
		}
		return items[i].Sort < items[j].Sort
	})
	return pageSlice(items, page, pageSize), int64(len(items)), nil
}

func (s *memoryStore) ListRoleMenuIDs(_ context.Context, tenantID, roleID string) ([]string, error) {
	ids := make([]string, 0)
	for _, item := range s.roleMenus {
		if !item.Deleted && item.TenantID == tenantID && item.RoleID == roleID {
			ids = append(ids, item.MenuID)
		}
	}
	sort.Strings(ids)
	return ids, nil
}

func (s *memoryStore) ReplaceRoleMenus(_ context.Context, tenantID, roleID string, menuIDs []string) error {
	for id, item := range s.roleMenus {
		if item.TenantID == tenantID && item.RoleID == roleID && !item.Deleted {
			item.Deleted = true
			s.roleMenus[id] = item
		}
	}
	for _, menuID := range menuIDs {
		id := newID()
		s.roleMenus[id] = RoleMenu{ID: id, TenantID: tenantID, RoleID: roleID, MenuID: menuID}
	}
	return nil
}

func (s *memoryStore) ListRoleDeptIDs(_ context.Context, tenantID, roleID string) ([]string, error) {
	ids := make([]string, 0)
	for _, item := range s.roleDepts {
		if !item.Deleted && item.TenantID == tenantID && item.RoleID == roleID {
			ids = append(ids, item.DeptID)
		}
	}
	sort.Strings(ids)
	return ids, nil
}

func (s *memoryStore) ReplaceRoleDepts(_ context.Context, tenantID, roleID string, deptIDs []string) error {
	for id, item := range s.roleDepts {
		if item.TenantID == tenantID && item.RoleID == roleID && !item.Deleted {
			item.Deleted = true
			s.roleDepts[id] = item
		}
	}
	for _, deptID := range deptIDs {
		id := newID()
		s.roleDepts[id] = RoleDept{ID: id, TenantID: tenantID, RoleID: roleID, DeptID: deptID}
	}
	return nil
}

func (s *memoryStore) ListUserRoleIDs(_ context.Context, tenantID, userID string) ([]string, error) {
	ids := make([]string, 0)
	for _, item := range s.userRoles {
		if !item.Deleted && item.TenantID == tenantID && item.UserID == userID {
			ids = append(ids, item.RoleID)
		}
	}
	sort.Strings(ids)
	return ids, nil
}

func (s *memoryStore) ListRolesByIDs(_ context.Context, tenantID string, roleIDs []string) ([]Role, error) {
	wanted := map[string]struct{}{}
	for _, id := range roleIDs {
		wanted[id] = struct{}{}
	}
	items := make([]Role, 0)
	for _, role := range s.roles {
		if role.Deleted || role.TenantID != tenantID || role.Status != StatusNormal {
			continue
		}
		if _, ok := wanted[role.ID]; ok {
			items = append(items, role)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Sort == items[j].Sort {
			return items[i].Name < items[j].Name
		}
		return items[i].Sort < items[j].Sort
	})
	return items, nil
}

func (s *memoryStore) ListMenuIDsByRoleIDs(_ context.Context, tenantID string, roleIDs []string) ([]string, error) {
	wanted := map[string]struct{}{}
	for _, id := range roleIDs {
		wanted[id] = struct{}{}
	}
	seen := map[string]struct{}{}
	ids := make([]string, 0)
	for _, item := range s.roleMenus {
		if item.Deleted || item.TenantID != tenantID {
			continue
		}
		if _, ok := wanted[item.RoleID]; !ok {
			continue
		}
		if _, ok := seen[item.MenuID]; ok {
			continue
		}
		seen[item.MenuID] = struct{}{}
		ids = append(ids, item.MenuID)
	}
	sort.Strings(ids)
	return ids, nil
}

func (s *memoryStore) ListMenusByIDs(_ context.Context, menuIDs []string) ([]Menu, error) {
	wanted := map[string]struct{}{}
	for _, id := range menuIDs {
		wanted[id] = struct{}{}
	}
	items := make([]Menu, 0)
	for _, menu := range s.menus {
		if menu.Deleted {
			continue
		}
		if _, ok := wanted[menu.ID]; ok {
			items = append(items, menu)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Sort == items[j].Sort {
			return items[i].Name < items[j].Name
		}
		return items[i].Sort < items[j].Sort
	})
	return items, nil
}

func (s *memoryStore) ListRoleUsers(_ context.Context, tenantID, roleID string) ([]RoleUserView, error) {
	items := make([]RoleUserView, 0)
	for _, item := range s.userRoles {
		if item.Deleted || item.TenantID != tenantID || item.RoleID != roleID {
			continue
		}
		user := s.users[item.UserID]
		if user.Deleted {
			continue
		}
		items = append(items, RoleUserView{UserID: user.ID, Username: user.Username, Nickname: user.Nickname})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Username < items[j].Username })
	return items, nil
}

func (s *memoryStore) ReplaceRoleUsers(_ context.Context, tenantID, roleID string, userIDs []string) error {
	for id, item := range s.userRoles {
		if item.TenantID == tenantID && item.RoleID == roleID && !item.Deleted {
			item.Deleted = true
			s.userRoles[id] = item
		}
	}
	for _, userID := range userIDs {
		id := newID()
		s.userRoles[id] = UserRole{ID: id, TenantID: tenantID, UserID: userID, RoleID: roleID}
	}
	return nil
}

func (s *memoryStore) ClearUserRoles(_ context.Context, tenantID, userID string) error {
	for id, item := range s.userRoles {
		if item.TenantID == tenantID && item.UserID == userID && !item.Deleted {
			item.Deleted = true
			s.userRoles[id] = item
		}
	}
	return nil
}

func (s *memoryStore) CountRoleUsers(_ context.Context, roleID string) (int64, error) {
	var count int64
	for _, item := range s.userRoles {
		if !item.Deleted && item.RoleID == roleID {
			count++
		}
	}
	return count, nil
}

func (s *memoryStore) InsertMembership(_ context.Context, membership TenantMembership) error {
	key := memberKey(membership.TenantID, membership.UserID)
	if _, ok := s.members[key]; ok {
		return conflict("账号已在该租户中")
	}
	s.members[key] = membership
	return nil
}

func (s *memoryStore) DeleteMembership(_ context.Context, tenantID, userID string) error {
	key := memberKey(tenantID, userID)
	if _, ok := s.members[key]; !ok {
		return ErrNotFound
	}
	delete(s.members, key)
	return nil
}

func (s *memoryStore) FindMembership(_ context.Context, tenantID, userID string) (TenantMembership, error) {
	item, ok := s.members[memberKey(tenantID, userID)]
	if !ok {
		return TenantMembership{}, ErrNotFound
	}
	return item, nil
}

func (s *memoryStore) ListMembershipsByUser(_ context.Context, userID string) ([]TenantMembership, error) {
	items := make([]TenantMembership, 0)
	for _, item := range s.members {
		if item.UserID == userID {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreateTime.Before(items[j].CreateTime) })
	return items, nil
}

func (s *memoryStore) ListMembers(_ context.Context, tenantID, keyword string, page, pageSize int) ([]MemberRecord, int64, error) {
	items := make([]MemberRecord, 0)
	for _, membership := range s.members {
		if membership.TenantID != tenantID {
			continue
		}
		user := s.users[membership.UserID]
		if keyword != "" && !strings.Contains(user.Username, keyword) && !strings.Contains(user.Nickname, keyword) {
			continue
		}
		items = append(items, MemberRecord{Membership: membership, Username: user.Username, Nickname: user.Nickname})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Membership.CreateTime.After(items[j].Membership.CreateTime)
	})
	total := int64(len(items))
	return pageSlice(items, page, pageSize), total, nil
}

func (s *memoryStore) ReplaceAssignments(_ context.Context, tenantID, userID string, assignments []OrganizationMembership) error {
	for key, item := range s.assignments {
		if item.TenantID == tenantID && item.UserID == userID {
			delete(s.assignments, key)
		}
	}
	for _, item := range assignments {
		s.assignments[item.TenantID+"/"+item.UserID+"/"+item.OrganizationID] = item
	}
	return nil
}

func (s *memoryStore) ListAssignments(_ context.Context, tenantID, userID string) ([]OrganizationMembership, error) {
	items := make([]OrganizationMembership, 0)
	for _, item := range s.assignments {
		if item.TenantID == tenantID && item.UserID == userID {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreateTime.Before(items[j].CreateTime) })
	return items, nil
}

func memberKey(tenantID, userID string) string {
	return tenantID + "/" + userID
}

func cloneMap[T any](source map[string]T) map[string]T {
	copied := make(map[string]T, len(source))
	for key, value := range source {
		copied[key] = value
	}
	return copied
}

func pageSlice[T any](items []T, page, pageSize int) []T {
	start := (page - 1) * pageSize
	if start >= len(items) {
		return []T{}
	}
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}
