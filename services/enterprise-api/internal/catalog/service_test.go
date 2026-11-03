package catalog

import (
	"context"
	"sync"
	"testing"

	"qingzhi/enterprise-api/internal/identity"
	"qingzhi/enterprise-api/internal/scoped"
	"qingzhi/enterprise-api/internal/weknora"
)

type memStore struct {
	mu    sync.Mutex
	items map[string]Item
}

func newMemStore() *memStore {
	return &memStore{items: map[string]Item{}}
}

func (s *memStore) List(context.Context) ([]Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Item, 0, len(s.items))
	for _, item := range s.items {
		out = append(out, item)
	}
	return out, nil
}

func (s *memStore) Get(_ context.Context, id string) (Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[id]
	if !ok {
		return Item{}, ErrNotFound
	}
	return item, nil
}

func (s *memStore) Insert(_ context.Context, item Item) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[item.ID] = item
	return nil
}

func (s *memStore) Update(_ context.Context, item Item) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[item.ID]; !ok {
		return ErrNotFound
	}
	s.items[item.ID] = item
	return nil
}

func (s *memStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; !ok {
		return ErrNotFound
	}
	delete(s.items, id)
	return nil
}

func (s *memStore) ReplaceTenants(_ context.Context, id string, tenantIDs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[id]
	if !ok {
		return ErrNotFound
	}
	item.TenantIDs = tenantIDs
	s.items[id] = item
	return nil
}

func (s *memStore) ReplaceShares(_ context.Context, id string, userIDs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[id]
	if !ok {
		return ErrNotFound
	}
	item.ShareUserIDs = userIDs
	s.items[id] = item
	return nil
}

type stubIdentity struct{}

func (stubIdentity) ListAssignmentRecords(context.Context, string, string) ([]identity.OrganizationMembership, error) {
	return nil, nil
}
func (stubIdentity) ListOrganizationRecords(context.Context, string) ([]identity.Organization, error) {
	return nil, nil
}
func (stubIdentity) ListUserRoleIDs(context.Context, string, string) ([]string, error) {
	return nil, nil
}
func (stubIdentity) FindOrganizationByID(context.Context, string) (identity.Organization, error) {
	return identity.Organization{}, identity.ErrNotFound
}
func (stubIdentity) GetRole(context.Context, string, string) (identity.Role, error) {
	return identity.Role{}, identity.ErrNotFound
}

type stubAgents struct {
	created    int
	copied     int
	updated    int
	deleted    []string
	lastConfig map[string]any
}

func (s *stubAgents) CreateRemoteAgent(_ context.Context, input weknora.CreateRemoteAgentInput) (weknora.RemoteAgent, error) {
	s.created++
	s.lastConfig = input.Config
	return weknora.RemoteAgent{ID: "remote-agent-" + input.Name, Name: input.Name}, nil
}
func (s *stubAgents) UpdateRemoteAgent(_ context.Context, id string, input weknora.CreateRemoteAgentInput) (weknora.RemoteAgent, error) {
	s.updated++
	s.lastConfig = input.Config
	return weknora.RemoteAgent{ID: id, Name: input.Name}, nil
}
func (s *stubAgents) DeleteRemoteAgent(_ context.Context, id string) error {
	s.deleted = append(s.deleted, id)
	return nil
}
func (s *stubAgents) CopyRemoteAgent(_ context.Context, id string) (weknora.RemoteAgent, error) {
	s.copied++
	return weknora.RemoteAgent{ID: "remote-copy-" + id, Name: "副本"}, nil
}

type stubMCP struct {
	creds map[string]string
}

