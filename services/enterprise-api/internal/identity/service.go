package identity

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Service applies identity rules on top of a Store.
type Service struct {
	store Store
}

// NewService returns an identity service.
func NewService(store Store) *Service {
	return &Service{store: store}
}

type CreateTenantInput struct {
	Name          string
	Code          string
	ContactUserID string
	ContactName   string
	ContactMobile string
	AccountCount  int
}

type UpdateTenantInput struct {
	Name          *string
	ContactUserID *string
	ContactName   *string
	ContactMobile *string
	Status        *int
	AccountCount  *int
}

type CreateOrganizationInput struct {
	Name         string
	Code         string
	ParentID     string
	Sort         int
	LeaderUserID string
	Phone        string
	Email        string
}

type UpdateOrganizationInput struct {
	Name         *string
	Code         *string
	ParentID     *string
	ParentSet    bool
	Sort         *int
	LeaderUserID *string
	Phone        *string
	Email        *string
	Status       *int
}

type CreateUserInput struct {
	Username string
	Password string
	Nickname string
	Remark   string
	Email    string
	Mobile   string
	Sex      int
	Avatar   string
}

type UpdateUserInput struct {
	Nickname *string
	Password *string
	Remark   *string
	Email    *string
	Mobile   *string
	Sex      *int
	Avatar   *string
	Status   *int
}

type CreateMemberInput struct {
	UserID string
	Status int
	AssignmentInput
}

func (s *Service) ListTenants(ctx context.Context, query ListQuery) (Page[Tenant], error) {
	page, pageSize, err := normalizePage(query.Page, query.PageSize)
	if err != nil {
		return Page[Tenant]{}, err
	}
	items, total, err := s.store.ListTenants(ctx, strings.TrimSpace(query.Keyword), page, pageSize)
	if err != nil {
		return Page[Tenant]{}, err
	}
	return newPage(items, page, pageSize, total), nil
}

func (s *Service) CreateTenant(ctx context.Context, input CreateTenantInput) (Tenant, error) {
	tenant := Tenant{
		Name:          strings.TrimSpace(input.Name),
		Code:          strings.ToLower(strings.TrimSpace(input.Code)),
		ContactUserID: strings.TrimSpace(input.ContactUserID),
		ContactName:   strings.TrimSpace(input.ContactName),
		ContactMobile: strings.TrimSpace(input.ContactMobile),
		Status:        StatusNormal,
		AccountCount:  input.AccountCount,
	}
	if err := validateName(tenant.Name); err != nil {
		return Tenant{}, err
	}
	if err := validateName(tenant.ContactName); err != nil {
		return Tenant{}, err
	}
	if err := validateTenantCode(tenant.Code); err != nil {
		return Tenant{}, err
	}
	if err := validateOptionalText(tenant.ContactMobile, 500, "联系手机最长 500 个字符"); err != nil {
		return Tenant{}, err
	}
	if err := validateAccountCount(tenant.AccountCount); err != nil {
		return Tenant{}, err
	}
	now := time.Now().UTC()
	tenant.ID = newID()
	tenant.CreateTime = now
	tenant.UpdateTime = now
	err := s.store.Transaction(ctx, func(tx Store) error {
		if err := ensureUser(ctx, tx, tenant.ContactUserID, "联系人不存在"); err != nil {
			return err
		}
		if err := tx.InsertTenant(ctx, tenant); err != nil {
			return err
		}
		return tx.InsertOrganization(ctx, rootOrganization(tenant, now))
	})
	return tenant, err
}

func (s *Service) GetTenant(ctx context.Context, id string) (Tenant, error) {
	tenant, err := s.store.FindTenant(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return Tenant{}, notFound("租户不存在")
	}
	return tenant, err
}

func (s *Service) UpdateTenant(ctx context.Context, id string, input UpdateTenantInput) (Tenant, error) {
	if input.Name == nil && input.ContactUserID == nil && input.ContactName == nil && input.ContactMobile == nil && input.Status == nil && input.AccountCount == nil {
		return Tenant{}, invalid("没有可更新的字段")
	}
	var updated Tenant
	err := s.store.Transaction(ctx, func(tx Store) error {
		tenant, err := tx.FindTenant(ctx, id)
		if errors.Is(err, ErrNotFound) {
			return notFound("租户不存在")
		}
		if err != nil {
			return err
		}
		if input.Name != nil {
			tenant.Name = strings.TrimSpace(*input.Name)
			if err := validateName(tenant.Name); err != nil {
				return err
			}
		}
		if input.ContactName != nil {
			tenant.ContactName = strings.TrimSpace(*input.ContactName)
			if err := validateName(tenant.ContactName); err != nil {
				return err
			}
		}
		if input.ContactMobile != nil {
			tenant.ContactMobile = strings.TrimSpace(*input.ContactMobile)
			if err := validateOptionalText(tenant.ContactMobile, 500, "联系手机最长 500 个字符"); err != nil {
				return err
			}
		}
		if input.ContactUserID != nil {
			tenant.ContactUserID = strings.TrimSpace(*input.ContactUserID)
			if err := ensureUser(ctx, tx, tenant.ContactUserID, "联系人不存在"); err != nil {
				return err
			}
		}
		if input.AccountCount != nil {
			if err := validateAccountCount(*input.AccountCount); err != nil {
				return err
			}
			tenant.AccountCount = *input.AccountCount
		}
		if input.Status != nil {
			if err := validateStatus(*input.Status); err != nil {
				return err
			}
			tenant.Status = *input.Status
		}
		tenant.UpdateTime = time.Now().UTC()
		if err := tx.UpdateTenant(ctx, tenant); err != nil {
			return err
		}
		updated = tenant
		return nil
	})
	return updated, err
}

func (s *Service) ListOrganizations(ctx context.Context, tenantID string) ([]OrganizationNode, error) {
	return s.ListOrganizationsWithAccess(ctx, tenantID, DataAccess{Scope: DataScopeAll})
}

// ListOrganizationsWithAccess returns organizations filtered by data scope.
func (s *Service) ListOrganizationsWithAccess(ctx context.Context, tenantID string, access DataAccess) ([]OrganizationNode, error) {
	if _, err := s.GetTenant(ctx, tenantID); err != nil {
		return nil, err
	}
	orgs, err := s.store.ListOrganizations(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return buildTree(FilterOrganizations(orgs, access)), nil
}

func (s *Service) ListOrganizationRecords(ctx context.Context, tenantID string) ([]Organization, error) {
	return s.store.ListOrganizations(ctx, tenantID)
}

func (s *Service) FindOrganizationByID(ctx context.Context, id string) (Organization, error) {
	return s.store.FindOrganizationByID(ctx, id)
}

func (s *Service) ListAssignmentRecords(ctx context.Context, tenantID, userID string) ([]OrganizationMembership, error) {
	return s.store.ListAssignments(ctx, tenantID, userID)
}

func (s *Service) ListUserRoleIDs(ctx context.Context, tenantID, userID string) ([]string, error) {
	return s.store.ListUserRoleIDs(ctx, tenantID, userID)
}

func (s *Service) ListMembershipTenantIDs(ctx context.Context, userID string) ([]string, error) {
	items, err := s.store.ListMembershipsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(items))
	for _, item := range items {
		if item.Status == StatusNormal {
			ids = append(ids, item.TenantID)
		}
	}
	return ids, nil
}

