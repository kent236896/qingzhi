package scoped

import "testing"

func TestCanViewScopes(t *testing.T) {
	actor := Actor{UserID: "u1", TenantIDs: []string{"t1"}}
	facts := ScopeFacts{
		OrgIDs:  map[string]struct{}{"org1": {}},
		RoleIDs: map[string]struct{}{"role1": {}},
	}

	personal := Resource{ScopeType: ScopePersonal, ScopeID: "u1", CreatorID: "u1", TenantIDs: []string{"t1"}}
	if !CanView(personal, Shares{}, actor, facts) {
		t.Fatal("owner should see personal")
	}
	other := Resource{ScopeType: ScopePersonal, ScopeID: "u2", CreatorID: "u2", TenantIDs: []string{"t1"}}
	if CanView(other, Shares{}, actor, facts) {
		t.Fatal("other personal hidden")
	}
	if !CanView(other, Shares{UserIDs: []string{"u1"}}, actor, facts) {
		t.Fatal("shared personal visible")
	}

	tenant := Resource{ScopeType: ScopeTenant, ScopeID: "t1", TenantIDs: []string{"t1"}}
	if !CanView(tenant, Shares{}, actor, facts) {
		t.Fatal("tenant scope visible to tenant members")
	}
	foreign := Resource{ScopeType: ScopeTenant, ScopeID: "t9", TenantIDs: []string{"t9"}}
	if CanView(foreign, Shares{}, actor, facts) {
		t.Fatal("foreign tenant hidden")
	}

	org := Resource{ScopeType: ScopeOrg, ScopeID: "org1", TenantIDs: []string{"t1"}}
	if !CanView(org, Shares{}, actor, facts) {
		t.Fatal("org member visible")
	}
	role := Resource{ScopeType: ScopeRole, ScopeID: "role1", TenantIDs: []string{"t1"}}
	if !CanView(role, Shares{}, actor, facts) {
		t.Fatal("role member visible")
	}
}

func TestCanWrite(t *testing.T) {
	owner := Actor{UserID: "u1", TenantIDs: []string{"t1"}}
	manager := Actor{UserID: "m1", TenantIDs: []string{"t1"}, CanManage: true}
	personal := Resource{ScopeType: ScopePersonal, ScopeID: "u1", CreatorID: "u1", TenantIDs: []string{"t1"}}
	if !CanWrite(personal, owner) {
		t.Fatal("owner write")
	}
	if CanWrite(personal, manager) {
		t.Fatal("manager must not write others personal")
	}
	tenant := Resource{ScopeType: ScopeTenant, ScopeID: "t1", TenantIDs: []string{"t1"}}
	if !CanWrite(tenant, manager) {
		t.Fatal("manager write tenant")
	}
	if CanWrite(tenant, owner) {
		t.Fatal("non-manage cannot write tenant")
	}
}
