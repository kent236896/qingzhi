package identity

// Data scope constants match sys_roles.data_scope.
const (
	DataScopeAll          = 1
	DataScopeCustom       = 2
	DataScopeDept         = 3
	DataScopeDeptAndChild = 4
	DataScopeSelf         = 5
)

// DataAccess is the effective data permission for one authenticated user in a tenant.
type DataAccess struct {
	Scope  int
	UserID string
	OrgIDs map[string]struct{}
}

// AllowsAll reports whether every row in the tenant is visible.
func (a DataAccess) AllowsAll() bool {
	return a.Scope == DataScopeAll
}

// AllowsSelfOnly reports whether only the current user's own rows are visible.
func (a DataAccess) AllowsSelfOnly() bool {
	return a.Scope == DataScopeSelf
}

// AllowsOrg reports whether an organization id is inside the allowed set.
func (a DataAccess) AllowsOrg(orgID string) bool {
	if a.AllowsAll() {
		return true
	}
	if a.AllowsSelfOnly() || a.OrgIDs == nil {
		return false
	}
	_, ok := a.OrgIDs[orgID]
	return ok
}

// ResolveDataAccess merges role data scopes into one effective access rule.
// All wins; otherwise department sets are unioned; otherwise only self remains.
func ResolveDataAccess(userID, primaryOrgID string, roles []Role, roleDepts map[string][]string, orgs []Organization) DataAccess {
	access := DataAccess{Scope: DataScopeSelf, UserID: userID, OrgIDs: map[string]struct{}{}}
	if len(roles) == 0 {
		return access
	}
	children := childrenByParent(orgs)
	hasScoped := false
	for _, role := range roles {
		switch role.DataScope {
		case DataScopeAll:
			return DataAccess{Scope: DataScopeAll, UserID: userID}
		case DataScopeCustom:
			hasScoped = true
			for _, id := range roleDepts[role.ID] {
				access.OrgIDs[id] = struct{}{}
			}
		case DataScopeDept:
			hasScoped = true
			if primaryOrgID != "" {
				access.OrgIDs[primaryOrgID] = struct{}{}
			}
		case DataScopeDeptAndChild:
			hasScoped = true
			if primaryOrgID != "" {
				for _, id := range collectDescendants(primaryOrgID, children) {
					access.OrgIDs[id] = struct{}{}
				}
			}
		case DataScopeSelf:
			// keep self unless another role expands scope
		default:
			hasScoped = true
		}
	}
	if hasScoped {
		access.Scope = DataScopeCustom
		return access
	}
	return access
}

func childrenByParent(orgs []Organization) map[string][]string {
	out := map[string][]string{}
	for _, org := range orgs {
		out[org.ParentID] = append(out[org.ParentID], org.ID)
	}
	return out
}

func collectDescendants(root string, children map[string][]string) []string {
	result := []string{root}
	queue := []string{root}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, child := range children[current] {
			result = append(result, child)
			queue = append(queue, child)
		}
	}
	return result
}

// FilterOrganizations keeps allowed departments and their ancestors for tree display.
func FilterOrganizations(orgs []Organization, access DataAccess) []Organization {
	if access.AllowsAll() {
		return orgs
	}
	if access.AllowsSelfOnly() {
		return nil
	}
	keep := map[string]struct{}{}
	byID := map[string]Organization{}
	for _, org := range orgs {
		byID[org.ID] = org
	}
	for id := range access.OrgIDs {
		for current := id; current != ""; {
			if _, ok := keep[current]; ok {
				break
			}
			keep[current] = struct{}{}
			org, ok := byID[current]
			if !ok {
				break
			}
			current = org.ParentID
		}
	}
	filtered := make([]Organization, 0, len(keep))
	for _, org := range orgs {
		if _, ok := keep[org.ID]; ok {
			filtered = append(filtered, org)
		}
	}
	return filtered
}

// MemberVisible reports whether a member is inside the data scope.
func MemberVisible(userID string, assignments []OrganizationMembership, access DataAccess) bool {
	if access.AllowsAll() {
		return true
	}
	if access.AllowsSelfOnly() {
		return userID == access.UserID
	}
	if userID == access.UserID {
		return true
	}
	for _, item := range assignments {
		if access.AllowsOrg(item.OrganizationID) {
			return true
		}
	}
	return false
}