func (s *Service) CreateOrganization(ctx context.Context, tenantID string, input CreateOrganizationInput) (Organization, error) {
	org := Organization{
		TenantID:     tenantID,
		Name:         strings.TrimSpace(input.Name),
		Code:         strings.TrimSpace(input.Code),
		ParentID:     strings.TrimSpace(input.ParentID),
		Sort:         input.Sort,
		LeaderUserID: strings.TrimSpace(input.LeaderUserID),
		Phone:        strings.TrimSpace(input.Phone),
		Email:        strings.TrimSpace(input.Email),
		Status:       StatusNormal,
	}
	if err := validateName(org.Name); err != nil {
		return Organization{}, err
	}
	if err := validateOrganizationCode(org.Code); err != nil {
		return Organization{}, err
	}
	if org.Sort < 0 || org.Sort > 1000000 {
		return Organization{}, invalid("排序值不合法")
	}
	if err := validateMobile(org.Phone); err != nil {
		return Organization{}, err
	}
	if err := validateEmail(org.Email); err != nil {
		return Organization{}, err
	}
	var created Organization
	err := s.store.Transaction(ctx, func(tx Store) error {
		if _, err := tx.FindTenant(ctx, tenantID); errors.Is(err, ErrNotFound) {
			return notFound("租户不存在")
		} else if err != nil {
			return err
		}
		existing, err := tx.ListOrganizations(ctx, tenantID)
		if err != nil {
			return err
		}
		parent, err := findOrganizationByID(ctx, tx, org.ParentID)
		if err != nil {
			return err
		}
		if _, ok := tenantRoot(existing); ok {
			if parent.ID == "" || parent.TenantID != tenantID {
				return invalid("该租户已有主体组织。部门必须挂在本租户组织下；子公司请选择子公司租户并挂接它的主体组织")
			}
			if cycle, err := organizationCycle(ctx, tx, "", org.ParentID); err != nil {
				return err
			} else if cycle {
				return invalid("不能把部门移动到自己或自己的下级")
			}
		} else {
			org.IsTenantRoot = true
			if err := validateRootParent(parent, tenantID); err != nil {
				return err
			}
			if cycle, err := organizationCycle(ctx, tx, "", org.ParentID); err != nil {
				return err
			} else if cycle {
				return invalid("不能把组织挂到自己或自己的下级")
			}
		}
		if err := ensureUser(ctx, tx, org.LeaderUserID, "负责人不存在"); err != nil {
			return err
		}
		now := time.Now().UTC()
		org.ID = newID()
		org.CreateTime = now
		org.UpdateTime = now
		if err := tx.InsertOrganization(ctx, org); err != nil {
			return err
		}
		created = org
		return nil
	})
	return created, err
}

func (s *Service) UpdateOrganization(ctx context.Context, tenantID, id string, input UpdateOrganizationInput) (Organization, error) {
	if input.Name == nil && input.Code == nil && !input.ParentSet && input.Sort == nil && input.LeaderUserID == nil && input.Phone == nil && input.Email == nil && input.Status == nil {
		return Organization{}, invalid("没有可更新的字段")
	}
	var updated Organization
	err := s.store.Transaction(ctx, func(tx Store) error {
		org, err := tx.FindOrganization(ctx, tenantID, id)
		if errors.Is(err, ErrNotFound) {
			return notFound("部门不存在")
		}
		if err != nil {
			return err
		}
		if input.Name != nil {
			org.Name = strings.TrimSpace(*input.Name)
			if err := validateName(org.Name); err != nil {
				return err
			}
		}
		if input.Code != nil {
			org.Code = strings.TrimSpace(*input.Code)
			if err := validateOrganizationCode(org.Code); err != nil {
				return err
			}
		}
		if input.Sort != nil {
			if *input.Sort < 0 || *input.Sort > 1000000 {
				return invalid("排序值不合法")
			}
			org.Sort = *input.Sort
		}
		if input.Phone != nil {
			org.Phone = strings.TrimSpace(*input.Phone)
			if err := validateMobile(org.Phone); err != nil {
				return err
			}
		}
		if input.Email != nil {
			org.Email = strings.TrimSpace(*input.Email)
			if err := validateEmail(org.Email); err != nil {
				return err
			}
		}
		if input.LeaderUserID != nil {
			org.LeaderUserID = strings.TrimSpace(*input.LeaderUserID)
			if err := ensureUser(ctx, tx, org.LeaderUserID, "负责人不存在"); err != nil {
				return err
			}
		}
		if input.Status != nil {
			if err := validateStatus(*input.Status); err != nil {
				return err
			}
			org.Status = *input.Status
		}
		if input.ParentSet {
			parentID := ""
			if input.ParentID != nil {
				parentID = strings.TrimSpace(*input.ParentID)
			}
			parent, err := findOrganizationByID(ctx, tx, parentID)
			if err != nil {
				return err
			}
			if org.IsTenantRoot || org.ParentID == "" {
				org.IsTenantRoot = true
				if err := validateRootParent(parent, tenantID); err != nil {
					return err
				}
			} else if parent.ID == "" || parent.TenantID != tenantID {
				return invalid("部门必须属于当前租户的主体组织")
			}
			if cycle, err := organizationCycle(ctx, tx, org.ID, parentID); err != nil {
				return err
			} else if cycle {
				return invalid("不能把组织挂到自己或自己的下级")
			}
			org.ParentID = parentID
		}
		org.UpdateTime = time.Now().UTC()
		if err := tx.UpdateOrganization(ctx, org); err != nil {
			return err
		}
		updated = org
		return nil
	})
	return updated, err
}

func (s *Service) DeleteOrganization(ctx context.Context, tenantID, id string) error {
	return s.store.Transaction(ctx, func(tx Store) error {
		org, err := tx.FindOrganization(ctx, tenantID, id)
		if errors.Is(err, ErrNotFound) {
			return notFound("部门不存在")
		}
		if err != nil {
			return err
		}
		if org.IsTenantRoot || org.ParentID == "" {
			return conflict("主体组织不能删除")
		}
		children, err := tx.CountChildOrganizations(ctx, id)
		if err != nil {
			return err
		}
		assignments, err := tx.CountOrganizationAssignments(ctx, id)
		if err != nil {
			return err
		}
		if children > 0 || assignments > 0 {
			return conflict("部门仍有下级或任职，不能删除")
		}
		return tx.DeleteOrganization(ctx, id)
	})
}

type CreatePostInput struct {
	Code   string
	Name   string
	Sort   int
	Remark string
}

type UpdatePostInput struct {
	Code   *string
	Name   *string
	Sort   *int
	Status *int
	Remark *string
}

func (s *Service) ListUserPosts(ctx context.Context, tenantID, userID string) ([]Post, error) {
	var posts []Post
	err := s.store.Transaction(ctx, func(tx Store) error {
		if _, err := tx.FindMembership(ctx, tenantID, userID); errors.Is(err, ErrNotFound) {
			return notFound("成员不存在")
		} else if err != nil {
			return err
		}
		ids, err := tx.ListUserPostIDs(ctx, tenantID, userID)
		if err != nil {
			return err
		}
		posts = make([]Post, 0, len(ids))
		for _, id := range ids {
			post, err := tx.FindPost(ctx, tenantID, id)
			if errors.Is(err, ErrNotFound) {
				continue
			}
			if err != nil {
				return err
			}
			posts = append(posts, post)
		}
		return nil
	})
	if posts == nil {
		posts = []Post{}
	}
	return posts, err
}

