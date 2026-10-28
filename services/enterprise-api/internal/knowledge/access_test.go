package knowledge

import "testing"

func TestVisibleOrgIDsIncludesAncestorsAndDescendants(t *testing.T) {
	// company
	// ├── finance
	// │   └── finance-team
	// └── rd
	parent := map[string]string{
		"company": "", "finance": "company", "finance-team": "finance", "rd": "company",
	}

	fromFinance := VisibleOrgIDs([]string{"finance"}, parent)
	for _, id := range []string{"company", "finance", "finance-team"} {
		if _, ok := fromFinance[id]; !ok {
			t.Fatalf("finance chain missing %s", id)
		}
	}
	if _, ok := fromFinance["rd"]; ok {
		t.Fatal("sibling rd must stay hidden for finance assignee")
	}

	fromTeam := VisibleOrgIDs([]string{"finance-team"}, parent)
	if _, ok := fromTeam["finance"]; !ok {
		t.Fatal("child assignee should see ancestor finance")
	}
	if _, ok := fromTeam["company"]; !ok {
		t.Fatal("child assignee should see ancestor company")
	}
	if _, ok := fromTeam["rd"]; ok {
		t.Fatal("sibling rd must stay hidden for finance-team assignee")
	}

	fromCompany := VisibleOrgIDs([]string{"company"}, parent)
	if _, ok := fromCompany["finance"]; !ok {
		t.Fatal("company assignee should see descendant finance")
	}
	if _, ok := fromCompany["rd"]; !ok {
		t.Fatal("company assignee should see descendant rd")
	}
}

func TestCanViewUnionsScopesAndKeepsPersonalPrivate(t *testing.T) {
	actor := Actor{UserID: "user", TenantIDs: []string{"t1"}, CanManage: false}
	facts := ScopeFacts{OrgIDs: map[string]struct{}{"org": {}}, PostIDs: map[string]struct{}{"post": {}}, RoleIDs: map[string]struct{}{"role": {}}}
	org := Base{ScopeType: ScopeOrg, ScopeID: "org", TenantIDs: []string{"t1", "t2"}}
	if !CanView(org, Shares{}, actor, facts) {
		t.Fatal("org knowledge base should be visible")
	}
	personal := Base{ScopeType: ScopePersonal, ScopeID: "other", TenantIDs: []string{"t1"}, CreatorID: "other"}
	if CanView(personal, Shares{}, actor, facts) {
		t.Fatal("another user's personal knowledge base should stay hidden")
	}
	if !CanView(personal, Shares{UserIDs: []string{"user"}}, actor, facts) {
		t.Fatal("shared personal knowledge base should be visible")
	}
	hiddenTenant := Base{ScopeType: ScopeRole, ScopeID: "role", TenantIDs: []string{"t9"}}
	if CanView(hiddenTenant, Shares{}, actor, facts) {
		t.Fatal("knowledge base outside the user's tenants should stay hidden")
	}
	manager := actor
	manager.CanManage = true
	foreign := Base{ScopeType: ScopePost, ScopeID: "elsewhere", TenantIDs: []string{"t1"}}
	if !CanView(foreign, Shares{}, manager, facts) {
		t.Fatal("manager should see organization, post, and role knowledge bases in their tenants")
	}
	if CanView(personal, Shares{}, manager, facts) {
		t.Fatal("manager should not see someone else's personal knowledge base")
	}
	owner := Base{ScopeType: ScopePersonal, ScopeID: "user", TenantIDs: []string{"t9"}}
	if !CanView(owner, Shares{}, actor, facts) {
		t.Fatal("owner should see their own personal knowledge base")
	}
}

func TestCanViewSharedOrgAndRoleTargets(t *testing.T) {
	actor := Actor{UserID: "user", TenantIDs: []string{"t1"}}
	facts := ScopeFacts{
		OrgIDs:  map[string]struct{}{"shared-org": {}},
		RoleIDs: map[string]struct{}{"shared-role": {}},
	}
	orgKB := Base{ScopeType: ScopeOrg, ScopeID: "owner-org", TenantIDs: []string{"t1"}}
	if CanView(orgKB, Shares{}, actor, facts) {
		t.Fatal("foreign org knowledge base should stay hidden without share")
	}
	if !CanView(orgKB, Shares{OrgIDs: []string{"shared-org"}}, actor, facts) {
		t.Fatal("org knowledge base shared to caller's org should be visible")
	}
	roleKB := Base{ScopeType: ScopeRole, ScopeID: "owner-role", TenantIDs: []string{"t1"}}
	if !CanView(roleKB, Shares{RoleIDs: []string{"shared-role"}}, actor, facts) {
		t.Fatal("role knowledge base shared to caller's role should be visible")
	}
	if !CanView(roleKB, Shares{OrgIDs: []string{"shared-org"}}, actor, facts) {
		t.Fatal("role knowledge base shared to caller's org should be visible")
	}
}

func TestCanShare(t *testing.T) {
	owner := Actor{UserID: "u1", TenantIDs: []string{"t1"}}
	personal := Base{ScopeType: ScopePersonal, ScopeID: "u1", TenantIDs: []string{"t1"}, CreatorID: "u1"}
	if !CanShare(personal, owner) {
		t.Fatal("owner should share personal knowledge base")
	}
	manager := Actor{UserID: "m1", TenantIDs: []string{"t1"}, CanManage: true}
	org := Base{ScopeType: ScopeOrg, ScopeID: "org", TenantIDs: []string{"t1"}}
	if !CanShare(org, manager) {
		t.Fatal("manager should share org knowledge base")
	}
	post := Base{ScopeType: ScopePost, ScopeID: "post", TenantIDs: []string{"t1"}}
	if CanShare(post, manager) {
		t.Fatal("post knowledge base should not support sharing")
	}
}
