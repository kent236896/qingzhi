package auth

import (
	"github.com/gin-gonic/gin"
	"qingzhi/enterprise-api/internal/identity"
)

const contextPrincipal = "auth_principal"

// Principal is the authenticated subject for the current request.
type Principal struct {
	UserID      string
	Username    string
	Nickname    string
	TenantID    string
	Permissions map[string]struct{}
	Access      identity.DataAccess
}

// HasPermission reports whether the principal owns the given permission identifier.
func (p Principal) HasPermission(permission string) bool {
	if permission == "" {
		return true
	}
	if p.Permissions == nil {
		return false
	}
	_, ok := p.Permissions[permission]
	return ok
}

// SetPrincipal stores the principal on the Gin context.
func SetPrincipal(c *gin.Context, principal Principal) {
	c.Set(contextPrincipal, principal)
}

// GetPrincipal returns the principal when the request is authenticated.
func GetPrincipal(c *gin.Context) (Principal, bool) {
	value, ok := c.Get(contextPrincipal)
	if !ok {
		return Principal{}, false
	}
	principal, ok := value.(Principal)
	return principal, ok
}
