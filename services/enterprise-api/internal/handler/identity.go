package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"qingzhi/enterprise-api/internal/auth"
	"qingzhi/enterprise-api/internal/identity"
	"qingzhi/enterprise-api/internal/response"
	"qingzhi/enterprise-api/internal/security"
)

var uuidPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// Identity serves account, tenant, and organization management routes.
type Identity struct {
	Logger   *slog.Logger
	Database Pinger
	Service  *identity.Service
	Security *security.Service
	Timeout  time.Duration
}

// Register adds the identity routes. They stay registered when the database is down.
// When enforce is true, routes require JWT authentication plus menu permissions.
func (h Identity) Register(group *gin.RouterGroup, enforce bool) {
	perm := func(permission string) gin.HandlerFunc {
		if !enforce {
			return func(c *gin.Context) { c.Next() }
		}
		return auth.RequirePermission(permission)
	}
	tenant := func(handlers ...gin.HandlerFunc) []gin.HandlerFunc {
		if !enforce {
			return handlers
		}
		return append([]gin.HandlerFunc{auth.RequireTenantMatch("tenantId")}, handlers...)
	}

	group.GET("/tenants", perm("system:tenant:query"), h.listTenants)
	group.POST("/tenants", perm("system:tenant:create"), h.createTenant)
	group.GET("/tenants/:tenantId", tenant(perm("system:tenant:query"), h.getTenant)...)
	group.PATCH("/tenants/:tenantId", tenant(perm("system:tenant:update"), h.updateTenant)...)

	group.GET("/tenants/:tenantId/organizations", tenant(perm("system:organization:query"), h.listOrganizations)...)
	group.POST("/tenants/:tenantId/organizations", tenant(perm("system:organization:create"), h.createOrganization)...)
	group.PATCH("/tenants/:tenantId/organizations/:organizationId", tenant(perm("system:organization:update"), h.updateOrganization)...)
	group.DELETE("/tenants/:tenantId/organizations/:organizationId", tenant(perm("system:organization:delete"), h.deleteOrganization)...)

	group.GET("/tenants/:tenantId/posts", tenant(perm("system:post:query"), h.listPosts)...)
	group.POST("/tenants/:tenantId/posts", tenant(perm("system:post:create"), h.createPost)...)
	group.GET("/tenants/:tenantId/posts/:postId", tenant(perm("system:post:query"), h.getPost)...)
	group.PATCH("/tenants/:tenantId/posts/:postId", tenant(perm("system:post:update"), h.updatePost)...)
	group.DELETE("/tenants/:tenantId/posts/:postId", tenant(perm("system:post:delete"), h.deletePost)...)

	group.GET("/tenants/:tenantId/roles", tenant(perm("system:role:query"), h.listRoles)...)
	group.POST("/tenants/:tenantId/roles", tenant(perm("system:role:create"), h.createRole)...)
	group.GET("/tenants/:tenantId/roles/:roleId", tenant(perm("system:role:query"), h.getRole)...)
	group.PATCH("/tenants/:tenantId/roles/:roleId", tenant(perm("system:role:update"), h.updateRole)...)
	group.DELETE("/tenants/:tenantId/roles/:roleId", tenant(perm("system:role:delete"), h.deleteRole)...)
	group.GET("/tenants/:tenantId/roles/:roleId/menus", tenant(perm("system:role:query"), h.listRoleMenus)...)
	group.PUT("/tenants/:tenantId/roles/:roleId/menus", tenant(perm("system:role:update"), h.replaceRoleMenus)...)
	group.GET("/tenants/:tenantId/roles/:roleId/depts", tenant(perm("system:role:query"), h.listRoleDepts)...)
	group.PUT("/tenants/:tenantId/roles/:roleId/depts", tenant(perm("system:role:update"), h.replaceRoleDepts)...)
	group.GET("/tenants/:tenantId/roles/:roleId/users", tenant(perm("system:role:query"), h.listRoleUsers)...)
	group.PUT("/tenants/:tenantId/roles/:roleId/users", tenant(perm("system:role:update"), h.replaceRoleUsers)...)

	group.GET("/menus", perm("system:menu:query"), h.listMenus)
	group.POST("/menus", perm("system:menu:create"), h.createMenu)
	group.GET("/menus/:menuId", perm("system:menu:query"), h.getMenu)
	group.PATCH("/menus/:menuId", perm("system:menu:update"), h.updateMenu)
	group.DELETE("/menus/:menuId", perm("system:menu:delete"), h.deleteMenu)

	group.GET("/users", perm("system:user:query"), h.listUsers)
	group.POST("/users", perm("system:user:create"), h.createUser)
	group.GET("/users/:userId", perm("system:user:query"), h.getUser)
	group.PATCH("/users/:userId", perm("system:user:update"), h.updateUser)

	group.GET("/tenants/:tenantId/members", tenant(perm("system:user:query"), h.listMembers)...)
	group.POST("/tenants/:tenantId/members", tenant(perm("system:user:create"), h.createMember)...)
	group.PUT("/tenants/:tenantId/members/:userId", tenant(perm("system:user:update"), h.replaceMember)...)
	group.DELETE("/tenants/:tenantId/members/:userId", tenant(perm("system:user:delete"), h.removeMember)...)
}

