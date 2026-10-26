package knowledge

const (
	ScopePersonal = "personal"
	ScopeOrg      = "org_node"
	ScopePost     = "position"
	ScopeRole     = "role"

	ShareTargetOrg  = "org_node"
	ShareTargetRole = "role"
)

// Actor is the signed-in user for a knowledge-base decision.
type Actor struct {
	UserID      string
	Username    string
	TenantIDs   []string
	CanManage   bool
	CanQueryAll bool
}

// ScopeFacts are the caller's org, post, and role memberships.
type ScopeFacts struct {
	OrgIDs  map[string]struct{}
	PostIDs map[string]struct{}
	RoleIDs map[string]struct{}
}

// Shares are extra viewers beyond the knowledge base's own scope.
type Shares struct {
	UserIDs []string
	OrgIDs  []string
	RoleIDs []string
}

// VisibleOrgIDs returns the caller's departments and every department below them.
func VisibleOrgIDs(assigned []string, parentOf map[string]string) map[string]struct{} {
	children := map[string][]string{}
	for id, parent := range parentOf {
		children[parent] = append(children[parent], id)
	}
	out := map[string]struct{}{}
	var walk func(id string)
	walk = func(id string) {
		if id == "" {
			return
		}
		if _, seen := out[id]; seen {
			return
		}
		out[id] = struct{}{}
		for _, child := range children[id] {
			walk(child)
		}
	}
	for _, id := range assigned {
		walk(id)
	}
	return out
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

// CanView reports whether the actor may read this knowledge base.
func CanView(base Base, shares Shares, actor Actor, facts ScopeFacts) bool {
	if base.ScopeType == ScopePersonal {
		if base.ScopeID == actor.UserID {
			return true
		}
		if !overlaps(base.TenantIDs, actor.TenantIDs) {
			return false
		}
		for _, userID := range shares.UserIDs {
			if userID == actor.UserID {
				return true
			}
		}
		return false
	}
	if !overlaps(base.TenantIDs, actor.TenantIDs) && !actor.CanQueryAll {
		return false
	}
	if actor.CanManage {
		return true
	}
	switch base.ScopeType {
	case ScopeOrg:
		if _, ok := facts.OrgIDs[base.ScopeID]; ok {
			return true
		}
	case ScopePost:
		if _, ok := facts.PostIDs[base.ScopeID]; ok {
			return true
		}
	case ScopeRole:
		if _, ok := facts.RoleIDs[base.ScopeID]; ok {
			return true
		}
	default:
		return false
	}
	// 组织/角色知识库还可分享给其他组织或角色；命中其一即可查看。
	if base.ScopeType == ScopeOrg || base.ScopeType == ScopeRole {
		return hasAny(shares.OrgIDs, facts.OrgIDs) || hasAny(shares.RoleIDs, facts.RoleIDs)
	}
	return false
}

// CanWrite reports whether the actor may change files or the knowledge base.
func CanWrite(base Base, actor Actor) bool {
	if !overlaps(base.TenantIDs, actor.TenantIDs) && base.CreatorID != actor.UserID {
		return false
	}
	if base.ScopeType == ScopePersonal {
		return base.ScopeID == actor.UserID
	}
	return actor.CanManage && overlaps(base.TenantIDs, actor.TenantIDs)
}

// CanShare reports whether the actor may edit share targets.
func CanShare(base Base, actor Actor) bool {
	if !CanWrite(base, actor) {
		return false
	}
	return base.ScopeType == ScopePersonal || base.ScopeType == ScopeOrg || base.ScopeType == ScopeRole
}