func (stubMCP) CreateRemoteMCPService(_ context.Context, input weknora.CreateRemoteMCPInput) (weknora.RemoteMCPService, error) {
	return weknora.RemoteMCPService{ID: "remote-mcp-" + input.Name, Name: input.Name, TransportType: input.TransportType}, nil
}
func (stubMCP) UpdateRemoteMCPService(_ context.Context, id string, updates map[string]any) (weknora.RemoteMCPService, error) {
	name, _ := updates["name"].(string)
	return weknora.RemoteMCPService{ID: id, Name: name}, nil
}
func (stubMCP) DeleteRemoteMCPService(context.Context, string) error { return nil }
func (stubMCP) TestRemoteMCPService(context.Context, string) (map[string]any, error) {
	return map[string]any{"ok": true}, nil
}
func (m *stubMCP) PutRemoteMCPCredentials(_ context.Context, _ string, values map[string]string) (weknora.MCPCredentialsStatus, error) {
	if m.creds == nil {
		m.creds = map[string]string{}
	}
	fields := map[string]weknora.MCPCredentialField{}
	for k, v := range values {
		m.creds[k] = v
		fields[k] = weknora.MCPCredentialField{Configured: v != ""}
	}
	return weknora.MCPCredentialsStatus{Fields: fields}, nil
}
func (m *stubMCP) DeleteRemoteMCPCredential(_ context.Context, _, field string) error {
	delete(m.creds, field)
	return nil
}
func (stubMCP) RefreshRemoteMCPMetadata(context.Context, string) (map[string]any, error) {
	return map[string]any{"refreshed": true}, nil
}
func (stubMCP) GetRemoteMCPMetadata(context.Context, string) (any, error) {
	return map[string]any{"tools": []any{}}, nil
}
func (stubMCP) GetRemoteMCPOAuthStatus(_ context.Context, _, attempt string) (weknora.MCPOAuthStatus, error) {
	if attempt == "done" {
		return weknora.MCPOAuthStatus{Authorized: true, State: "authorized"}, nil
	}
	return weknora.MCPOAuthStatus{Authorized: false, State: "reauth_required"}, nil
}
func (stubMCP) CreateRemoteMCPAuthorizeURL(_ context.Context, id, redirectURI, frontendRedirect string) (weknora.MCPAuthorizeURLResult, error) {
	return weknora.MCPAuthorizeURLResult{
		AuthorizationURL:     "https://auth.example/authorize?svc=" + id + "&r=" + redirectURI + "&f=" + frontendRedirect,
		AuthorizationAttempt: "attempt-1",
	}, nil
}
func (stubMCP) RevokeRemoteMCPOAuthToken(context.Context, string) error { return nil }

