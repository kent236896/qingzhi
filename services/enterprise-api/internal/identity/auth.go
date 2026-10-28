package identity

import (
	"context"
	"errors"
	"strings"
	"time"
)

// LoginInput is the credential payload for admin login.
type LoginInput struct {
	Username string
	Password string
	TenantID string
	LoginIP  string
}

// LoginResult is returned after a successful password check.
type LoginResult struct {
	User     User
	TenantID string
	Access   SessionAccess
}

// SessionAccess is the authorization snapshot for one user in one tenant.
type SessionAccess struct {
	UserID      string
	Username    string
	Nickname    string
	TenantID    string
	Permissions []string
	Access      DataAccess
	Roles       []Role
}

// Authenticate validates username/password and resolves the active tenant.
func (s *Service) Authenticate(ctx context.Context, input LoginInput) (LoginResult, error) {
	username := strings.TrimSpace(input.Username)
	if username == "" || input.Password == "" {
		return LoginResult{}, unauthorized("用户名或密码错误")
	}
	user, err := s.store.FindUserByUsername(ctx, username)
	if errors.Is(err, ErrNotFound) {
		return LoginResult{}, unauthorized("用户名或密码错误")
	}
	if err != nil {
		return LoginResult{}, err
	}
	if user.Status != StatusNormal {
		return LoginResult{}, unauthorized("账号已停用")
	}
	if !user.LocalPasswordEnabled {
		return LoginResult{}, unauthorized("该账号不支持本地密码登录，请使用统一认证中心登录")
	}
	if !checkPassword(user.PasswordHash, input.Password) {
		return LoginResult{}, unauthorized("用户名或密码错误")
	}

	memberships, err := s.store.ListMembershipsByUser(ctx, user.ID)
	if err != nil {
		return LoginResult{}, err
	}
	active := make([]TenantMembership, 0, len(memberships))
	for _, membership := range memberships {
		if membership.Status != StatusNormal {
			continue
		}
		tenant, err := s.store.FindTenant(ctx, membership.TenantID)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return LoginResult{}, err
		}
		if tenant.Status != StatusNormal {
			continue
		}
		active = append(active, membership)
	}
	if len(active) == 0 {
		return LoginResult{}, unauthorized("账号未加入可用租户")
	}

	tenantID := strings.TrimSpace(input.TenantID)
	if tenantID == "" {
		if len(active) != 1 {
			return LoginResult{}, invalid("请选择租户后登录")
		}
		tenantID = active[0].TenantID
	} else {
		found := false
		for _, membership := range active {
			if membership.TenantID == tenantID {
				found = true
				break
			}
		}
		if !found {
			return LoginResult{}, unauthorized("账号不属于该租户")
		}
	}

	access, err := s.LoadSessionAccess(ctx, user.ID, tenantID)
	if err != nil {
		return LoginResult{}, err
	}
	_ = s.store.UpdateUserLogin(ctx, user.ID, strings.TrimSpace(input.LoginIP), time.Now().UTC())
	user.PasswordHash = ""
	return LoginResult{User: user, TenantID: tenantID, Access: access}, nil
}

// FirstActiveTenantID returns the oldest normal tenant. Used when SSO auto-provision
// has no explicit default tenant configured after a fresh deploy.
func (s *Service) FirstActiveTenantID(ctx context.Context) (string, error) {
	// ListTenants is newest-first; scan a page and pick the earliest create_time.
	items, _, err := s.store.ListTenants(ctx, "", 1, 100)
	if err != nil {
		return "", err
	}
	var best *Tenant
	for i := range items {
		item := items[i]
		if item.Status != StatusNormal {
			continue
		}
		if best == nil || item.CreateTime.Before(best.CreateTime) {
			copy := item
			best = &copy
		}
	}
	if best == nil {
		return "", notFound("系统中尚无可用租户，请先创建租户")
	}
	return best.ID, nil
}

// LoginFederated establishes a local session for an already-mapped external identity.
// It does not accept external claims for roles or permissions.
func (s *Service) LoginFederated(ctx context.Context, userID, tenantID, loginIP string) (LoginResult, error) {
	userID = strings.TrimSpace(userID)
	tenantID = strings.TrimSpace(tenantID)
	if userID == "" {
		return LoginResult{}, unauthorized("外部身份未映射到本平台账号")
	}
	user, err := s.store.FindUser(ctx, userID)
	if errors.Is(err, ErrNotFound) {
		return LoginResult{}, unauthorized("外部身份未映射到本平台账号")
	}
	if err != nil {
		return LoginResult{}, err
	}
	if user.Status != StatusNormal {
		return LoginResult{}, unauthorized("账号已停用")
	}

	memberships, err := s.store.ListMembershipsByUser(ctx, user.ID)
	if err != nil {
		return LoginResult{}, err
	}
	active := make([]TenantMembership, 0, len(memberships))
	for _, membership := range memberships {
		if membership.Status != StatusNormal {
			continue
		}
		tenant, err := s.store.FindTenant(ctx, membership.TenantID)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return LoginResult{}, err
		}
		if tenant.Status != StatusNormal {
			continue
		}
		active = append(active, membership)
	}
	if len(active) == 0 {
		return LoginResult{}, unauthorized("账号未加入可用租户")
	}
	if tenantID == "" {
		if len(active) != 1 {
			return LoginResult{}, invalid("请选择租户后登录")
		}
		tenantID = active[0].TenantID
	} else {
		found := false
		for _, membership := range active {
			if membership.TenantID == tenantID {
				found = true
				break
			}
		}
		if !found {
			return LoginResult{}, unauthorized("账号不属于该租户")
		}
	}

	access, err := s.LoadSessionAccess(ctx, user.ID, tenantID)
	if err != nil {
		return LoginResult{}, err
	}
	_ = s.store.UpdateUserLogin(ctx, user.ID, strings.TrimSpace(loginIP), time.Now().UTC())
	user.PasswordHash = ""
	return LoginResult{User: user, TenantID: tenantID, Access: access}, nil
}