func (s *Service) ReplaceUserPosts(ctx context.Context, tenantID, userID string, postIDs []string) ([]Post, error) {
	unique := make([]string, 0, len(postIDs))
	seen := map[string]struct{}{}
	for _, id := range postIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	var posts []Post
	err := s.store.Transaction(ctx, func(tx Store) error {
		if _, err := tx.FindMembership(ctx, tenantID, userID); errors.Is(err, ErrNotFound) {
			return notFound("成员不存在")
		} else if err != nil {
			return err
		}
		for _, id := range unique {
			post, err := tx.FindPost(ctx, tenantID, id)
			if errors.Is(err, ErrNotFound) {
				return invalid("岗位不存在")
			}
			if err != nil {
				return err
			}
			if post.Status != StatusNormal {
				return invalid("不能任职已停用的岗位")
			}
		}
		if err := tx.ReplaceUserPosts(ctx, tenantID, userID, unique); err != nil {
			return err
		}
		posts = make([]Post, 0, len(unique))
		for _, id := range unique {
			post, err := tx.FindPost(ctx, tenantID, id)
			if err != nil {
				return err
			}
			posts = append(posts, post)
		}
		return nil
	})
	if posts == nil {
		posts = []Post{}
	}
	return posts, err
}

func (s *Service) ListPosts(ctx context.Context, tenantID string, query ListQuery) (Page[Post], error) {
	page, pageSize, err := normalizePage(query.Page, query.PageSize)
	if err != nil {
		return Page[Post]{}, err
	}
	var result Page[Post]
	err = s.store.Transaction(ctx, func(tx Store) error {
		if _, err := tx.FindTenant(ctx, tenantID); errors.Is(err, ErrNotFound) {
			return notFound("租户不存在")
		} else if err != nil {
			return err
		}
		items, total, err := tx.ListPosts(ctx, tenantID, strings.TrimSpace(query.Keyword), page, pageSize)
		if err != nil {
			return err
		}
		result = newPage(items, page, pageSize, total)
		return nil
	})
	return result, err
}

func (s *Service) CreatePost(ctx context.Context, tenantID string, input CreatePostInput) (Post, error) {
	post := Post{
		TenantID: tenantID,
		Code:     strings.TrimSpace(input.Code),
		Name:     strings.TrimSpace(input.Name),
		Sort:     input.Sort,
		Status:   StatusNormal,
		Remark:   strings.TrimSpace(input.Remark),
	}
	if err := validatePostCode(post.Code); err != nil {
		return Post{}, err
	}
	if err := validatePostName(post.Name); err != nil {
		return Post{}, err
	}
	if post.Sort < 0 || post.Sort > 1000000 {
		return Post{}, invalid("排序值不合法")
	}
	if err := validateRemark(post.Remark); err != nil {
		return Post{}, err
	}
	now := time.Now().UTC()
	post.ID = newID()
	post.CreateTime = now
	post.UpdateTime = now
	err := s.store.Transaction(ctx, func(tx Store) error {
		if _, err := tx.FindTenant(ctx, tenantID); errors.Is(err, ErrNotFound) {
			return notFound("租户不存在")
		} else if err != nil {
			return err
		}
		return tx.InsertPost(ctx, post)
	})
	return post, err
}

func (s *Service) GetPost(ctx context.Context, tenantID, id string) (Post, error) {
	post, err := s.store.FindPost(ctx, tenantID, id)
	if errors.Is(err, ErrNotFound) {
		return Post{}, notFound("岗位不存在")
	}
	return post, err
}

func (s *Service) UpdatePost(ctx context.Context, tenantID, id string, input UpdatePostInput) (Post, error) {
	if input.Code == nil && input.Name == nil && input.Sort == nil && input.Status == nil && input.Remark == nil {
		return Post{}, invalid("没有可更新的字段")
	}
	var updated Post
	err := s.store.Transaction(ctx, func(tx Store) error {
		post, err := tx.FindPost(ctx, tenantID, id)
		if errors.Is(err, ErrNotFound) {
			return notFound("岗位不存在")
		}
		if err != nil {
			return err
		}
		if input.Code != nil {
			post.Code = strings.TrimSpace(*input.Code)
			if err := validatePostCode(post.Code); err != nil {
				return err
			}
		}
		if input.Name != nil {
			post.Name = strings.TrimSpace(*input.Name)
			if err := validatePostName(post.Name); err != nil {
				return err
			}
		}
		if input.Sort != nil {
			if *input.Sort < 0 || *input.Sort > 1000000 {
				return invalid("排序值不合法")
			}
			post.Sort = *input.Sort
		}
		if input.Remark != nil {
			post.Remark = strings.TrimSpace(*input.Remark)
			if err := validateRemark(post.Remark); err != nil {
				return err
			}
		}
		if input.Status != nil {
			if err := validateStatus(*input.Status); err != nil {
				return err
			}
			post.Status = *input.Status
		}
		post.UpdateTime = time.Now().UTC()
		if err := tx.UpdatePost(ctx, post); err != nil {
			return err
		}
		updated = post
		return nil
	})
	return updated, err
}

func (s *Service) DeletePost(ctx context.Context, tenantID, id string) error {
	return s.store.Transaction(ctx, func(tx Store) error {
		if _, err := tx.FindPost(ctx, tenantID, id); errors.Is(err, ErrNotFound) {
			return notFound("岗位不存在")
		} else if err != nil {
			return err
		}
		return tx.DeletePost(ctx, id)
	})
}

type CreateRoleInput struct {
	Name      string
	Code      string
	Sort      int
	Type      int
	DataScope int
	Remark    string
}

type UpdateRoleInput struct {
	Name      *string
	Code      *string
	Sort      *int
	Status    *int
	Type      *int
	DataScope *int
	Remark    *string
}

func (s *Service) ListRoles(ctx context.Context, tenantID string, query ListQuery) (Page[Role], error) {
	page, pageSize, err := normalizePage(query.Page, query.PageSize)
	if err != nil {
		return Page[Role]{}, err
	}
	var result Page[Role]
	err = s.store.Transaction(ctx, func(tx Store) error {
		if _, err := tx.FindTenant(ctx, tenantID); errors.Is(err, ErrNotFound) {
			return notFound("租户不存在")
		} else if err != nil {
			return err
		}
		items, total, err := tx.ListRoles(ctx, tenantID, strings.TrimSpace(query.Keyword), page, pageSize)
		if err != nil {
			return err
		}
		result = newPage(items, page, pageSize, total)
		return nil
	})
	return result, err
}

func (s *Service) CreateRole(ctx context.Context, tenantID string, input CreateRoleInput) (Role, error) {
	roleType := input.Type
	if roleType == 0 {
		roleType = RoleTypeCustom
	}
	dataScope := input.DataScope
	if dataScope == 0 {
		dataScope = DataScopeAll
	}
	role := Role{
		TenantID:  tenantID,
		Name:      strings.TrimSpace(input.Name),
		Code:      strings.TrimSpace(input.Code),
		Sort:      input.Sort,
		Status:    StatusNormal,
		Type:      roleType,
		DataScope: dataScope,
		Remark:    strings.TrimSpace(input.Remark),
	}
	if err := validateRole(role); err != nil {
		return Role{}, err
	}
	now := time.Now().UTC()
	role.ID = newID()
	role.CreateTime = now
	role.UpdateTime = now
	err := s.store.Transaction(ctx, func(tx Store) error {
		if _, err := tx.FindTenant(ctx, tenantID); errors.Is(err, ErrNotFound) {
			return notFound("租户不存在")
		} else if err != nil {
			return err
		}
		return tx.InsertRole(ctx, role)
	})
	return role, err
}

func (s *Service) GetRole(ctx context.Context, tenantID, id string) (Role, error) {
	role, err := s.store.FindRole(ctx, tenantID, id)
	if errors.Is(err, ErrNotFound) {
		return Role{}, notFound("角色不存在")
	}
	return role, err
}