func TestCreateAndListPersonalAgent(t *testing.T) {
	agents := newMemStore()
	remote := &stubAgents{}
	svc := &Service{
		agents: agents, mcp: newMemStore(), identity: stubIdentity{},
		agentAPI: remote, mcpAPI: &stubMCP{},
	}
	actor := Actor{UserID: "u1", Username: "alice", TenantIDs: []string{"t1"}}
	item, err := svc.CreateAgent(context.Background(), actor, CreateInput{
		Name: "助手A", ScopeType: scoped.ScopePersonal, TenantIDs: []string{"t1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if item.ScopeID != "u1" || !item.Writable {
		t.Fatalf("unexpected item: %+v", item)
	}
	if remote.created != 1 {
		t.Fatal("expected remote create")
	}
	list, err := svc.ListAgents(context.Background(), actor)
	if err != nil || len(list) != 1 {
		t.Fatalf("list=%v err=%v", list, err)
	}
	other := Actor{UserID: "u2", Username: "bob", TenantIDs: []string{"t1"}}
	hidden, err := svc.ListAgents(context.Background(), other)
	if err != nil || len(hidden) != 0 {
		t.Fatalf("other user should not see personal agent: %v", hidden)
	}
}

func TestTenantAgentVisibility(t *testing.T) {
	agents := newMemStore()
	svc := &Service{
		agents: agents, mcp: newMemStore(), identity: stubIdentity{},
		agentAPI: &stubAgents{}, mcpAPI: &stubMCP{},
	}
	manager := Actor{UserID: "m1", Username: "mgr", TenantIDs: []string{"t1"}, CanManage: true}
	_, err := svc.CreateAgent(context.Background(), manager, CreateInput{
		Name: "租户助手", ScopeType: scoped.ScopeTenant, ScopeID: "t1", TenantIDs: []string{"t1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	member := Actor{UserID: "u1", Username: "alice", TenantIDs: []string{"t1"}}
	list, err := svc.ListAgents(context.Background(), member)
	if err != nil || len(list) != 1 {
		t.Fatalf("tenant member should see tenant agent: %v %v", list, err)
	}
	if list[0].Writable {
		t.Fatal("member must not write tenant agent")
	}
	outsider := Actor{UserID: "u9", Username: "x", TenantIDs: []string{"t9"}}
	hidden, err := svc.ListAgents(context.Background(), outsider)
	if err != nil || len(hidden) != 0 {
		t.Fatalf("outsider should not see: %v", hidden)
	}
}

func TestNonManagerCannotCreateTenantAgent(t *testing.T) {
	svc := &Service{
		agents: newMemStore(), mcp: newMemStore(), identity: stubIdentity{},
		agentAPI: &stubAgents{}, mcpAPI: &stubMCP{},
	}
	actor := Actor{UserID: "u1", Username: "alice", TenantIDs: []string{"t1"}}
	_, err := svc.CreateAgent(context.Background(), actor, CreateInput{
		Name: "租户助手", ScopeType: scoped.ScopeTenant, ScopeID: "t1", TenantIDs: []string{"t1"},
	})
	if err == nil {
		t.Fatal("expected forbidden")
	}
	var domain *Error
	if !asDomain(err, &domain) || domain.Kind != ErrForbidden {
		t.Fatalf("want forbidden, got %v", err)
	}
}

func TestDeleteAgentRemovesRemote(t *testing.T) {
	agents := newMemStore()
	remote := &stubAgents{}
	svc := &Service{
		agents: agents, mcp: newMemStore(), identity: stubIdentity{},
		agentAPI: remote, mcpAPI: &stubMCP{},
	}
	actor := Actor{UserID: "u1", Username: "alice", TenantIDs: []string{"t1"}}
	item, err := svc.CreateAgent(context.Background(), actor, CreateInput{
		Name: "助手", ScopeType: scoped.ScopePersonal, TenantIDs: []string{"t1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	stored, _ := agents.Get(context.Background(), item.ID)
	if stored.RemoteID == "" {
		t.Fatal("remote id missing in store")
	}
	if err := svc.DeleteAgent(context.Background(), actor, item.ID); err != nil {
		t.Fatal(err)
	}
	if len(remote.deleted) != 1 || remote.deleted[0] != stored.RemoteID {
		t.Fatalf("remote delete=%v", remote.deleted)
	}
}

func TestUpdateAndCopyAgent(t *testing.T) {
	agents := newMemStore()
	remote := &stubAgents{}
	svc := &Service{
		agents: agents, mcp: newMemStore(), identity: stubIdentity{},
		agentAPI: remote, mcpAPI: &stubMCP{},
	}
	actor := Actor{UserID: "u1", Username: "alice", TenantIDs: []string{"t1"}}
	item, err := svc.CreateAgent(context.Background(), actor, CreateInput{
		Name: "助手", Description: "旧", ScopeType: scoped.ScopePersonal, TenantIDs: []string{"t1"},
		Config: map[string]any{"agent_mode": "quick-answer"},
	})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := svc.UpdateAgent(context.Background(), actor, item.ID, UpdateInput{
		Name: "助手改", Description: "新", Config: map[string]any{"agent_mode": "quick-answer"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "助手改" || remote.updated != 1 {
		t.Fatalf("update=%+v remote.updated=%d", updated, remote.updated)
	}
	copied, err := svc.CopyAgent(context.Background(), actor, item.ID, CopyInput{})
	if err != nil {
		t.Fatal(err)
	}
	if remote.copied != 1 || copied.ID == item.ID || copied.Name == "" {
		t.Fatalf("copy=%+v remote.copied=%d", copied, remote.copied)
	}
	list, err := svc.ListAgents(context.Background(), actor)
	if err != nil || len(list) != 2 {
		t.Fatalf("list=%v err=%v", list, err)
	}
}

func TestUpdateMCPAndSharedCredentials(t *testing.T) {
	mcpStore := newMemStore()
	remote := &stubMCP{}
	svc := &Service{
		agents: newMemStore(), mcp: mcpStore, identity: stubIdentity{},
		agentAPI: &stubAgents{}, mcpAPI: remote,
	}
	actor := Actor{UserID: "u1", Username: "alice", TenantIDs: []string{"t1"}}
	item, err := svc.CreateMCP(context.Background(), actor, CreateInput{
		Name: "工具", ScopeType: scoped.ScopePersonal, TenantIDs: []string{"t1"},
		TransportType: "sse", URL: "https://example.com/mcp", AuthMode: MCPAuthModeShared,
	})
	if err != nil {
		t.Fatal(err)
	}
	updated, err := svc.UpdateMCP(context.Background(), actor, item.ID, UpdateInput{
		Name: "工具改", URL: "https://example.com/mcp2", TransportType: "sse", AuthMode: MCPAuthModeShared,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.URL != "https://example.com/mcp2" || updated.AuthMode != MCPAuthModeShared {
		t.Fatalf("unexpected %+v", updated)
	}
	status, err := svc.PutMCPCredentials(context.Background(), actor, item.ID, map[string]string{"api_key": "secret"})
	if err != nil || !status.Fields["api_key"].Configured {
		t.Fatalf("creds=%+v err=%v", status, err)
	}
	// Switch to user_oauth locally then credentials must be rejected.
	stored, _ := mcpStore.Get(context.Background(), item.ID)
	stored.AuthMode = MCPAuthModeUserOAuth
	_ = mcpStore.Update(context.Background(), stored)
	_, err = svc.PutMCPCredentials(context.Background(), actor, item.ID, map[string]string{"api_key": "x"})
	if err == nil {
		t.Fatal("expected invalid for user_oauth credentials")
	}
}

func TestMCPOAuthUserFlow(t *testing.T) {
	mcpStore := newMemStore()
	svc := &Service{
		agents: newMemStore(), mcp: mcpStore, identity: stubIdentity{},
		agentAPI: &stubAgents{}, mcpAPI: &stubMCP{},
		weknoraBaseURL: "http://weknora.test:8083",
	}
	actor := Actor{UserID: "u1", Username: "alice", TenantIDs: []string{"t1"}}
	item, err := svc.CreateMCP(context.Background(), actor, CreateInput{
		Name: "oauth工具", ScopeType: scoped.ScopePersonal, TenantIDs: []string{"t1"},
		TransportType: "sse", URL: "https://example.com/mcp", AuthMode: MCPAuthModeUserOAuth,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.StartMCPOAuthAuthorize(context.Background(), actor, item.ID, "http://app/toolbox")
	if err != nil {
		t.Fatal(err)
	}
	st, err := svc.GetMCPOAuthStatus(context.Background(), actor, item.ID, "")
	if err != nil || st.Authorized {
		t.Fatalf("status=%+v err=%v", st, err)
	}
	st, err = svc.GetMCPOAuthStatus(context.Background(), actor, item.ID, "done")
	if err != nil || !st.Authorized {
		t.Fatalf("attempt status=%+v err=%v", st, err)
	}
	if err := svc.RevokeMCPOAuthToken(context.Background(), actor, item.ID); err != nil {
		t.Fatal(err)
	}
	// shared MCP must not start oauth
	shared, err := svc.CreateMCP(context.Background(), actor, CreateInput{
		Name: "shared工具", ScopeType: scoped.ScopePersonal, TenantIDs: []string{"t1"},
		TransportType: "sse", URL: "https://example.com/s", AuthMode: MCPAuthModeShared,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.StartMCPOAuthAuthorize(context.Background(), actor, shared.ID, "http://app/toolbox")
	if err == nil {
		t.Fatal("expected invalid for shared auth_mode oauth")
	}
}

func asDomain(err error, target **Error) bool {
	if err == nil {
		return false
	}
	e, ok := err.(*Error)
	if !ok {
		return false
	}
	*target = e
	return true
}