// FindUserForFederation looks up a local account for first-time IdP binding.
// Prefer email (case-insensitive), then exact username. Disabled accounts are skipped.
func (s *Service) FindUserForFederation(ctx context.Context, email, username string) (User, error) {
	email = strings.TrimSpace(email)
	username = strings.TrimSpace(username)
	if email != "" {
		user, err := s.store.FindUserByEmail(ctx, email)
		if err == nil {
			if user.Status == StatusNormal {
				return user, nil
			}
		} else if !errors.Is(err, ErrNotFound) {
			return User{}, err
		}
	}
	if username != "" {
		user, err := s.store.FindUserByUsername(ctx, username)
		if err == nil {
			if user.Status == StatusNormal {
				return user, nil
			}
			return User{}, ErrNotFound
		}
		if !errors.Is(err, ErrNotFound) {
			return User{}, err
		}
	}
	return User{}, ErrNotFound
}

// CreateFederatedUser creates a minimum-privilege local account for an external subject.
// Callers must never grant admin roles from external claims.
func (s *Service) CreateFederatedUser(ctx context.Context, username, nickname, email, tenantID string) (User, error) {
	password, err := randomPassword()
	if err != nil {
		return User{}, err
	}
	disabled := false
	user, err := s.CreateUser(ctx, CreateUserInput{
		Username:             username,
		Password:             password,
		Nickname:             nickname,
		Email:                email,
		Remark:               "federated",
		LocalPasswordEnabled: &disabled,
	})
	if err != nil {
		return User{}, err
	}
	if strings.TrimSpace(tenantID) != "" {
		if err := s.EnsureTenantMembership(ctx, tenantID, user.ID); err != nil {
			return User{}, err
		}
	}
	return user, nil
}

// EnsureTenantMembership links a user to a tenant without roles or department posts.
func (s *Service) EnsureTenantMembership(ctx context.Context, tenantID, userID string) error {
	tenantID = strings.TrimSpace(tenantID)
	userID = strings.TrimSpace(userID)
	if tenantID == "" || userID == "" {
		return invalid("租户或账号标识不合法")
	}
	return s.store.Transaction(ctx, func(tx Store) error {
		if _, err := tx.FindTenant(ctx, tenantID); errors.Is(err, ErrNotFound) {
			return notFound("租户不存在")
		} else if err != nil {
			return err
		}
		if _, err := tx.FindUser(ctx, userID); errors.Is(err, ErrNotFound) {
			return notFound("账号不存在")
		} else if err != nil {
			return err
		}
		if _, err := tx.FindMembership(ctx, tenantID, userID); err == nil {
			return nil
		} else if !errors.Is(err, ErrNotFound) {
			return err
		}
		now := time.Now().UTC()
		return tx.InsertMembership(ctx, TenantMembership{
			ID:         newID(),
			TenantID:   tenantID,
			UserID:     userID,
			Status:     StatusNormal,
			CreateTime: now,
			UpdateTime: now,
		})
	})
}