func (s *Service) UpdateRole(ctx context.Context, tenantID, id string, input UpdateRoleInput) (Role, error) {
	if input.Name == nil && input.Code == nil && input.Sort == nil && input.Status == nil && input.Type == nil && input.DataScope == nil && input.Remark == nil {
		return Role{}, invalid("没有可更新的字段")
	}
	var updated Role
	err := s.store.Transaction(ctx, func(tx Store) error {
		role, err := tx.FindRole(ctx, tenantID, id)
		if errors.Is(err, ErrNotFound) {
			return notFound("角色不存在")
		}
		if err != nil {
			return err
		}
		if input.Name != nil {
			role.Name = strings.TrimSpace(*input.Name)
		}
		if input.Code != nil {
			role.Code = strings.TrimSpace(*input.Code)
		}
		if input.Sort != nil {
			role.Sort = *input.Sort
		}
		if input.Status != nil {
			role.Status = *input.Status
		}
		if input.Type != nil {
			role.Type = *input.Type
		}
		if input.DataScope != nil {
			role.DataScope = *input.DataScope
		}
		if input.Remark != nil {
			role.Remark = strings.TrimSpace(*input.Remark)
		}
		if err := validateRole(role); err != nil {
			return err
		}
		role.UpdateTime = time.Now().UTC()
		if err := tx.UpdateRole(ctx, role); err != nil {
			return err
		}
		if role.DataScope != DataScopeCustom {
			if err := tx.ReplaceRoleDepts(ctx, tenantID, id, nil); err != nil {
				return err
			}
		}
		updated = role
		return nil
	})
	return updated, err
}

func (s *Service) DeleteRole(ctx context.Context, tenantID, id string) error {
	return s.store.Transaction(ctx, func(tx Store) error {
		role, err := tx.FindRole(ctx, tenantID, id)
		if errors.Is(err, ErrNotFound) {
			return notFound("角色不存在")
		} else if err != nil {
			return err
		}
		if role.Type == RoleTypeSystem {
			return conflict("系统角色不能删除")
		}
		users, err := tx.CountRoleUsers(ctx, id)
		if err != nil {
			return err
		}
		if users > 0 {
			return conflict("角色仍有用户，不能删除")
		}
		if err := tx.ReplaceRoleMenus(ctx, tenantID, id, nil); err != nil {
			return err
		}
		if err := tx.ReplaceRoleDepts(ctx, tenantID, id, nil); err != nil {
			return err
		}
		return tx.DeleteRole(ctx, id)
	})
}

func (s *Service) ListRoleMenus(ctx context.Context, tenantID, roleID string) ([]string, error) {
	var menuIDs []string
	err := s.store.Transaction(ctx, func(tx Store) error {
		if _, err := tx.FindRole(ctx, tenantID, roleID); errors.Is(err, ErrNotFound) {
			return notFound("角色不存在")
		} else if err != nil {
			return err
		}
		ids, err := tx.ListRoleMenuIDs(ctx, tenantID, roleID)
		if err != nil {
			return err
		}
		menuIDs = ids
		return nil
	})
	return menuIDs, err
}

func (s *Service) ReplaceRoleMenus(ctx context.Context, tenantID, roleID string, menuIDs []string) ([]string, error) {
	normalized, err := uniqueIDs(menuIDs, "菜单标识不合法")
	if err != nil {
		return nil, err
	}
	err = s.store.Transaction(ctx, func(tx Store) error {
		if _, err := tx.FindRole(ctx, tenantID, roleID); errors.Is(err, ErrNotFound) {
			return notFound("角色不存在")
		} else if err != nil {
			return err
		}
		for _, menuID := range normalized {
			if _, err := tx.FindMenu(ctx, menuID); errors.Is(err, ErrNotFound) {
				return notFound("菜单不存在")
			} else if err != nil {
				return err
			}
		}
		return tx.ReplaceRoleMenus(ctx, tenantID, roleID, normalized)
	})
	return normalized, err
}

func (s *Service) ListRoleUsers(ctx context.Context, tenantID, roleID string) ([]RoleUserView, error) {
	var users []RoleUserView
	err := s.store.Transaction(ctx, func(tx Store) error {
		if _, err := tx.FindRole(ctx, tenantID, roleID); errors.Is(err, ErrNotFound) {
			return notFound("角色不存在")
		} else if err != nil {
			return err
		}
		items, err := tx.ListRoleUsers(ctx, tenantID, roleID)
		if err != nil {
			return err
		}
		users = items
		return nil
	})
	return users, err
}

func (s *Service) ReplaceRoleUsers(ctx context.Context, tenantID, roleID string, userIDs []string) ([]RoleUserView, error) {
	normalized, err := uniqueIDs(userIDs, "用户标识不合法")
	if err != nil {
		return nil, err
	}
	var users []RoleUserView
	err = s.store.Transaction(ctx, func(tx Store) error {
		if _, err := tx.FindRole(ctx, tenantID, roleID); errors.Is(err, ErrNotFound) {
			return notFound("角色不存在")
		} else if err != nil {
			return err
		}
		for _, userID := range normalized {
			if _, err := tx.FindUser(ctx, userID); errors.Is(err, ErrNotFound) {
				return notFound("账号不存在")
			} else if err != nil {
				return err
			}
			if _, err := tx.FindMembership(ctx, tenantID, userID); errors.Is(err, ErrNotFound) {
				return invalid("账号尚未加入该租户")
			} else if err != nil {
				return err
			}
		}
		if err := tx.ReplaceRoleUsers(ctx, tenantID, roleID, normalized); err != nil {
			return err
		}
		items, err := tx.ListRoleUsers(ctx, tenantID, roleID)
		if err != nil {
			return err
		}
		users = items
		return nil
	})
	return users, err
}

func validateRole(role Role) error {
	if err := validateRoleName(role.Name); err != nil {
		return err
	}
	if err := validateRoleCode(role.Code); err != nil {
		return err
	}
	if role.Sort < 0 || role.Sort > 1000000 {
		return invalid("排序值不合法")
	}
	if err := validateRoleType(role.Type); err != nil {
		return err
	}
	if err := validateDataScope(role.DataScope); err != nil {
		return err
	}
	if err := validateStatus(role.Status); err != nil {
		return err
	}
	return validateRemark(role.Remark)
}

func validateDataScope(scope int) error {
	switch scope {
	case DataScopeAll, DataScopeCustom, DataScopeDept, DataScopeDeptAndChild, DataScopeSelf:
		return nil
	default:
		return invalid("数据范围必须是 1 到 5")
	}
}