func (h Identity) principalAccess(c *gin.Context) identity.DataAccess {
	if principal, ok := auth.GetPrincipal(c); ok {
		return principal.Access
	}
	return identity.DataAccess{Scope: identity.DataScopeAll}
}

func (h Identity) principalTenant(c *gin.Context) string {
	if principal, ok := auth.GetPrincipal(c); ok {
		return principal.TenantID
	}
	return ""
}

func (h Identity) listTenants(c *gin.Context) {
	if !h.available(c) {
		return
	}
	query, ok := listQuery(c)
	if !ok {
		return
	}
	page, err := h.Service.ListTenants(c.Request.Context(), query)
	h.write(c, page, err)
}

func (h Identity) createTenant(c *gin.Context) {
	if !h.available(c) {
		return
	}
	var body struct {
		Name          string `json:"name"`
		Code          string `json:"code"`
		ContactUserID string `json:"contact_user_id"`
		ContactName   string `json:"contact_name"`
		ContactMobile string `json:"contact_mobile"`
		AccountCount  int    `json:"account_count"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	item, err := h.Service.CreateTenant(c.Request.Context(), identity.CreateTenantInput{
		Name: body.Name, Code: body.Code, ContactUserID: body.ContactUserID, ContactName: body.ContactName,
		ContactMobile: body.ContactMobile, AccountCount: body.AccountCount,
	})
	h.write(c, item, err)
}

func (h Identity) getTenant(c *gin.Context) {
	id, ok := pathID(c, "tenantId")
	if !ok || !h.available(c) {
		return
	}
	item, err := h.Service.GetTenant(c.Request.Context(), id)
	h.write(c, item, err)
}

func (h Identity) updateTenant(c *gin.Context) {
	id, ok := pathID(c, "tenantId")
	if !ok || !h.available(c) {
		return
	}
	var body struct {
		Name          *string `json:"name"`
		ContactUserID *string `json:"contact_user_id"`
		ContactName   *string `json:"contact_name"`
		ContactMobile *string `json:"contact_mobile"`
		Status        *int    `json:"status"`
		AccountCount  *int    `json:"account_count"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	item, err := h.Service.UpdateTenant(c.Request.Context(), id, identity.UpdateTenantInput{
		Name: body.Name, ContactUserID: body.ContactUserID, ContactName: body.ContactName,
		ContactMobile: body.ContactMobile, Status: body.Status, AccountCount: body.AccountCount,
	})
	h.write(c, item, err)
}

func (h Identity) listOrganizations(c *gin.Context) {
	id, ok := pathID(c, "tenantId")
	if !ok || !h.available(c) {
		return
	}
	items, err := h.Service.ListOrganizationsWithAccess(c.Request.Context(), id, h.principalAccess(c))
	h.write(c, items, err)
}

func (h Identity) createOrganization(c *gin.Context) {
	id, ok := pathID(c, "tenantId")
	if !ok || !h.available(c) {
		return
	}
	var body struct {
		Name         string `json:"name"`
		Code         string `json:"code"`
		ParentID     string `json:"parent_id"`
		Sort         int    `json:"sort"`
		LeaderUserID string `json:"leader_user_id"`
		Phone        string `json:"phone"`
		Email        string `json:"email"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	item, err := h.Service.CreateOrganization(c.Request.Context(), id, identity.CreateOrganizationInput{
		Name: body.Name, Code: body.Code, ParentID: body.ParentID, Sort: body.Sort,
		LeaderUserID: body.LeaderUserID, Phone: body.Phone, Email: body.Email,
	})
	h.write(c, item, err)
}

func (h Identity) updateOrganization(c *gin.Context) {
	tenantID, ok := pathID(c, "tenantId")
	if !ok {
		return
	}
	organizationID, ok := pathID(c, "organizationId")
	if !ok || !h.available(c) {
		return
	}
	var body struct {
		Name         *string         `json:"name"`
		Code         *string         `json:"code"`
		ParentID     json.RawMessage `json:"parent_id"`
		Sort         *int            `json:"sort"`
		LeaderUserID *string         `json:"leader_user_id"`
		Phone        *string         `json:"phone"`
		Email        *string         `json:"email"`
		Status       *int            `json:"status"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	parentSet, parentID, valid := decodeParent(body.ParentID)
	if !valid {
		response.InvalidParams(c, []response.FieldError{{Field: "parent_id", Message: "parent_id不合法"}})
		return
	}
	item, err := h.Service.UpdateOrganization(c.Request.Context(), tenantID, organizationID, identity.UpdateOrganizationInput{
		Name: body.Name, Code: body.Code, ParentSet: parentSet, ParentID: parentID, Sort: body.Sort,
		LeaderUserID: body.LeaderUserID, Phone: body.Phone, Email: body.Email, Status: body.Status,
	})
	h.write(c, item, err)
}

func (h Identity) deleteOrganization(c *gin.Context) {
	tenantID, ok := pathID(c, "tenantId")
	if !ok {
		return
	}
	organizationID, ok := pathID(c, "organizationId")
	if !ok || !h.available(c) {
		return
	}
	h.write(c, nil, h.Service.DeleteOrganization(c.Request.Context(), tenantID, organizationID))
}

func (h Identity) listPosts(c *gin.Context) {
	id, ok := pathID(c, "tenantId")
	if !ok || !h.available(c) {
		return
	}
	query, ok := listQuery(c)
	if !ok {
		return
	}
	page, err := h.Service.ListPosts(c.Request.Context(), id, query)
	h.write(c, page, err)
}

func (h Identity) createPost(c *gin.Context) {
	id, ok := pathID(c, "tenantId")
	if !ok || !h.available(c) {
		return
	}
	var body struct {
		Code   string `json:"code"`
		Name   string `json:"name"`
		Sort   int    `json:"sort"`
		Remark string `json:"remark"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	item, err := h.Service.CreatePost(c.Request.Context(), id, identity.CreatePostInput{
		Code: body.Code, Name: body.Name, Sort: body.Sort, Remark: body.Remark,
	})
	h.write(c, item, err)
}

func (h Identity) getPost(c *gin.Context) {
	tenantID, ok := pathID(c, "tenantId")
	if !ok {
		return
	}
	postID, ok := pathID(c, "postId")
	if !ok || !h.available(c) {
		return
	}
	item, err := h.Service.GetPost(c.Request.Context(), tenantID, postID)
	h.write(c, item, err)
}

func (h Identity) updatePost(c *gin.Context) {
	tenantID, ok := pathID(c, "tenantId")
	if !ok {
		return
	}
	postID, ok := pathID(c, "postId")
	if !ok || !h.available(c) {
		return
	}
	var body struct {
		Code   *string `json:"code"`
		Name   *string `json:"name"`
		Sort   *int    `json:"sort"`
		Status *int    `json:"status"`
		Remark *string `json:"remark"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	item, err := h.Service.UpdatePost(c.Request.Context(), tenantID, postID, identity.UpdatePostInput{
		Code: body.Code, Name: body.Name, Sort: body.Sort, Status: body.Status, Remark: body.Remark,
	})
	h.write(c, item, err)
}

func (h Identity) deletePost(c *gin.Context) {
	tenantID, ok := pathID(c, "tenantId")
	if !ok {
		return
	}
	postID, ok := pathID(c, "postId")
	if !ok || !h.available(c) {
		return
	}
	h.write(c, nil, h.Service.DeletePost(c.Request.Context(), tenantID, postID))
}

func (h Identity) listRoles(c *gin.Context) {
	id, ok := pathID(c, "tenantId")
	if !ok || !h.available(c) {
		return
	}
	query, ok := listQuery(c)
	if !ok {
		return
	}
	page, err := h.Service.ListRoles(c.Request.Context(), id, query)
	h.write(c, page, err)
}

func (h Identity) createRole(c *gin.Context) {
	id, ok := pathID(c, "tenantId")
	if !ok || !h.available(c) {
		return
	}
	var body struct {
		Name      string `json:"name"`
		Code      string `json:"code"`
		Sort      int    `json:"sort"`
		Type      int    `json:"type"`
		DataScope int    `json:"data_scope"`
		Remark    string `json:"remark"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	item, err := h.Service.CreateRole(c.Request.Context(), id, identity.CreateRoleInput{
		Name: body.Name, Code: body.Code, Sort: body.Sort, Type: body.Type,
		DataScope: body.DataScope, Remark: body.Remark,
	})
	h.write(c, item, err)
}

func (h Identity) getRole(c *gin.Context) {
	tenantID, ok := pathID(c, "tenantId")
	if !ok {
		return
	}
	roleID, ok := pathID(c, "roleId")
	if !ok || !h.available(c) {
		return
	}
	item, err := h.Service.GetRole(c.Request.Context(), tenantID, roleID)
	h.write(c, item, err)
}

func (h Identity) updateRole(c *gin.Context) {
	tenantID, ok := pathID(c, "tenantId")
	if !ok {
		return
	}
	roleID, ok := pathID(c, "roleId")
	if !ok || !h.available(c) {
		return
	}
	var body struct {
		Name      *string `json:"name"`
		Code      *string `json:"code"`
		Sort      *int    `json:"sort"`
		Status    *int    `json:"status"`
		Type      *int    `json:"type"`
		DataScope *int    `json:"data_scope"`
		Remark    *string `json:"remark"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	previous, prevErr := h.Service.GetRole(c.Request.Context(), tenantID, roleID)
	item, err := h.Service.UpdateRole(c.Request.Context(), tenantID, roleID, identity.UpdateRoleInput{
		Name: body.Name, Code: body.Code, Sort: body.Sort, Status: body.Status, Type: body.Type,
		DataScope: body.DataScope, Remark: body.Remark,
	})
	if err == nil && prevErr == nil && body.DataScope != nil && previous.DataScope != item.DataScope {
		h.recordPermission(c, security.PermissionLog{
			TenantID: tenantID, RoleID: roleID, RoleName: item.Name, ChangeType: security.ChangeDataScope,
			Summary:    fmt.Sprintf("数据范围由%s调整为%s", dataScopeText(previous.DataScope), dataScopeText(item.DataScope)),
			BeforeText: dataScopeText(previous.DataScope), AfterText: dataScopeText(item.DataScope),
		})
	}
	h.write(c, item, err)
}

func (h Identity) deleteRole(c *gin.Context) {
	tenantID, ok := pathID(c, "tenantId")
	if !ok {
		return
	}
	roleID, ok := pathID(c, "roleId")
	if !ok || !h.available(c) {
		return
	}
	h.write(c, nil, h.Service.DeleteRole(c.Request.Context(), tenantID, roleID))
}

func (h Identity) listRoleMenus(c *gin.Context) {
	tenantID, ok := pathID(c, "tenantId")
	if !ok {
		return
	}
	roleID, ok := pathID(c, "roleId")
	if !ok || !h.available(c) {
		return
	}
	items, err := h.Service.ListRoleMenus(c.Request.Context(), tenantID, roleID)
	h.write(c, items, err)
}

func (h Identity) listRoleDepts(c *gin.Context) {
	tenantID, ok := pathID(c, "tenantId")
	if !ok {
		return
	}
	roleID, ok := pathID(c, "roleId")
	if !ok || !h.available(c) {
		return
	}
	items, err := h.Service.ListRoleDepts(c.Request.Context(), tenantID, roleID)
	h.write(c, items, err)
}

func (h Identity) replaceRoleDepts(c *gin.Context) {
	tenantID, ok := pathID(c, "tenantId")
	if !ok {
		return
	}
	roleID, ok := pathID(c, "roleId")
	if !ok || !h.available(c) {
		return
	}
	var body struct {
		DeptIDs []string `json:"dept_ids"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	before, _ := h.Service.ListRoleDepts(c.Request.Context(), tenantID, roleID)
	role, _ := h.Service.GetRole(c.Request.Context(), tenantID, roleID)
	items, err := h.Service.ReplaceRoleDepts(c.Request.Context(), tenantID, roleID, body.DeptIDs)
	if err == nil {
		h.recordPermission(c, security.PermissionLog{
			TenantID: tenantID, RoleID: roleID, RoleName: role.Name, ChangeType: security.ChangeDepts,
			Summary:    fmt.Sprintf("数据权限部门由 %d 个调整为 %d 个", len(before), len(items)),
			BeforeText: strings.Join(before, ","), AfterText: strings.Join(items, ","),
		})
	}
	h.write(c, items, err)
}

func (h Identity) replaceRoleMenus(c *gin.Context) {
	tenantID, ok := pathID(c, "tenantId")
	if !ok {
		return
	}
	roleID, ok := pathID(c, "roleId")
	if !ok || !h.available(c) {
		return
	}
	var body struct {
		MenuIDs []string `json:"menu_ids"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	before, _ := h.Service.ListRoleMenus(c.Request.Context(), tenantID, roleID)
	role, _ := h.Service.GetRole(c.Request.Context(), tenantID, roleID)
	items, err := h.Service.ReplaceRoleMenus(c.Request.Context(), tenantID, roleID, body.MenuIDs)
	if err == nil {
		h.recordPermission(c, security.PermissionLog{
			TenantID: tenantID, RoleID: roleID, RoleName: role.Name, ChangeType: security.ChangeMenus,
			Summary:    fmt.Sprintf("菜单权限由 %d 项调整为 %d 项", len(before), len(items)),
			BeforeText: strings.Join(before, ","), AfterText: strings.Join(items, ","),
		})
	}
	h.write(c, items, err)
}

func (h Identity) listRoleUsers(c *gin.Context) {
	tenantID, ok := pathID(c, "tenantId")
	if !ok {
		return
	}
	roleID, ok := pathID(c, "roleId")
	if !ok || !h.available(c) {
		return
	}
	items, err := h.Service.ListRoleUsers(c.Request.Context(), tenantID, roleID)
	h.write(c, items, err)
}

func (h Identity) replaceRoleUsers(c *gin.Context) {
	tenantID, ok := pathID(c, "tenantId")
	if !ok {
		return
	}
	roleID, ok := pathID(c, "roleId")
	if !ok || !h.available(c) {
		return
	}
	var body struct {
		UserIDs []string `json:"user_ids"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	before, _ := h.Service.ListRoleUsers(c.Request.Context(), tenantID, roleID)
	role, _ := h.Service.GetRole(c.Request.Context(), tenantID, roleID)
	items, err := h.Service.ReplaceRoleUsers(c.Request.Context(), tenantID, roleID, body.UserIDs)
	if err == nil {
		h.recordPermission(c, security.PermissionLog{
			TenantID: tenantID, RoleID: roleID, RoleName: role.Name, ChangeType: security.ChangeUsers,
			Summary:    fmt.Sprintf("角色用户由 %d 人调整为 %d 人", len(before), len(items)),
			BeforeText: roleUserNames(before), AfterText: roleUserNames(items),
		})
	}
	h.write(c, items, err)
}

func (h Identity) listMenus(c *gin.Context) {
	if !h.available(c) {
		return
	}
	items, err := h.Service.ListMenus(c.Request.Context())
	h.write(c, items, err)
}

func (h Identity) createMenu(c *gin.Context) {
	if !h.available(c) {
		return
	}
	var body struct {
		Name          string `json:"name"`
		Permission    string `json:"permission"`
		Type          int    `json:"type"`
		Sort          int    `json:"sort"`
		ParentID      string `json:"parent_id"`
		Path          string `json:"path"`
		Icon          string `json:"icon"`
		Component     string `json:"component"`
		ComponentName string `json:"component_name"`
		Visible       *bool  `json:"visible"`
		KeepAlive     *bool  `json:"keep_alive"`
		AlwaysShow    *bool  `json:"always_show"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	item, err := h.Service.CreateMenu(c.Request.Context(), identity.CreateMenuInput{
		Name: body.Name, Permission: body.Permission, Type: body.Type, Sort: body.Sort, ParentID: body.ParentID,
		Path: body.Path, Icon: body.Icon, Component: body.Component, ComponentName: body.ComponentName,
		Visible: body.Visible, KeepAlive: body.KeepAlive, AlwaysShow: body.AlwaysShow,
	})
	h.write(c, item, err)
}

func (h Identity) getMenu(c *gin.Context) {
	id, ok := pathID(c, "menuId")
	if !ok || !h.available(c) {
		return
	}
	item, err := h.Service.GetMenu(c.Request.Context(), id)
	h.write(c, item, err)
}

func (h Identity) updateMenu(c *gin.Context) {
	id, ok := pathID(c, "menuId")
	if !ok || !h.available(c) {
		return
	}
	var body struct {
		Name          *string         `json:"name"`
		Permission    *string         `json:"permission"`
		Type          *int            `json:"type"`
		Sort          *int            `json:"sort"`
		ParentID      json.RawMessage `json:"parent_id"`
		Path          *string         `json:"path"`
		Icon          *string         `json:"icon"`
		Component     *string         `json:"component"`
		ComponentName *string         `json:"component_name"`
		Status        *int            `json:"status"`
		Visible       *bool           `json:"visible"`
		KeepAlive     *bool           `json:"keep_alive"`
		AlwaysShow    *bool           `json:"always_show"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	parentSet, parentID, valid := decodeParent(body.ParentID)
	if !valid {
		response.InvalidParams(c, []response.FieldError{{Field: "parent_id", Message: "parent_id不合法"}})
		return
	}
	item, err := h.Service.UpdateMenu(c.Request.Context(), id, identity.UpdateMenuInput{
		Name: body.Name, Permission: body.Permission, Type: body.Type, Sort: body.Sort,
		ParentSet: parentSet, ParentID: parentID, Path: body.Path, Icon: body.Icon,
		Component: body.Component, ComponentName: body.ComponentName, Status: body.Status,
		Visible: body.Visible, KeepAlive: body.KeepAlive, AlwaysShow: body.AlwaysShow,
	})
	h.write(c, item, err)
}

func (h Identity) deleteMenu(c *gin.Context) {
	id, ok := pathID(c, "menuId")
	if !ok || !h.available(c) {
		return
	}
	h.write(c, nil, h.Service.DeleteMenu(c.Request.Context(), id))
}

func (h Identity) listUsers(c *gin.Context) {
	if !h.available(c) {
		return
	}
	query, ok := listQuery(c)
	if !ok {
		return
	}
	page, err := h.Service.ListUsersWithAccess(c.Request.Context(), h.principalTenant(c), query, h.principalAccess(c))
	h.write(c, page, err)
}

func (h Identity) createUser(c *gin.Context) {
	if !h.available(c) {
		return
	}
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Nickname string `json:"nickname"`
		Remark   string `json:"remark"`
		Email    string `json:"email"`
		Mobile   string `json:"mobile"`
		Sex      int    `json:"sex"`
		Avatar   string `json:"avatar"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	if h.rejectWeakPassword(c, body.Password) {
		return
	}
	item, err := h.Service.CreateUser(c.Request.Context(), identity.CreateUserInput{
		Username: body.Username, Password: body.Password, Nickname: body.Nickname, Remark: body.Remark,
		Email: body.Email, Mobile: body.Mobile, Sex: body.Sex, Avatar: body.Avatar,
	})
	h.write(c, item, err)
}

func (h Identity) getUser(c *gin.Context) {
	id, ok := pathID(c, "userId")
	if !ok || !h.available(c) {
		return
	}
	item, err := h.Service.GetUser(c.Request.Context(), id)
	h.write(c, item, err)
}

func (h Identity) updateUser(c *gin.Context) {
	id, ok := pathID(c, "userId")
	if !ok || !h.available(c) {
		return
	}
	var body struct {
		Nickname *string `json:"nickname"`
		Password *string `json:"password"`
		Remark   *string `json:"remark"`
		Email    *string `json:"email"`
		Mobile   *string `json:"mobile"`
		Sex      *int    `json:"sex"`
		Avatar   *string `json:"avatar"`
		Status   *int    `json:"status"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	if body.Password != nil && h.rejectWeakPassword(c, *body.Password) {
		return
	}
	item, err := h.Service.UpdateUser(c.Request.Context(), id, identity.UpdateUserInput{
		Nickname: body.Nickname, Password: body.Password, Remark: body.Remark, Email: body.Email, Mobile: body.Mobile,
		Sex: body.Sex, Avatar: body.Avatar, Status: body.Status,
	})
	h.write(c, item, err)
}

func (h Identity) listMembers(c *gin.Context) {
	id, ok := pathID(c, "tenantId")
	if !ok || !h.available(c) {
		return
	}
	query, ok := listQuery(c)
	if !ok {
		return
	}
	page, err := h.Service.ListMembersWithAccess(c.Request.Context(), id, query, h.principalAccess(c))
	h.write(c, page, err)
}

func (h Identity) createMember(c *gin.Context) {
	id, ok := pathID(c, "tenantId")
	if !ok || !h.available(c) {
		return
	}
	var body struct {
		UserID                string    `json:"user_id"`
		Status                int       `json:"status"`
		OrganizationIDs       *[]string `json:"organization_ids"`
		PrimaryOrganizationID string    `json:"primary_organization_id"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	assignments, ok := assignmentInput(c, body.OrganizationIDs, body.PrimaryOrganizationID)
	if !ok {
		return
	}
	item, err := h.Service.CreateMember(c.Request.Context(), id, identity.CreateMemberInput{
		UserID: body.UserID, Status: body.Status, AssignmentInput: assignments,
	})
	h.write(c, item, err)
}

func (h Identity) replaceMember(c *gin.Context) {
	tenantID, ok := pathID(c, "tenantId")
	if !ok {
		return
	}
	userID, ok := pathID(c, "userId")
	if !ok || !h.available(c) {
		return
	}
	var body struct {
		OrganizationIDs       *[]string `json:"organization_ids"`
		PrimaryOrganizationID string    `json:"primary_organization_id"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	assignments, ok := assignmentInput(c, body.OrganizationIDs, body.PrimaryOrganizationID)
	if !ok {
		return
	}
	item, err := h.Service.ReplaceMemberAssignments(c.Request.Context(), tenantID, userID, assignments)
	h.write(c, item, err)
}

func (h Identity) removeMember(c *gin.Context) {
	tenantID, ok := pathID(c, "tenantId")
	if !ok {
		return
	}
	userID, ok := pathID(c, "userId")
	if !ok || !h.available(c) {
		return
	}
	h.write(c, nil, h.Service.RemoveMember(c.Request.Context(), tenantID, userID))
}

func (h Identity) available(c *gin.Context) bool {
	if h.Service == nil {
		response.Fail(c, http.StatusServiceUnavailable, response.CodeNotReady, "服务暂不可用", nil)
		return false
	}
	if h.Database == nil {
		return true
	}
	timeout := h.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()
	if err := h.Database.Ping(ctx); err != nil {
		if h.Logger != nil {
			h.Logger.Warn("identity database unavailable",
				slog.String("component", "app"),
				slog.String("error", err.Error()),
			)
		}
		response.Fail(c, http.StatusServiceUnavailable, response.CodeNotReady, "服务暂不可用", nil)
		return false
	}
	return true
}

func (h Identity) write(c *gin.Context, data any, err error) {
	if err == nil {
		response.OK(c, data)
		return
	}
	var domain *identity.Error
	if errors.As(err, &domain) {
		switch {
		case errors.Is(err, identity.ErrUnauthorized):
			response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, domain.Message, nil)
		case errors.Is(err, identity.ErrForbidden):
			response.Fail(c, http.StatusForbidden, response.CodeForbidden, domain.Message, nil)
		case errors.Is(err, identity.ErrInvalid):
			response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, domain.Message, nil)
		case errors.Is(err, identity.ErrNotFound):
			response.Fail(c, http.StatusNotFound, response.CodeNotFound, domain.Message, nil)
		case errors.Is(err, identity.ErrConflict):
			response.Fail(c, http.StatusConflict, response.CodeConflict, domain.Message, nil)
		default:
			response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, domain.Message, nil)
		}
		return
	}
	if h.Logger != nil {
		h.Logger.Error("identity request failed",
			slog.String("component", "app"),
			slog.String("error", err.Error()),
		)
	}
	response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误", nil)
}

func (h Identity) rejectWeakPassword(c *gin.Context, password string) bool {
	if h.Security == nil {
		return false
	}
	if err := h.Security.CheckPassword(c.Request.Context(), password); err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, err.Error(), nil)
		return true
	}
	return false
}

func (h Identity) recordPermission(c *gin.Context, item security.PermissionLog) {
	if h.Security == nil {
		return
	}
	if principal, ok := auth.GetPrincipal(c); ok {
		item.OperatorID = principal.UserID
		item.OperatorName = principal.Username
	}
	if err := h.Security.RecordPermission(c.Request.Context(), item); err != nil && h.Logger != nil {
		h.Logger.Warn("permission log failed", slog.String("component", "app"), slog.String("error", err.Error()))
	}
}

func roleUserNames(items []identity.RoleUserView) string {
	names := make([]string, 0, len(items))
	for _, item := range items {
		names = append(names, item.Username)
	}
	return strings.Join(names, ",")
}

func dataScopeText(scope int) string {
	switch scope {
	case identity.DataScopeAll:
		return "全部"
	case identity.DataScopeCustom:
		return "自定义"
	case identity.DataScopeDept:
		return "本部门"
	case identity.DataScopeDeptAndChild:
		return "本部门及以下"
	case identity.DataScopeSelf:
		return "仅本人"
	default:
		return "未知"
	}
}

func pathID(c *gin.Context, name string) (string, bool) {
	value := c.Param(name)
	if !uuidPattern.MatchString(value) {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "请求参数不合法", nil)
		return "", false
	}
	return value, true
}

func listQuery(c *gin.Context) (identity.ListQuery, bool) {
	page, err := optionalInt(c.Query("page"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "分页参数不合法", nil)
		return identity.ListQuery{}, false
	}
	pageSize, err := optionalInt(c.Query("page_size"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "分页参数不合法", nil)
		return identity.ListQuery{}, false
	}
	keyword := strings.TrimSpace(c.Query("keyword"))
	if len(keyword) > 64 {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "请求参数不合法", nil)
		return identity.ListQuery{}, false
	}
	return identity.ListQuery{Page: page, PageSize: pageSize, Keyword: keyword}, true
}

func optionalInt(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	return strconv.Atoi(raw)
}

func assignmentInput(c *gin.Context, ids *[]string, primary string) (identity.AssignmentInput, bool) {
	if ids == nil {
		response.InvalidParams(c, []response.FieldError{{Field: "organization_ids", Message: "organization_ids为必填项"}})
		return identity.AssignmentInput{}, false
	}
	return identity.AssignmentInput{OrganizationIDs: *ids, PrimaryOrganizationID: primary}, true
}

func decodeParent(raw json.RawMessage) (bool, *string, bool) {
	if len(raw) == 0 {
		return false, nil, true
	}
	if string(raw) == "null" {
		return true, nil, true
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return false, nil, false
	}
	return true, &value, true
}
