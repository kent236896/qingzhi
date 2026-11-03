package handler

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"qingzhi/enterprise-api/internal/auth"
	"qingzhi/enterprise-api/internal/catalog"
	"qingzhi/enterprise-api/internal/identity"
	"qingzhi/enterprise-api/internal/response"
	"qingzhi/enterprise-api/internal/scoped"
	"qingzhi/enterprise-api/internal/weknora"
)

// Catalog serves agents, MCP toolbox, and artifacts.
// Agent/MCP definition writes go through weknora admin plane only; never proxy
// DELETE agents or PUT credentials with External Token (see docs/weknora-call-planes.md).
type Catalog struct {
	Logger   *slog.Logger
	Service  *catalog.Service
	Identity *identity.Service
	Memory   *weknora.MemoryService
}

func (h Catalog) Register(group *gin.RouterGroup, enforce bool) {
	perm := func(permission string) gin.HandlerFunc {
		if !enforce {
			return func(c *gin.Context) { c.Next() }
		}
		return auth.RequirePermission(permission)
	}
	agentQuery := perm("agent:query")
	toolboxQuery := perm("toolbox:query")
	artifactQuery := perm("artifact:query")

	group.GET("/catalog/options", h.options)

	group.GET("/agents", agentQuery, h.listAgents)
	group.POST("/agents", agentQuery, h.createAgent)
	group.PUT("/agents/:id", agentQuery, h.updateAgent)
	group.POST("/agents/:id/copy", agentQuery, h.copyAgent)
	group.DELETE("/agents/:id", agentQuery, h.deleteAgent)

	group.GET("/toolbox/mcp", toolboxQuery, h.listMCP)
	group.POST("/toolbox/mcp", toolboxQuery, h.createMCP)
	group.PUT("/toolbox/mcp/:id", toolboxQuery, h.updateMCP)
	group.DELETE("/toolbox/mcp/:id", toolboxQuery, h.deleteMCP)
	group.POST("/toolbox/mcp/:id/test", toolboxQuery, h.testMCP)
	group.PUT("/toolbox/mcp/:id/credentials", toolboxQuery, h.putMCPCredentials)
	group.DELETE("/toolbox/mcp/:id/credentials/:field", toolboxQuery, h.deleteMCPCredential)
	group.GET("/toolbox/mcp/:id/metadata", toolboxQuery, h.getMCPMetadata)
	group.POST("/toolbox/mcp/:id/metadata/refresh", toolboxQuery, h.refreshMCPMetadata)
	group.GET("/toolbox/mcp/:id/oauth/status", toolboxQuery, h.mcpOAuthStatus)
	group.POST("/toolbox/mcp/:id/oauth/authorize-url", toolboxQuery, h.mcpOAuthAuthorizeURL)
	group.DELETE("/toolbox/mcp/:id/oauth/token", toolboxQuery, h.mcpOAuthRevoke)

	group.GET("/artifacts", artifactQuery, h.listArtifactsUser)
	group.DELETE("/artifacts", artifactQuery, h.deleteArtifactUser)

	group.GET("/admin/artifacts", artifactQuery, h.listArtifactsAdmin)
	group.DELETE("/admin/artifacts", perm("artifact:manage"), h.deleteArtifactAdmin)
}

