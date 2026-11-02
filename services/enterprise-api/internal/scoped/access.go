package scoped

const (
	ScopePersonal = "personal"
	ScopeTenant   = "tenant"
	ScopeOrg      = "org_node"
	ScopeRole     = "role"
)

// Actor is the signed-in user for a visibility decision.
type Actor struct {
	UserID    string
	Username  string
	TenantIDs []string
	CanManage bool
}

// ScopeFacts are the caller's org and role memberships.
type ScopeFacts struct {
	OrgIDs  map[string]struct{}
	RoleIDs map[string]struct{}
}

// Shares are extra viewers beyond the resource's own scope.
type Shares struct {
	UserIDs []string
	OrgIDs  []string
	RoleIDs []string
}

// Resource is the minimum shape needed for CanView / CanWrite.
type Resource struct {
	ScopeType string
	ScopeID   string
	CreatorID string
	TenantIDs []string
}

func overlaps(left, right []string) bool {
	set := map[string]struct{}{}
	for _, item := range left {
		set[item] = struct{}{}
	}
	for _, item := range right {
		if _, ok := set[item]; ok {
			return true
		}
	}
	return false
}

func hasAny(ids []string, facts map[string]struct{}) bool {
	for _, id := range ids {
		if _, ok := facts[id]; ok {
			return true
		}
	}
	return false
}

// ValidScope reports whether scope_type is supported for agents/toolbox.
func ValidScope(scopeType string) bool {
	switch scopeType {
	case ScopePersonal, ScopeTenant, ScopeOrg, ScopeRole:
		return true
	default:
		return false
	}
}

// CanView reports whether the actor may see this resource.
// manage 只提升写权限，不绕过组织/角色可见范围。
func CanView(item Resource, shares Shares, actor Actor, facts ScopeFacts) bool {
	if item.ScopeType == ScopePersonal {
		if item.ScopeID == actor.UserID || item.CreatorID == actor.UserID {
			return true
		}
		if !overlaps(item.TenantIDs, actor.TenantIDs) {
			return false
		}
		for _, userID := range shares.UserIDs {
			if userID == actor.UserID {
				return true
			}
		}
		return false
	}
	if !overlaps(item.TenantIDs, actor.TenantIDs) {
		return false
	}
	switch item.ScopeType {
	case ScopeTenant:
		return true
	case ScopeOrg:
		if _, ok := facts.OrgIDs[item.ScopeID]; ok {
			return true
		}
		return hasAny(shares.OrgIDs, facts.OrgIDs) || hasAny(shares.RoleIDs, facts.RoleIDs)
	case ScopeRole:
		if _, ok := facts.RoleIDs[item.ScopeID]; ok {
			return true
		}
		return hasAny(shares.OrgIDs, facts.OrgIDs) || hasAny(shares.RoleIDs, facts.RoleIDs)
	default:
		return false
	}
}

// CanWrite reports whether the actor may change the resource.
func CanWrite(item Resource, actor Actor) bool {
	if item.ScopeType == ScopePersonal {
		return item.ScopeID == actor.UserID || item.CreatorID == actor.UserID
	}
	if !overlaps(item.TenantIDs, actor.TenantIDs) {
		return false
	}
	return actor.CanManage
}