// LoadSessionAccess loads roles, permissions and data scope for a user in a tenant.
func (s *Service) LoadSessionAccess(ctx context.Context, userID, tenantID string) (SessionAccess, error) {
	var result SessionAccess
	err := s.store.Transaction(ctx, func(tx Store) error {
		user, err := tx.FindUser(ctx, userID)
		if errors.Is(err, ErrNotFound) {
			return unauthorized("账号不存在或已失效")
		}
		if err != nil {
			return err
		}
		if user.Status != StatusNormal {
			return unauthorized("账号已停用")
		}
		membership, err := tx.FindMembership(ctx, tenantID, userID)
		if errors.Is(err, ErrNotFound) {
			return unauthorized("账号不属于该租户")
		}
		if err != nil {
			return err
		}
		if membership.Status != StatusNormal {
			return unauthorized("租户成员已停用")
		}
		tenant, err := tx.FindTenant(ctx, tenantID)
		if errors.Is(err, ErrNotFound) {
			return unauthorized("租户不存在")
		}
		if err != nil {
			return err
		}
		if tenant.Status != StatusNormal {
			return unauthorized("租户已停用")
		}

		roleIDs, err := tx.ListUserRoleIDs(ctx, tenantID, userID)
		if err != nil {
			return err
		}
		roles, err := tx.ListRolesByIDs(ctx, tenantID, roleIDs)
		if err != nil {
			return err
		}
		roleDepts := map[string][]string{}
		for _, role := range roles {
			if role.DataScope != DataScopeCustom {
				continue
			}
			ids, err := tx.ListRoleDeptIDs(ctx, tenantID, role.ID)
			if err != nil {
				return err
			}
			roleDepts[role.ID] = ids
		}
		orgs, err := tx.ListOrganizations(ctx, tenantID)
		if err != nil {
			return err
		}
		assignments, err := tx.ListAssignments(ctx, tenantID, userID)
		if err != nil {
			return err
		}
		primaryOrg := ""
		for _, item := range assignments {
			if item.IsPrimary {
				primaryOrg = item.OrganizationID
				break
			}
		}
		menuIDs, err := tx.ListMenuIDsByRoleIDs(ctx, tenantID, roleIDs)
		if err != nil {
			return err
		}
		menus, err := tx.ListMenusByIDs(ctx, menuIDs)
		if err != nil {
			return err
		}
		permissions := make([]string, 0)
		seen := map[string]struct{}{}
		for _, menu := range menus {
			if menu.Status != StatusNormal || menu.Permission == "" {
				continue
			}
			if _, ok := seen[menu.Permission]; ok {
				continue
			}
			seen[menu.Permission] = struct{}{}
			permissions = append(permissions, menu.Permission)
		}
		result = SessionAccess{
			UserID:      user.ID,
			Username:    user.Username,
			Nickname:    user.Nickname,
			TenantID:    tenantID,
			Permissions: permissions,
			Access:      ResolveDataAccess(user.ID, primaryOrg, roles, roleDepts, orgs),
			Roles:       roles,
		}
		return nil
	})
	return result, err
}

// ListAuthorizedMenus returns the menu tree visible to the current user's roles.
func (s *Service) ListAuthorizedMenus(ctx context.Context, userID, tenantID string) ([]MenuNode, error) {
	var nodes []MenuNode
	err := s.store.Transaction(ctx, func(tx Store) error {
		roleIDs, err := tx.ListUserRoleIDs(ctx, tenantID, userID)
		if err != nil {
			return err
		}
		menuIDs, err := tx.ListMenuIDsByRoleIDs(ctx, tenantID, roleIDs)
		if err != nil {
			return err
		}
		allowed := map[string]struct{}{}
		for _, id := range menuIDs {
			allowed[id] = struct{}{}
		}
		all, err := tx.ListMenus(ctx)
		if err != nil {
			return err
		}
		byID := map[string]Menu{}
		for _, menu := range all {
			byID[menu.ID] = menu
		}
		keep := map[string]struct{}{}
		for id := range allowed {
			for current := id; current != ""; {
				if _, ok := keep[current]; ok {
					break
				}
				menu, ok := byID[current]
				if !ok || menu.Status != StatusNormal {
					break
				}
				keep[current] = struct{}{}
				current = menu.ParentID
			}
		}
		filtered := make([]Menu, 0, len(keep))
		for _, menu := range all {
			if _, ok := keep[menu.ID]; ok {
				filtered = append(filtered, menu)
			}
		}
		nodes = buildMenuTree(filtered)
		return nil
	})
	return nodes, err
}

// ListRoleDepts returns department ids bound to a custom data-scope role.
func (s *Service) ListRoleDepts(ctx context.Context, tenantID, roleID string) ([]string, error) {
	var ids []string
	err := s.store.Transaction(ctx, func(tx Store) error {
		if _, err := tx.FindRole(ctx, tenantID, roleID); errors.Is(err, ErrNotFound) {
			return notFound("角色不存在")
		} else if err != nil {
			return err
		}
		items, err := tx.ListRoleDeptIDs(ctx, tenantID, roleID)
		if err != nil {
			return err
		}
		ids = items
		return nil
	})
	return ids, err
}

// ReplaceRoleDepts replaces the custom department set of a role.
func (s *Service) ReplaceRoleDepts(ctx context.Context, tenantID, roleID string, deptIDs []string) ([]string, error) {
	normalized, err := uniqueIDs(deptIDs, "部门标识不合法")
	if err != nil {
		return nil, err
	}
	err = s.store.Transaction(ctx, func(tx Store) error {
		role, err := tx.FindRole(ctx, tenantID, roleID)
		if errors.Is(err, ErrNotFound) {
			return notFound("角色不存在")
		}
		if err != nil {
			return err
		}
		if role.DataScope != DataScopeCustom && len(normalized) > 0 {
			return invalid("仅自定义数据范围可配置部门")
		}
		for _, deptID := range normalized {
			if _, err := tx.FindOrganization(ctx, tenantID, deptID); errors.Is(err, ErrNotFound) {
				return notFound("部门不存在")
			} else if err != nil {
				return err
			}
		}
		return tx.ReplaceRoleDepts(ctx, tenantID, roleID, normalized)
	})
	if err != nil {
		return nil, err
	}
	return s.ListRoleDepts(ctx, tenantID, roleID)
}