func (h Catalog) options(c *gin.Context) {
	principal, ok := auth.GetPrincipal(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录或令牌无效", nil)
		return
	}
	if h.Identity == nil {
		response.Fail(c, http.StatusServiceUnavailable, response.CodeNotReady, "服务暂不可用", nil)
		return
	}
	manageAgents := principal.HasPermission("agent:manage")
	manageToolbox := principal.HasPermission("toolbox:manage")
	canManage := manageAgents || manageToolbox
	canQueryAll := principal.HasPermission("system:tenant:query")

	tenantIDs := []string{}
	ids, err := h.Identity.ListMembershipTenantIDs(c.Request.Context(), principal.UserID)
	if err != nil {
		h.write(c, nil, err)
		return
	}
	tenantIDs = ids
	if principal.TenantID != "" {
		found := false
		for _, id := range tenantIDs {
			if id == principal.TenantID {
				found = true
				break
			}
		}
		if !found {
			tenantIDs = append(tenantIDs, principal.TenantID)
		}
	}

	type choice struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		TenantID string `json:"tenant_id,omitempty"`
	}
	tenants := make([]choice, 0)
	if canQueryAll {
		page, listErr := h.Identity.ListTenants(c.Request.Context(), identity.ListQuery{Page: 1, PageSize: 100})
		if listErr != nil {
			h.write(c, nil, listErr)
			return
		}
		for _, item := range page.Items {
			tenants = append(tenants, choice{ID: item.ID, Name: item.Name})
		}
	} else {
		for _, id := range tenantIDs {
			item, getErr := h.Identity.GetTenant(c.Request.Context(), id)
			if getErr == nil {
				tenants = append(tenants, choice{ID: item.ID, Name: item.Name})
			}
		}
	}
	orgs := make([]choice, 0)
	roles := make([]choice, 0)
	users := make([]choice, 0)
	if canManage {
		for _, tenant := range tenants {
			list, listErr := h.Identity.ListOrganizationRecords(c.Request.Context(), tenant.ID)
			if listErr == nil {
				for _, org := range list {
					orgs = append(orgs, choice{ID: org.ID, Name: org.Name, TenantID: org.TenantID})
				}
			}
			rolePage, roleErr := h.Identity.ListRoles(c.Request.Context(), tenant.ID, identity.ListQuery{Page: 1, PageSize: 100})
			if roleErr == nil {
				for _, role := range rolePage.Items {
					roles = append(roles, choice{ID: role.ID, Name: role.Name, TenantID: role.TenantID})
				}
			}
		}
	}
	userPage, userErr := h.Identity.ListUsers(c.Request.Context(), identity.ListQuery{Page: 1, PageSize: 100})
	if userErr == nil {
		for _, user := range userPage.Items {
			users = append(users, choice{ID: user.ID, Name: user.Nickname + "（" + user.Username + "）"})
		}
	}
	response.OK(c, map[string]any{
		"tenants": tenants, "organizations": orgs, "roles": roles, "users": users,
		"can_manage_agents": manageAgents, "can_manage_toolbox": manageToolbox,
	})
}

func (h Catalog) listAgents(c *gin.Context) {
	svc, actor, ok := h.ready(c, "agent:manage")
	if !ok {
		return
	}
	items, err := svc.ListAgents(c.Request.Context(), actor)
	h.write(c, items, err)
}

func (h Catalog) createAgent(c *gin.Context) {
	svc, actor, ok := h.ready(c, "agent:manage")
	if !ok {
		return
	}
	var body struct {
		Name         string         `json:"name"`
		Description  string         `json:"description"`
		ScopeType    string         `json:"scope_type"`
		ScopeID      string         `json:"scope_id"`
		TenantIDs    []string       `json:"tenant_ids"`
		ShareUserIDs []string       `json:"share_user_ids"`
		Config       map[string]any `json:"config"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	if body.ScopeType != scoped.ScopePersonal && !actor.CanManage {
		response.Fail(c, http.StatusForbidden, response.CodeForbidden, "租户/组织/角色智能体需要管理权限", nil)
		return
	}
	item, err := svc.CreateAgent(c.Request.Context(), actor, catalog.CreateInput{
		Name: body.Name, Description: body.Description, ScopeType: body.ScopeType, ScopeID: body.ScopeID,
		TenantIDs: body.TenantIDs, ShareUserIDs: body.ShareUserIDs, Config: body.Config,
	})
	h.write(c, item, err)
}

func (h Catalog) updateAgent(c *gin.Context) {
	svc, actor, ok := h.ready(c, "agent:manage")
	if !ok {
		return
	}
	var body struct {
		Name         string         `json:"name"`
		Description  string         `json:"description"`
		ShareUserIDs []string       `json:"share_user_ids"`
		Config       map[string]any `json:"config"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	item, err := svc.UpdateAgent(c.Request.Context(), actor, c.Param("id"), catalog.UpdateInput{
		Name: body.Name, Description: body.Description, ShareUserIDs: body.ShareUserIDs, Config: body.Config,
	})
	h.write(c, item, err)
}

func (h Catalog) copyAgent(c *gin.Context) {
	svc, actor, ok := h.ready(c, "agent:manage")
	if !ok {
		return
	}
	var body struct {
		Name         string   `json:"name"`
		ScopeType    string   `json:"scope_type"`
		ScopeID      string   `json:"scope_id"`
		TenantIDs    []string `json:"tenant_ids"`
		ShareUserIDs []string `json:"share_user_ids"`
	}
	// Empty body means default personal copy.
	if err := c.ShouldBindJSON(&body); err != nil && !errors.Is(err, io.EOF) {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "请求体不是合法的 JSON", nil)
		return
	}
	if body.ScopeType != "" && body.ScopeType != scoped.ScopePersonal && !actor.CanManage {
		response.Fail(c, http.StatusForbidden, response.CodeForbidden, "租户/组织/角色智能体需要管理权限", nil)
		return
	}
	item, err := svc.CopyAgent(c.Request.Context(), actor, c.Param("id"), catalog.CopyInput{
		Name: body.Name, ScopeType: body.ScopeType, ScopeID: body.ScopeID,
		TenantIDs: body.TenantIDs, ShareUserIDs: body.ShareUserIDs,
	})
	h.write(c, item, err)
}

