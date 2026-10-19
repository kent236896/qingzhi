package knowledge

import "testing"

func TestVisibleOrgIDsIncludesDescendants(t *testing.T) {
	parent := map[string]string{"root": "", "child": "root", "leaf": "child", "other": ""}
	got := VisibleOrgIDs([]string{"child"}, parent)
	if _, ok := got["child"]; !ok {
		t.Fatal("missing own department")
	}
	if _, ok := got["leaf"]; !ok {
		t.Fatal("missing descendant")
	}
	if _, ok := got["root"]; ok {
		t.Fatal("parent should stay hidden")
	}
	if _, ok := got["other"]; ok {
		t.Fatal("unrelated department should stay hidden")
	}
}

func TestCanViewUnionsScopesAndKeepsPersonalPrivate(t *testing.T) {
	actor := Actor{UserID: "user", TenantIDs: []string{"t1"}, CanManage: false}
	facts := ScopeFacts{OrgIDs: map[string]struct{}{"org": {}}, PostIDs: map[string]struct{}{"post": {}}, RoleIDs: map[string]struct{}{"role": {}}}
	org := Base{ScopeType: ScopeOrg, ScopeID: "org", TenantIDs: []string{"t1", "t2"}}
	if !CanView(org, nil, actor, facts) {
		t.Fatal("org knowledge base should be visible")
	}
	personal := Base{ScopeType: ScopePersonal, ScopeID: "other", TenantIDs: []string{"t1"}, CreatorID: "other"}
	if CanView(personal, nil, actor, facts) {
		t.Fatal("another user's personal knowledge base should stay hidden")
	}
	if !CanView(personal, []string{"user"}, actor, facts) {
		t.Fatal("shared personal knowledge base should be visible")
	}
	hiddenTenant := Base{ScopeType: ScopeRole, ScopeID: "role", TenantIDs: []string{"t9"}}
	if CanView(hiddenTenant, nil, actor, facts) {
		t.Fatal("knowledge base outside the user's tenants should stay hidden")
	}
	manager := actor
	manager.CanManage = true
	foreign := Base{ScopeType: ScopePost, ScopeID: "elsewhere", TenantIDs: []string{"t1"}}
	if !CanView(foreign, nil, manager, facts) {
		t.Fatal("manager should see organization, post, and role knowledge bases in their tenants")
	}
	if CanView(personal, nil, manager, facts) {
		t.Fatal("manager should not see someone else's personal knowledge base")
	}
	owner := Base{ScopeType: ScopePersonal, ScopeID: "user", TenantIDs: []string{"t9"}}
	if !CanView(owner, nil, actor, facts) {
		t.Fatal("owner should see their own personal knowledge base")
	}
}