func uniqueIDs(ids []string, emptyMessage string) ([]string, error) {
	result := make([]string, 0, len(ids))
	seen := map[string]struct{}{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			return nil, invalid(emptyMessage)
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result, nil
}

type CreateMenuInput struct {
	Name          string
	Permission    string
	Type          int
	Sort          int
	ParentID      string
	Path          string
	Icon          string
	Component     string
	ComponentName string
	Visible       *bool
	KeepAlive     *bool
	AlwaysShow    *bool
}

type UpdateMenuInput struct {
	Name          *string
	Permission    *string
	Type          *int
	Sort          *int
	ParentSet     bool
	ParentID      *string
	Path          *string
	Icon          *string
	Component     *string
	ComponentName *string
	Status        *int
	Visible       *bool
	KeepAlive     *bool
	AlwaysShow    *bool
}

func (s *Service) ListMenus(ctx context.Context) ([]MenuNode, error) {
	menus, err := s.store.ListMenus(ctx)
	if err != nil {
		return nil, err
	}
	return buildMenuTree(menus), nil
}

func (s *Service) CreateMenu(ctx context.Context, input CreateMenuInput) (Menu, error) {
	menu := Menu{
		Name:          strings.TrimSpace(input.Name),
		Permission:    strings.TrimSpace(input.Permission),
		Type:          input.Type,
		Sort:          input.Sort,
		ParentID:      strings.TrimSpace(input.ParentID),
		Path:          strings.TrimSpace(input.Path),
		Icon:          strings.TrimSpace(input.Icon),
		Component:     strings.TrimSpace(input.Component),
		ComponentName: strings.TrimSpace(input.ComponentName),
		Status:        StatusNormal,
		Visible:       boolOr(input.Visible, true),
		KeepAlive:     boolOr(input.KeepAlive, true),
		AlwaysShow:    boolOr(input.AlwaysShow, true),
	}
	if menu.Icon == "" {
		menu.Icon = "#"
	}
	if err := validateMenu(menu); err != nil {
		return Menu{}, err
	}
	var created Menu
	err := s.store.Transaction(ctx, func(tx Store) error {
		if err := ensureMenuParentForType(ctx, tx, menu.ParentID, menu.Type); err != nil {
			return err
		}
		now := time.Now().UTC()
		menu.ID = newID()
		menu.CreateTime = now
		menu.UpdateTime = now
		if err := tx.InsertMenu(ctx, menu); err != nil {
			return err
		}
		created = menu
		return nil
	})
	return created, err
}

func (s *Service) GetMenu(ctx context.Context, id string) (Menu, error) {
	menu, err := s.store.FindMenu(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return Menu{}, notFound("菜单不存在")
	}
	return menu, err
}

func (s *Service) UpdateMenu(ctx context.Context, id string, input UpdateMenuInput) (Menu, error) {
	if !menuUpdatePresent(input) {
		return Menu{}, invalid("没有可更新的字段")
	}
	var updated Menu
	err := s.store.Transaction(ctx, func(tx Store) error {
		menu, err := tx.FindMenu(ctx, id)
		if errors.Is(err, ErrNotFound) {
			return notFound("菜单不存在")
		}
		if err != nil {
			return err
		}
		if input.Name != nil {
			menu.Name = strings.TrimSpace(*input.Name)
		}
		if input.Permission != nil {
			menu.Permission = strings.TrimSpace(*input.Permission)
		}
		if input.Type != nil {
			menu.Type = *input.Type
		}
		if input.Sort != nil {
			menu.Sort = *input.Sort
		}
		if input.Path != nil {
			menu.Path = strings.TrimSpace(*input.Path)
		}
		if input.Icon != nil {
			menu.Icon = strings.TrimSpace(*input.Icon)
			if menu.Icon == "" {
				menu.Icon = "#"
			}
		}
		if input.Component != nil {
			menu.Component = strings.TrimSpace(*input.Component)
		}
		if input.ComponentName != nil {
			menu.ComponentName = strings.TrimSpace(*input.ComponentName)
		}
		if input.Status != nil {
			menu.Status = *input.Status
		}
		if input.Visible != nil {
			menu.Visible = *input.Visible
		}
		if input.KeepAlive != nil {
			menu.KeepAlive = *input.KeepAlive
		}
		if input.AlwaysShow != nil {
			menu.AlwaysShow = *input.AlwaysShow
		}
		if input.ParentSet {
			parentID := ""
			if input.ParentID != nil {
				parentID = strings.TrimSpace(*input.ParentID)
			}
			menus, err := tx.ListMenus(ctx)
			if err != nil {
				return err
			}
			if menuParentCreatesCycle(menus, menu.ID, parentID) {
				return invalid("不能把菜单移动到自己或自己的下级")
			}
			menu.ParentID = parentID
		}
		if err := ensureMenuParentForType(ctx, tx, menu.ParentID, menu.Type); err != nil {
			return err
		}
		if err := validateMenu(menu); err != nil {
			return err
		}
		menu.UpdateTime = time.Now().UTC()
		if err := tx.UpdateMenu(ctx, menu); err != nil {
			return err
		}
		updated = menu
		return nil
	})
	return updated, err
}

func (s *Service) DeleteMenu(ctx context.Context, id string) error {
	return s.store.Transaction(ctx, func(tx Store) error {
		if _, err := tx.FindMenu(ctx, id); errors.Is(err, ErrNotFound) {
			return notFound("菜单不存在")
		} else if err != nil {
			return err
		}
		children, err := tx.CountChildMenus(ctx, id)
		if err != nil {
			return err
		}
		if children > 0 {
			return conflict("菜单仍有下级，不能删除")
		}
		return tx.DeleteMenu(ctx, id)
	})
}

func validateMenu(menu Menu) error {
	if err := validateMenuName(menu.Name); err != nil {
		return err
	}
	if err := validateMenuType(menu.Type); err != nil {
		return err
	}
	if err := validateOptionalText(menu.Permission, 100, "权限标识长度不能超过 100 个字符"); err != nil {
		return err
	}
	if menu.Type == MenuTypeButton {
		if strings.TrimSpace(menu.Permission) == "" {
			return invalid("按钮必须填写权限标识")
		}
		if strings.TrimSpace(menu.ParentID) == "" {
			return invalid("按钮必须挂在菜单下")
		}
	}
	if menu.Sort < 0 || menu.Sort > 1000000 {
		return invalid("排序值不合法")
	}
	if err := validateOptionalText(menu.Path, 200, "路由地址长度不能超过 200 个字符"); err != nil {
		return err
	}
	if err := validateOptionalText(menu.Icon, 100, "菜单图标长度不能超过 100 个字符"); err != nil {
		return err
	}
	if err := validateOptionalText(menu.Component, 255, "组件路径长度不能超过 255 个字符"); err != nil {
		return err
	}
	if err := validateOptionalText(menu.ComponentName, 255, "组件名长度不能超过 255 个字符"); err != nil {
		return err
	}
	return validateStatus(menu.Status)
}

func menuUpdatePresent(input UpdateMenuInput) bool {
	return input.Name != nil || input.Permission != nil || input.Type != nil || input.Sort != nil || input.ParentSet ||
		input.Path != nil || input.Icon != nil || input.Component != nil || input.ComponentName != nil ||
		input.Status != nil || input.Visible != nil || input.KeepAlive != nil || input.AlwaysShow != nil
}

func boolOr(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func ensureMenuParent(ctx context.Context, tx Store, parentID string) error {
	return ensureMenuParentForType(ctx, tx, parentID, 0)
}

func ensureMenuParentForType(ctx context.Context, tx Store, parentID string, menuType int) error {
	if parentID == "" {
		return nil
	}
	parent, err := tx.FindMenu(ctx, parentID)
	if errors.Is(err, ErrNotFound) {
		return notFound("上级菜单不存在")
	}
	if err != nil {
		return err
	}
	if menuType == MenuTypeButton && parent.Type != MenuTypeMenu {
		return invalid("按钮只能挂在菜单类型节点下")
	}
	if parent.Type == MenuTypeButton {
		return invalid("按钮不能作为上级菜单")
	}
	return nil
}

func (s *Service) ListUsers(ctx context.Context, query ListQuery) (Page[User], error) {
	return s.ListUsersWithAccess(ctx, "", query, DataAccess{Scope: DataScopeAll})
}

// ListUsersWithAccess filters global users by the caller's tenant data scope.
func (s *Service) ListUsersWithAccess(ctx context.Context, tenantID string, query ListQuery, access DataAccess) (Page[User], error) {
	page, pageSize, err := normalizePage(query.Page, query.PageSize)
	if err != nil {
		return Page[User]{}, err
	}
	if access.AllowsAll() {
		items, total, err := s.store.ListUsers(ctx, strings.TrimSpace(query.Keyword), page, pageSize)
		if err != nil {
			return Page[User]{}, err
		}
		return newPage(stripPasswords(items), page, pageSize, total), nil
	}
	items, _, err := s.store.ListUsers(ctx, strings.TrimSpace(query.Keyword), 1, 10000)
	if err != nil {
		return Page[User]{}, err
	}
	filtered := make([]User, 0, len(items))
	for _, user := range items {
		if access.AllowsSelfOnly() {
			if user.ID == access.UserID {
				filtered = append(filtered, user)
			}
			continue
		}
		if user.ID == access.UserID {
			filtered = append(filtered, user)
			continue
		}
		if tenantID == "" {
			continue
		}
		assignments, err := s.store.ListAssignments(ctx, tenantID, user.ID)
		if err != nil {
			return Page[User]{}, err
		}
		if MemberVisible(user.ID, assignments, access) {
			filtered = append(filtered, user)
		}
	}
	total := int64(len(filtered))
	start := (page - 1) * pageSize
	if start > len(filtered) {
		start = len(filtered)
	}
	end := start + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	return newPage(stripPasswords(filtered[start:end]), page, pageSize, total), nil
}

func (s *Service) CreateUser(ctx context.Context, input CreateUserInput) (User, error) {
	user := User{
		Username: strings.TrimSpace(input.Username),
		Nickname: strings.TrimSpace(input.Nickname),
		Remark:   strings.TrimSpace(input.Remark),
		Email:    strings.TrimSpace(input.Email),
		Mobile:   strings.TrimSpace(input.Mobile),
		Sex:      input.Sex,
		Avatar:   strings.TrimSpace(input.Avatar),
		Status:   StatusNormal,
	}
	if err := validateUsername(user.Username); err != nil {
		return User{}, err
	}
	if err := validatePassword(input.Password); err != nil {
		return User{}, err
	}
	hash, err := hashPassword(input.Password)
	if err != nil {
		return User{}, err
	}
	user.PasswordHash = hash
	if err := validateName(user.Nickname); err != nil {
		return User{}, err
	}
	if err := validateRemark(user.Remark); err != nil {
		return User{}, err
	}
	if err := validateEmail(user.Email); err != nil {
		return User{}, err
	}
	if err := validateMobile(user.Mobile); err != nil {
		return User{}, err
	}
	if err := validateSex(user.Sex); err != nil {
		return User{}, err
	}
	if err := validateAvatar(user.Avatar); err != nil {
		return User{}, err
	}
	now := time.Now().UTC()
	user.ID = newID()
	user.CreateTime = now
	user.UpdateTime = now
	if err := s.store.InsertUser(ctx, user); err != nil {
		return User{}, err
	}
	user.PasswordHash = ""
	return user, nil
}

func (s *Service) GetUser(ctx context.Context, id string) (UserDetail, error) {
	var detail UserDetail
	err := s.store.Transaction(ctx, func(tx Store) error {
		user, err := tx.FindUser(ctx, id)
		if errors.Is(err, ErrNotFound) {
			return notFound("账号不存在")
		}
		if err != nil {
			return err
		}
		memberships, err := tx.ListMembershipsByUser(ctx, id)
		if err != nil {
			return err
		}
		views := make([]UserMembership, 0, len(memberships))
		for _, membership := range memberships {
			tenant, err := tx.FindTenant(ctx, membership.TenantID)
			if err != nil {
				return err
			}
			assignments, err := assignmentViews(ctx, tx, membership.TenantID, membership.UserID)
			if err != nil {
				return err
			}
			rootID, rootName, err := rootOrganizationView(ctx, tx, membership.TenantID)
			if err != nil {
				return err
			}
			views = append(views, UserMembership{
				TenantID:             tenant.ID,
				TenantName:           tenant.Name,
				TenantCode:           tenant.Code,
				Status:               membership.Status,
				RootOrganizationID:   rootID,
				RootOrganizationName: rootName,
				Organizations:        assignments,
			})
		}
		user.PasswordHash = ""
		detail = UserDetail{User: user, Memberships: views}
		return nil
	})
	if detail.Memberships == nil {
		detail.Memberships = []UserMembership{}
	}
	return detail, err
}

func (s *Service) GetOwnProfile(ctx context.Context, userID string) (User, error) {
	user, err := s.store.FindUser(ctx, userID)
	if errors.Is(err, ErrNotFound) {
		return User{}, notFound("账号不存在")
	}
	if err != nil {
		return User{}, err
	}
	user.PasswordHash = ""
	return user, nil
}

func (s *Service) UpdateOwnProfile(ctx context.Context, userID string, input UpdateUserInput) (User, error) {
	input.Password = nil
	input.Status = nil
	return s.UpdateUser(ctx, userID, input)
}

func (s *Service) ChangeOwnPassword(ctx context.Context, userID, currentPassword, nextPassword string) error {
	if strings.TrimSpace(currentPassword) == "" || strings.TrimSpace(nextPassword) == "" {
		return invalid("请填写当前密码和新密码")
	}
	if currentPassword == nextPassword {
		return invalid("新密码不能与当前密码相同")
	}
	user, err := s.store.FindUser(ctx, userID)
	if errors.Is(err, ErrNotFound) {
		return notFound("账号不存在")
	}
	if err != nil {
		return err
	}
	if !checkPassword(user.PasswordHash, currentPassword) {
		return unauthorized("当前密码不正确")
	}
	hash, err := hashPassword(nextPassword)
	if err != nil {
		return err
	}
	return s.store.UpdateUserPassword(ctx, userID, hash)
}

func (s *Service) UpdateUser(ctx context.Context, id string, input UpdateUserInput) (User, error) {
	if input.Nickname == nil && input.Password == nil && input.Remark == nil && input.Email == nil && input.Mobile == nil && input.Sex == nil && input.Avatar == nil && input.Status == nil {
		return User{}, invalid("没有可更新的字段")
	}
	var updated User
	err := s.store.Transaction(ctx, func(tx Store) error {
		user, err := tx.FindUser(ctx, id)
		if errors.Is(err, ErrNotFound) {
			return notFound("账号不存在")
		}
		if err != nil {
			return err
		}
		if input.Password != nil {
			hash, err := hashPassword(*input.Password)
			if err != nil {
				return err
			}
			if err := tx.UpdateUserPassword(ctx, id, hash); err != nil {
				return err
			}
		}
		if input.Nickname != nil {
			user.Nickname = strings.TrimSpace(*input.Nickname)
			if err := validateName(user.Nickname); err != nil {
				return err
			}
		}
		if input.Remark != nil {
			user.Remark = strings.TrimSpace(*input.Remark)
			if err := validateRemark(user.Remark); err != nil {
				return err
			}
		}
		if input.Email != nil {
			user.Email = strings.TrimSpace(*input.Email)
			if err := validateEmail(user.Email); err != nil {
				return err
			}
		}
		if input.Sex != nil {
			if err := validateSex(*input.Sex); err != nil {
				return err
			}
			user.Sex = *input.Sex
		}
		if input.Avatar != nil {
			user.Avatar = strings.TrimSpace(*input.Avatar)
			if err := validateAvatar(user.Avatar); err != nil {
				return err
			}
		}
		if input.Mobile != nil {
			user.Mobile = strings.TrimSpace(*input.Mobile)
			if err := validateMobile(user.Mobile); err != nil {
				return err
			}
		}
		if input.Status != nil {
			if err := validateStatus(*input.Status); err != nil {
				return err
			}
			user.Status = *input.Status
		}
		profileChanged := input.Nickname != nil || input.Remark != nil || input.Email != nil || input.Mobile != nil ||
			input.Sex != nil || input.Avatar != nil || input.Status != nil
		if profileChanged {
			user.UpdateTime = time.Now().UTC()
			if err := tx.UpdateUser(ctx, user); err != nil {
				return err
			}
		}
		fresh, err := tx.FindUser(ctx, id)
		if err != nil {
			return err
		}
		fresh.PasswordHash = ""
		updated = fresh
		return nil
	})
	return updated, err
}

func (s *Service) ListMembers(ctx context.Context, tenantID string, query ListQuery) (Page[Member], error) {
	return s.ListMembersWithAccess(ctx, tenantID, query, DataAccess{Scope: DataScopeAll})
}

// ListMembersWithAccess returns tenant members filtered by data scope.
func (s *Service) ListMembersWithAccess(ctx context.Context, tenantID string, query ListQuery, access DataAccess) (Page[Member], error) {
	page, pageSize, err := normalizePage(query.Page, query.PageSize)
	if err != nil {
		return Page[Member]{}, err
	}
	var result Page[Member]
	err = s.store.Transaction(ctx, func(tx Store) error {
		if _, err := tx.FindTenant(ctx, tenantID); errors.Is(err, ErrNotFound) {
			return notFound("租户不存在")
		} else if err != nil {
			return err
		}
		fetchPage, fetchSize := page, pageSize
		if !access.AllowsAll() {
			fetchPage, fetchSize = 1, 10000
		}
		rows, total, err := tx.ListMembers(ctx, tenantID, strings.TrimSpace(query.Keyword), fetchPage, fetchSize)
		if err != nil {
			return err
		}
		orgs, err := tx.ListOrganizations(ctx, tenantID)
		if err != nil {
			return err
		}
		names := map[string]string{}
		for _, org := range orgs {
			names[org.ID] = org.Name
		}
		items := make([]Member, 0, len(rows))
		for _, row := range rows {
			assignments, err := tx.ListAssignments(ctx, tenantID, row.Membership.UserID)
			if err != nil {
				return err
			}
			if !MemberVisible(row.Membership.UserID, assignments, access) {
				continue
			}
			items = append(items, toMember(row, assignments, names))
		}
		if access.AllowsAll() {
			result = newPage(items, page, pageSize, total)
			return nil
		}
		total = int64(len(items))
		start := (page - 1) * pageSize
		if start > len(items) {
			start = len(items)
		}
		end := start + pageSize
		if end > len(items) {
			end = len(items)
		}
		result = newPage(items[start:end], page, pageSize, total)
		return nil
	})
	return result, err
}

func stripPasswords(users []User) []User {
	out := make([]User, len(users))
	for i, user := range users {
		user.PasswordHash = ""
		out[i] = user
	}
	return out
}

func (s *Service) CreateMember(ctx context.Context, tenantID string, input CreateMemberInput) (Member, error) {
	if strings.TrimSpace(input.UserID) == "" {
		return Member{}, invalid("账号标识不合法")
	}
	if err := validateStatus(input.Status); err != nil {
		return Member{}, err
	}
	if err := validateAssignments(input.AssignmentInput); err != nil {
		return Member{}, err
	}
	var created Member
	err := s.store.Transaction(ctx, func(tx Store) error {
		if _, err := tx.FindTenant(ctx, tenantID); errors.Is(err, ErrNotFound) {
			return notFound("租户不存在")
		} else if err != nil {
			return err
		}
		user, err := tx.FindUser(ctx, input.UserID)
		if errors.Is(err, ErrNotFound) {
			return notFound("账号不存在")
		}
		if err != nil {
			return err
		}
		if _, err := tx.FindMembership(ctx, tenantID, input.UserID); err == nil {
			return conflict("账号已在该租户中")
		} else if !errors.Is(err, ErrNotFound) {
			return err
		}
		if err := ensureOrganizations(ctx, tx, tenantID, input.OrganizationIDs); err != nil {
			return err
		}
		now := time.Now().UTC()
		membership := TenantMembership{
			ID:         newID(),
			TenantID:   tenantID,
			UserID:     input.UserID,
			Status:     input.Status,
			CreateTime: now,
			UpdateTime: now,
		}
		if err := tx.InsertMembership(ctx, membership); err != nil {
			return err
		}
		assignments := newAssignments(tenantID, input.UserID, input.AssignmentInput, now)
		if err := tx.ReplaceAssignments(ctx, tenantID, input.UserID, assignments); err != nil {
			return err
		}
		orgs, err := tx.ListOrganizations(ctx, tenantID)
		if err != nil {
			return err
		}
		created = toMember(MemberRecord{Membership: membership, Username: user.Username, Nickname: user.Nickname}, assignments, orgNames(orgs))
		return nil
	})
	return created, err
}

func (s *Service) ReplaceMemberAssignments(ctx context.Context, tenantID, userID string, input AssignmentInput) (Member, error) {
	if err := validateAssignments(input); err != nil {
		return Member{}, err
	}
	var updated Member
	err := s.store.Transaction(ctx, func(tx Store) error {
		membership, err := tx.FindMembership(ctx, tenantID, userID)
		if errors.Is(err, ErrNotFound) {
			return notFound("成员不存在")
		}
		if err != nil {
			return err
		}
		user, err := tx.FindUser(ctx, userID)
		if err != nil {
			return err
		}
		if err := ensureOrganizations(ctx, tx, tenantID, input.OrganizationIDs); err != nil {
			return err
		}
		now := time.Now().UTC()
		assignments := newAssignments(tenantID, userID, input, now)
		if err := tx.ReplaceAssignments(ctx, tenantID, userID, assignments); err != nil {
			return err
		}
		orgs, err := tx.ListOrganizations(ctx, tenantID)
		if err != nil {
			return err
		}
		updated = toMember(MemberRecord{Membership: membership, Username: user.Username, Nickname: user.Nickname}, assignments, orgNames(orgs))
		return nil
	})
	return updated, err
}

func (s *Service) RemoveMember(ctx context.Context, tenantID, userID string) error {
	return s.store.Transaction(ctx, func(tx Store) error {
		if _, err := tx.FindMembership(ctx, tenantID, userID); errors.Is(err, ErrNotFound) {
			return notFound("成员不存在")
		} else if err != nil {
			return err
		}
		if err := tx.ReplaceAssignments(ctx, tenantID, userID, nil); err != nil {
			return err
		}
		if err := tx.ClearUserRoles(ctx, tenantID, userID); err != nil {
			return err
		}
		if err := tx.ClearUserPosts(ctx, tenantID, userID); err != nil {
			return err
		}
		return tx.DeleteMembership(ctx, tenantID, userID)
	})
}

func ensureUser(ctx context.Context, tx Store, userID, missing string) error {
	if userID == "" {
		return nil
	}
	if _, err := tx.FindUser(ctx, userID); errors.Is(err, ErrNotFound) {
		return invalid(missing)
	} else if err != nil {
		return err
	}
	return nil
}

func rootOrganization(tenant Tenant, now time.Time) Organization {
	return Organization{
		ID:           newID(),
		TenantID:     tenant.ID,
		Name:         tenant.Name,
		Code:         tenant.Code,
		IsTenantRoot: true,
		Status:       StatusNormal,
		CreateTime:   now,
		UpdateTime:   now,
	}
}

func rootOrganizationView(ctx context.Context, tx Store, tenantID string) (string, string, error) {
	orgs, err := tx.ListOrganizations(ctx, tenantID)
	if err != nil {
		return "", "", err
	}
	if root, ok := tenantRoot(orgs); ok {
		return root.ID, root.Name, nil
	}
	for _, org := range orgs {
		if org.ParentID == "" {
			return org.ID, org.Name, nil
		}
	}
	return "", "", nil
}

func tenantRoot(orgs []Organization) (Organization, bool) {
	for _, org := range orgs {
		if org.IsTenantRoot {
			return org, true
		}
	}
	return Organization{}, false
}

func findOrganizationByID(ctx context.Context, tx Store, id string) (Organization, error) {
	if id == "" {
		return Organization{}, nil
	}
	org, err := tx.FindOrganizationByID(ctx, id)
	if errors.Is(err, ErrNotFound) {
		return Organization{}, invalid("上级组织不存在")
	}
	return org, err
}

func isCompanyRoot(org Organization) bool {
	return org.ID != "" && (org.IsTenantRoot || org.ParentID == "")
}

func validateRootParent(parent Organization, tenantID string) error {
	if parent.ID == "" {
		return nil
	}
	if !isCompanyRoot(parent) {
		return invalid("子公司只能挂在其他租户的主体组织下面")
	}
	if parent.TenantID == tenantID {
		return invalid("主体组织不能挂到本租户的组织下面")
	}
	return nil
}

func organizationCycle(ctx context.Context, tx Store, id, parentID string) (bool, error) {
	seen := map[string]struct{}{}
	if id != "" {
		seen[id] = struct{}{}
	}
	current := parentID
	for current != "" {
		if _, ok := seen[current]; ok {
			return true, nil
		}
		seen[current] = struct{}{}
		parent, err := tx.FindOrganizationByID(ctx, current)
		if errors.Is(err, ErrNotFound) {
			return false, invalid("上级组织不存在")
		}
		if err != nil {
			return false, err
		}
		current = parent.ParentID
	}
	return false, nil
}

func ensureParent(ctx context.Context, tx Store, tenantID, parentID string) error {
	if parentID == "" {
		return nil
	}
	if _, err := tx.FindOrganization(ctx, tenantID, parentID); errors.Is(err, ErrNotFound) {
		return invalid("上级部门不存在")
	} else if err != nil {
		return err
	}
	return nil
}

func ensureOrganizations(ctx context.Context, tx Store, tenantID string, ids []string) error {
	for _, id := range ids {
		if _, err := tx.FindOrganization(ctx, tenantID, id); errors.Is(err, ErrNotFound) {
			return invalid("部门不属于该租户")
		} else if err != nil {
			return err
		}
	}
	return nil
}

func assignmentViews(ctx context.Context, tx Store, tenantID, userID string) ([]Assignment, error) {
	rows, err := tx.ListAssignments(ctx, tenantID, userID)
	if err != nil {
		return nil, err
	}
	orgs, err := tx.ListOrganizations(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return viewAssignments(rows, orgNames(orgs)), nil
}

func newAssignments(tenantID, userID string, input AssignmentInput, now time.Time) []OrganizationMembership {
	rows := make([]OrganizationMembership, 0, len(input.OrganizationIDs))
	for _, id := range input.OrganizationIDs {
		rows = append(rows, OrganizationMembership{
			ID:             newID(),
			TenantID:       tenantID,
			OrganizationID: id,
			UserID:         userID,
			IsPrimary:      id == input.PrimaryOrganizationID,
			CreateTime:     now,
			UpdateTime:     now,
		})
	}
	return rows
}

func toMember(row MemberRecord, assignments []OrganizationMembership, names map[string]string) Member {
	views := viewAssignments(assignments, names)
	primary := ""
	for _, item := range views {
		if item.IsPrimary {
			primary = item.ID
		}
	}
	return Member{
		UserID:                row.Membership.UserID,
		Username:              row.Username,
		Nickname:              row.Nickname,
		Status:                row.Membership.Status,
		PrimaryOrganizationID: primary,
		Organizations:         views,
	}
}

func viewAssignments(rows []OrganizationMembership, names map[string]string) []Assignment {
	views := make([]Assignment, 0, len(rows))
	for _, row := range rows {
		views = append(views, Assignment{ID: row.OrganizationID, Name: names[row.OrganizationID], IsPrimary: row.IsPrimary})
	}
	return views
}

func orgNames(orgs []Organization) map[string]string {
	names := make(map[string]string, len(orgs))
	for _, org := range orgs {
		names[org.ID] = org.Name
	}
	return names
}

func buildTree(orgs []Organization) []OrganizationNode {
	nodes := make(map[string]*OrganizationNode, len(orgs))
	order := make([]string, 0, len(orgs))
	for _, org := range orgs {
		node := OrganizationNode{Organization: org, Children: []OrganizationNode{}}
		nodes[org.ID] = &node
		order = append(order, org.ID)
	}
	roots := make([]*OrganizationNode, 0)
	for _, id := range order {
		node := nodes[id]
		if node.ParentID == "" || nodes[node.ParentID] == nil {
			roots = append(roots, node)
			continue
		}
		parent := nodes[node.ParentID]
		parent.Children = append(parent.Children, *node)
	}
	result := make([]OrganizationNode, 0, len(roots))
	for _, root := range roots {
		result = append(result, freezeTree(root, nodes))
	}
	return result
}

func buildMenuTree(menus []Menu) []MenuNode {
	nodes := make(map[string]*MenuNode, len(menus))
	order := make([]string, 0, len(menus))
	for _, menu := range menus {
		node := MenuNode{Menu: menu, Children: []MenuNode{}}
		nodes[menu.ID] = &node
		order = append(order, menu.ID)
	}
	roots := make([]*MenuNode, 0)
	for _, id := range order {
		node := nodes[id]
		if node.ParentID == "" || nodes[node.ParentID] == nil {
			roots = append(roots, node)
			continue
		}
		parent := nodes[node.ParentID]
		parent.Children = append(parent.Children, *node)
	}
	result := make([]MenuNode, 0, len(roots))
	for _, root := range roots {
		result = append(result, freezeMenuTree(root, nodes))
	}
	return result
}

func freezeMenuTree(node *MenuNode, nodes map[string]*MenuNode) MenuNode {
	copied := *node
	copied.Children = make([]MenuNode, 0, len(node.Children))
	for _, child := range node.Children {
		current := nodes[child.ID]
		if current == nil {
			copied.Children = append(copied.Children, child)
			continue
		}
		copied.Children = append(copied.Children, freezeMenuTree(current, nodes))
	}
	return copied
}

func freezeTree(node *OrganizationNode, nodes map[string]*OrganizationNode) OrganizationNode {
	copied := *node
	copied.Children = make([]OrganizationNode, 0, len(node.Children))
	for _, child := range node.Children {
		current := nodes[child.ID]
		if current == nil {
			copied.Children = append(copied.Children, child)
			continue
		}
		copied.Children = append(copied.Children, freezeTree(current, nodes))
	}
	return copied
}

func newPage[T any](items []T, page, pageSize int, total int64) Page[T] {
	if items == nil {
		items = []T{}
	}
	return Page[T]{Items: items, Page: page, PageSize: pageSize, Total: total}
}

func newID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		panic(err)
	}
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", buf[0:4], buf[4:6], buf[6:8], buf[8:10], buf[10:16])
}
