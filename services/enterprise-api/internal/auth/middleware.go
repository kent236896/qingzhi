package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"qingzhi/enterprise-api/internal/identity"
	"qingzhi/enterprise-api/internal/response"
)

// Guard authenticates requests and enforces permissions.
type Guard struct {
	Tokens  *TokenIssuer
	Service *identity.Service
}

// Authenticate validates Bearer JWT and loads the current session access.
func (g Guard) Authenticate() gin.HandlerFunc {
	return func(c *gin.Context) {
		if g.Tokens == nil || g.Service == nil {
			response.Fail(c, http.StatusServiceUnavailable, response.CodeNotReady, "服务暂不可用", nil)
			return
		}
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(strings.ToLower(header), "bearer ") {
			response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录或令牌无效", nil)
			return
		}
		token := strings.TrimSpace(header[7:])
		claims, err := g.Tokens.Parse(token)
		if err != nil {
			response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录或令牌无效", nil)
			return
		}
		session, err := g.Service.LoadSessionAccess(c.Request.Context(), claims.UserID, claims.TenantID)
		if err != nil {
			var domain *identity.Error
			if errors.As(err, &domain) && errors.Is(err, identity.ErrUnauthorized) {
				response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, domain.Message, nil)
				return
			}
			response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录或令牌无效", nil)
			return
		}
		permissions := map[string]struct{}{}
		for _, item := range session.Permissions {
			permissions[item] = struct{}{}
		}
		SetPrincipal(c, Principal{
			UserID:      session.UserID,
			Username:    session.Username,
			Nickname:    session.Nickname,
			TenantID:    session.TenantID,
			AuthMethod:  claims.AuthMethod,
			Permissions: permissions,
			Access:      session.Access,
		})
		c.Next()
	}
}

// RequirePermission rejects callers that lack the menu permission identifier.
func RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal, ok := GetPrincipal(c)
		if !ok {
			response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录或令牌无效", nil)
			return
		}
		if !principal.HasPermission(permission) {
			response.Fail(c, http.StatusForbidden, response.CodeForbidden, "没有访问权限", nil)
			return
		}
		c.Next()
	}
}

// RequireTenantMatch ensures path tenant id equals the token tenant.
func RequireTenantMatch(param string) gin.HandlerFunc {
	return requireTenant(param, "")
}

// RequireTenantMatchOrPermission allows the token tenant, or another tenant when the caller has permission.
func RequireTenantMatchOrPermission(param, permission string) gin.HandlerFunc {
	return requireTenant(param, permission)
}

func requireTenant(param, crossTenantPermission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		principal, ok := GetPrincipal(c)
		if !ok {
			response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录或令牌无效", nil)
			return
		}
		tenantID := c.Param(param)
		if tenantID != "" && tenantID != principal.TenantID {
			if crossTenantPermission == "" || !principal.HasPermission(crossTenantPermission) {
				response.Fail(c, http.StatusForbidden, response.CodeForbidden, "不能访问其他租户数据", nil)
				return
			}
		}
		c.Next()
	}
}