func (h Catalog) deleteAgent(c *gin.Context) {
	svc, actor, ok := h.ready(c, "agent:manage")
	if !ok {
		return
	}
	err := svc.DeleteAgent(c.Request.Context(), actor, c.Param("id"))
	h.write(c, map[string]any{"deleted": err == nil}, err)
}

func (h Catalog) listMCP(c *gin.Context) {
	svc, actor, ok := h.ready(c, "toolbox:manage")
	if !ok {
		return
	}
	items, err := svc.ListMCP(c.Request.Context(), actor)
	h.write(c, items, err)
}

func (h Catalog) createMCP(c *gin.Context) {
	svc, actor, ok := h.ready(c, "toolbox:manage")
	if !ok {
		return
	}
	var body struct {
		Name          string   `json:"name"`
		Description   string   `json:"description"`
		ScopeType     string   `json:"scope_type"`
		ScopeID       string   `json:"scope_id"`
		TenantIDs     []string `json:"tenant_ids"`
		ShareUserIDs  []string `json:"share_user_ids"`
		TransportType string   `json:"transport_type"`
		URL           string   `json:"url"`
		AuthMode      string   `json:"auth_mode"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	if body.ScopeType != scoped.ScopePersonal && !actor.CanManage {
		response.Fail(c, http.StatusForbidden, response.CodeForbidden, "租户/组织/角色 MCP 需要管理权限", nil)
		return
	}
	item, err := svc.CreateMCP(c.Request.Context(), actor, catalog.CreateInput{
		Name: body.Name, Description: body.Description, ScopeType: body.ScopeType, ScopeID: body.ScopeID,
		TenantIDs: body.TenantIDs, ShareUserIDs: body.ShareUserIDs, TransportType: body.TransportType,
		URL: body.URL, AuthMode: body.AuthMode,
	})
	h.write(c, item, err)
}

func (h Catalog) updateMCP(c *gin.Context) {
	svc, actor, ok := h.ready(c, "toolbox:manage")
	if !ok {
		return
	}
	var body struct {
		Name          string   `json:"name"`
		Description   string   `json:"description"`
		ShareUserIDs  []string `json:"share_user_ids"`
		TransportType string   `json:"transport_type"`
		URL           string   `json:"url"`
		AuthMode      string   `json:"auth_mode"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	item, err := svc.UpdateMCP(c.Request.Context(), actor, c.Param("id"), catalog.UpdateInput{
		Name: body.Name, Description: body.Description, ShareUserIDs: body.ShareUserIDs,
		TransportType: body.TransportType, URL: body.URL, AuthMode: body.AuthMode,
	})
	h.write(c, item, err)
}

func (h Catalog) deleteMCP(c *gin.Context) {
	svc, actor, ok := h.ready(c, "toolbox:manage")
	if !ok {
		return
	}
	err := svc.DeleteMCP(c.Request.Context(), actor, c.Param("id"))
	h.write(c, map[string]any{"deleted": err == nil}, err)
}

func (h Catalog) testMCP(c *gin.Context) {
	svc, actor, ok := h.ready(c, "toolbox:manage")
	if !ok {
		return
	}
	data, err := svc.TestMCP(c.Request.Context(), actor, c.Param("id"))
	h.write(c, data, err)
}

func (h Catalog) putMCPCredentials(c *gin.Context) {
	svc, actor, ok := h.ready(c, "toolbox:manage")
	if !ok {
		return
	}
	var body map[string]string
	if !response.BindJSON(c, &body) {
		return
	}
	data, err := svc.PutMCPCredentials(c.Request.Context(), actor, c.Param("id"), body)
	h.write(c, data, err)
}

func (h Catalog) deleteMCPCredential(c *gin.Context) {
	svc, actor, ok := h.ready(c, "toolbox:manage")
	if !ok {
		return
	}
	err := svc.DeleteMCPCredential(c.Request.Context(), actor, c.Param("id"), c.Param("field"))
	h.write(c, map[string]any{"deleted": err == nil}, err)
}

func (h Catalog) getMCPMetadata(c *gin.Context) {
	svc, actor, ok := h.ready(c, "toolbox:manage")
	if !ok {
		return
	}
	data, err := svc.GetMCPMetadata(c.Request.Context(), actor, c.Param("id"))
	h.write(c, data, err)
}

func (h Catalog) refreshMCPMetadata(c *gin.Context) {
	svc, actor, ok := h.ready(c, "toolbox:manage")
	if !ok {
		return
	}
	data, err := svc.RefreshMCPMetadata(c.Request.Context(), actor, c.Param("id"))
	h.write(c, data, err)
}

func (h Catalog) mcpOAuthStatus(c *gin.Context) {
	svc, actor, principal, ok := h.readyUser(c, "toolbox:manage")
	if !ok {
		return
	}
	ctx, err := h.withUserToken(c.Request.Context(), principal)
	if err != nil {
		h.write(c, nil, err)
		return
	}
	data, err := svc.GetMCPOAuthStatus(ctx, actor, c.Param("id"), c.Query("authorization_attempt"))
	h.write(c, data, err)
}

func (h Catalog) mcpOAuthAuthorizeURL(c *gin.Context) {
	svc, actor, principal, ok := h.readyUser(c, "toolbox:manage")
	if !ok {
		return
	}
	var body struct {
		FrontendRedirect string `json:"frontend_redirect"`
	}
	if err := c.ShouldBindJSON(&body); err != nil && !errors.Is(err, io.EOF) {
		response.Fail(c, http.StatusBadRequest, response.CodeBadRequest, "请求体不是合法的 JSON", nil)
		return
	}
	frontendRedirect := strings.TrimSpace(body.FrontendRedirect)
	if frontendRedirect == "" {
		frontendRedirect = strings.TrimSpace(c.GetHeader("X-Frontend-Redirect"))
	}
	if frontendRedirect == "" {
		// Prefer workbench toolbox; WeKnora will append #mcp_oauth_result=...
		origin := strings.TrimRight(strings.TrimSpace(c.GetHeader("Origin")), "/")
		if origin != "" {
			frontendRedirect = origin + "/toolbox"
		}
	}
	ctx, err := h.withUserToken(c.Request.Context(), principal)
	if err != nil {
		h.write(c, nil, err)
		return
	}
	data, err := svc.StartMCPOAuthAuthorize(ctx, actor, c.Param("id"), frontendRedirect)
	h.write(c, data, err)
}

func (h Catalog) mcpOAuthRevoke(c *gin.Context) {
	svc, actor, principal, ok := h.readyUser(c, "toolbox:manage")
	if !ok {
		return
	}
	ctx, err := h.withUserToken(c.Request.Context(), principal)
	if err != nil {
		h.write(c, nil, err)
		return
	}
	err = svc.RevokeMCPOAuthToken(ctx, actor, c.Param("id"))
	h.write(c, map[string]any{"revoked": err == nil}, err)
}

// readyUser is ready() plus the authenticated principal (for External Token issuance).
func (h Catalog) readyUser(c *gin.Context, managePerm string) (*catalog.Service, catalog.Actor, auth.Principal, bool) {
	svc, actor, ok := h.ready(c, managePerm)
	if !ok {
		return nil, catalog.Actor{}, auth.Principal{}, false
	}
	principal, ok := auth.GetPrincipal(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录或令牌无效", nil)
		return nil, catalog.Actor{}, auth.Principal{}, false
	}
	return svc, actor, principal, true
}

func (h Catalog) listArtifactsUser(c *gin.Context) {
	svc, principal, ok := h.readyPrincipal(c)
	if !ok {
		return
	}
	ctx, err := h.withUserToken(c.Request.Context(), principal)
	if err != nil {
		h.write(c, nil, err)
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	data, err := svc.ListArtifactsWithToken(ctx, c.Query("keyword"), page, pageSize)
	h.write(c, data, err)
}

func (h Catalog) deleteArtifactUser(c *gin.Context) {
	svc, principal, ok := h.readyPrincipal(c)
	if !ok {
		return
	}
	ctx, err := h.withUserToken(c.Request.Context(), principal)
	if err != nil {
		h.write(c, nil, err)
		return
	}
	index, _ := strconv.Atoi(c.Query("index"))
	err = svc.DeleteArtifactUser(ctx, c.Query("session_id"), c.Query("message_id"), index)
	h.write(c, map[string]any{"deleted": err == nil}, err)
}

func (h Catalog) listArtifactsAdmin(c *gin.Context) {
	svc, _, ok := h.ready(c, "artifact:manage")
	if !ok {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	data, err := svc.ListArtifactsAdmin(c.Request.Context(), c.Query("keyword"), page, pageSize)
	h.write(c, data, err)
}

func (h Catalog) deleteArtifactAdmin(c *gin.Context) {
	svc, _, ok := h.ready(c, "artifact:manage")
	if !ok {
		return
	}
	index, _ := strconv.Atoi(c.Query("index"))
	err := svc.DeleteArtifactAdmin(c.Request.Context(), c.Query("session_id"), c.Query("message_id"), index)
	h.write(c, map[string]any{"deleted": err == nil}, err)
}

func (h Catalog) withUserToken(ctx context.Context, principal auth.Principal) (context.Context, error) {
	if h.Memory == nil {
		return ctx, &catalog.Error{Kind: catalog.ErrNotReady, Message: "记忆/身份令牌服务不可用"}
	}
	token, _, _, err := h.Memory.IssueExternalUserToken(ctx, principal)
	if err != nil {
		return ctx, &catalog.Error{Kind: catalog.ErrNotReady, Message: err.Error()}
	}
	return weknora.WithExternalUserToken(ctx, token), nil
}

func (h Catalog) ready(c *gin.Context, managePerm string) (*catalog.Service, catalog.Actor, bool) {
	principal, ok := auth.GetPrincipal(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录或令牌无效", nil)
		return nil, catalog.Actor{}, false
	}
	if h.Service == nil {
		response.Fail(c, http.StatusServiceUnavailable, response.CodeNotReady, "服务暂不可用", nil)
		return nil, catalog.Actor{}, false
	}
	actor := catalog.Actor{
		UserID:    principal.UserID,
		Username:  principal.Username,
		CanManage: principal.HasPermission(managePerm),
	}
	if h.Identity != nil {
		ids, err := h.Identity.ListMembershipTenantIDs(c.Request.Context(), principal.UserID)
		if err != nil {
			h.write(c, nil, err)
			return nil, catalog.Actor{}, false
		}
		actor.TenantIDs = ids
	}
	if principal.TenantID != "" {
		found := false
		for _, id := range actor.TenantIDs {
			if id == principal.TenantID {
				found = true
				break
			}
		}
		if !found {
			actor.TenantIDs = append(actor.TenantIDs, principal.TenantID)
		}
	}
	return h.Service, actor, true
}

func (h Catalog) readyPrincipal(c *gin.Context) (*catalog.Service, auth.Principal, bool) {
	principal, ok := auth.GetPrincipal(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录或令牌无效", nil)
		return nil, auth.Principal{}, false
	}
	if h.Service == nil {
		response.Fail(c, http.StatusServiceUnavailable, response.CodeNotReady, "服务暂不可用", nil)
		return nil, auth.Principal{}, false
	}
	return h.Service, principal, true
}

func (h Catalog) write(c *gin.Context, data any, err error) {
	if err == nil {
		response.OK(c, data)
		return
	}
	var domain *catalog.Error
	if errors.As(err, &domain) {
		switch {
		case errors.Is(domain.Kind, catalog.ErrInvalid):
			response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, domain.Message, nil)
		case errors.Is(domain.Kind, catalog.ErrNotFound):
			response.Fail(c, http.StatusNotFound, response.CodeNotFound, domain.Message, nil)
		case errors.Is(domain.Kind, catalog.ErrConflict):
			response.Fail(c, http.StatusConflict, response.CodeConflict, domain.Message, nil)
		case errors.Is(domain.Kind, catalog.ErrForbidden):
			response.Fail(c, http.StatusForbidden, response.CodeForbidden, domain.Message, nil)
		case errors.Is(domain.Kind, catalog.ErrNotReady):
			response.Fail(c, http.StatusServiceUnavailable, response.CodeNotReady, domain.Message, nil)
		default:
			response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, domain.Message, nil)
		}
		return
	}
	var identityErr *identity.Error
	if errors.As(err, &identityErr) {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, identityErr.Message, nil)
		return
	}
	if h.Logger != nil {
		h.Logger.Error("catalog request failed", slog.String("component", "catalog"), slog.String("error", err.Error()))
	}
	msg := err.Error()
	if strings.Contains(msg, "HTTP") || strings.Contains(msg, "MCP") || strings.Contains(msg, "智能体") || strings.Contains(msg, "产物") {
		response.Fail(c, http.StatusBadGateway, response.CodeInternal, msg, nil)
		return
	}
	response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误", nil)
}
