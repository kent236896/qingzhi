package identity

import "testing"

func TestResolveDataAccessAllWins(t *testing.T) {
	access := ResolveDataAccess("u1", "org1", []Role{
		{ID: "r1", DataScope: DataScopeSelf},
		{ID: "r2", DataScope: DataScopeAll},
	}, nil, nil)
	if !access.AllowsAll() {
		t.Fatalf("access = %#v", access)
	}
}

func TestResolveDataAccessDeptAndChild(t *testing.T) {
	orgs := []Organization{
		{ID: "root", ParentID: ""},
		{ID: "child", ParentID: "root"},
		{ID: "leaf", ParentID: "child"},
		{ID: "other", ParentID: ""},
	}
	access := ResolveDataAccess("u1", "child", []Role{{ID: "r1", DataScope: DataScopeDeptAndChild}}, nil, orgs)
	if access.AllowsAll() || access.AllowsSelfOnly() {
		t.Fatalf("access = %#v", access)
	}
	if !access.AllowsOrg("child") || !access.AllowsOrg("leaf") || access.AllowsOrg("other") {
		t.Fatalf("orgs = %#v", access.OrgIDs)
	}
}

func TestFilterOrganizationsKeepsAncestors(t *testing.T) {
	orgs := []Organization{
		{ID: "root", ParentID: "", Name: "R"},
		{ID: "child", ParentID: "root", Name: "C"},
		{ID: "other", ParentID: "", Name: "O"},
	}
	access := DataAccess{Scope: DataScopeCustom, OrgIDs: map[string]struct{}{"child": {}}}
	filtered := FilterOrganizations(orgs, access)
	if len(filtered) != 2 {
		t.Fatalf("filtered = %#v", filtered)
	}
}
