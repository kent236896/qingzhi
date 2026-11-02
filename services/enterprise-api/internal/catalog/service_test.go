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
	created int
	deleted []string
}

func (s *stubAgents) CreateRemoteAgent(_ context.Context, input weknora.CreateRemoteAgentInput) (weknora.RemoteAgent, error) {
	s.created++
	return weknora.RemoteAgent{ID: "remote-agent-" + input.Name, Name: input.Name}, nil
}
func (s *stubAgents) UpdateRemoteAgent(context.Context, string, weknora.CreateRemoteAgentInput) (weknora.RemoteAgent, error) {
	return weknora.RemoteAgent{}, nil
}
func (s *stubAgents) DeleteRemoteAgent(_ context.Context, id string) error {
	s.deleted = append(s.deleted, id)
	return nil
}

type stubMCP struct{}

func (stubMCP) CreateRemoteMCPService(_ context.Context, input weknora.CreateRemoteMCPInput) (weknora.RemoteMCPService, error) {
	return weknora.RemoteMCPService{ID: "remote-mcp-" + input.Name, Name: input.Name, TransportType: input.TransportType}, nil
}
func (stubMCP) UpdateRemoteMCPService(context.Context, string, map[string]any) (weknora.RemoteMCPService, error) {
	return weknora.RemoteMCPService{}, nil
}
func (stubMCP) DeleteRemoteMCPService(context.Context, string) error { return nil }
func (stubMCP) TestRemoteMCPService(context.Context, string) (map[string]any, error) {
	return map[string]any{"ok": true}, nil
}

func TestCreateAndListPersonalAgent(t *testing.T) {
	agents := newMemStore()
	remote := &stubAgents{}
	svc := &Service{
		agents: agents, mcp: newMemStore(), identity: stubIdentity{},
		agentAPI: remote, mcpAPI: stubMCP{},
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
		agentAPI: &stubAgents{}, mcpAPI: stubMCP{},
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
		agentAPI: &stubAgents{}, mcpAPI: stubMCP{},
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
		agentAPI: remote, mcpAPI: stubMCP{},
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
