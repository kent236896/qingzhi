package chat

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"qingzhi/enterprise-api/internal/auth"
	"qingzhi/enterprise-api/internal/knowledge"
	"qingzhi/enterprise-api/internal/weknora"
)

type stubRemote struct {
	sessions []weknora.ChatSession
	created  weknora.ChatSession
	askBody  weknora.KnowledgeQARequest
}

func (s *stubRemote) CreateSession(ctx context.Context, title, description string) (weknora.ChatSession, error) {
	if weknora.ExternalUserTokenFrom(ctx) == "" {
		return weknora.ChatSession{}, errors.New("missing external token")
	}
	s.created = weknora.ChatSession{ID: "sess-1", Title: title, Description: description}
	return s.created, nil
}
func (s *stubRemote) GetSession(ctx context.Context, sessionID string) (weknora.ChatSession, error) {
	if weknora.ExternalUserTokenFrom(ctx) == "" {
		return weknora.ChatSession{}, errors.New("missing external token")
	}
	return weknora.ChatSession{ID: sessionID, Title: "t"}, nil
}
func (s *stubRemote) ListSessions(ctx context.Context, page, pageSize int) ([]weknora.ChatSession, int, error) {
	if weknora.ExternalUserTokenFrom(ctx) == "" {
		return nil, 0, errors.New("missing external token")
	}
	return s.sessions, len(s.sessions), nil
}
func (s *stubRemote) UpdateSession(ctx context.Context, sessionID, title, description string) (weknora.ChatSession, error) {
	return weknora.ChatSession{ID: sessionID, Title: title}, nil
}
func (s *stubRemote) DeleteSession(ctx context.Context, sessionID string) error { return nil }
func (s *stubRemote) LoadMessages(ctx context.Context, sessionID string, limit int, before *time.Time) ([]weknora.ChatMessage, error) {
	return []weknora.ChatMessage{{ID: "m1", SessionID: sessionID, Role: "user", Content: "hi"}}, nil
}
func (s *stubRemote) StopSession(ctx context.Context, sessionID, messageID string) error { return nil }
func (s *stubRemote) OpenKnowledgeChatStream(ctx context.Context, sessionID string, req weknora.KnowledgeQARequest) (*http.Response, error) {
	if weknora.ExternalUserTokenFrom(ctx) == "" {
		return nil, errors.New("missing external token")
	}
	s.askBody = req
	return &http.Response{StatusCode: 200, Body: http.NoBody}, nil
}

type stubKnowledge struct {
	remoteIDs map[string]string
	deny      map[string]bool
}

func (s *stubKnowledge) ListChatKnowledgeBases(ctx context.Context, actor knowledge.Actor, keyword string) ([]knowledge.ChatKnowledgeBase, error) {
	return []knowledge.ChatKnowledgeBase{{ID: "kb1", Name: "文档库"}}, nil
}
func (s *stubKnowledge) ResolveRemoteKnowledgeBaseIDs(ctx context.Context, actor knowledge.Actor, platformIDs []string) ([]string, error) {
	out := make([]string, 0, len(platformIDs))
	for _, id := range platformIDs {
		if s.deny[id] {
			return nil, &knowledge.Error{Kind: knowledge.ErrForbidden, Message: "无权查看这个知识库"}
		}
		remote, ok := s.remoteIDs[id]
		if !ok || remote == "" {
			return nil, &knowledge.Error{Kind: knowledge.ErrInvalid, Message: "知识库尚未同步到 WeKnora，无法用于对话"}
		}
		out = append(out, remote)
	}
	return out, nil
}

type stubTokens struct {
	enabled bool
}

func (s *stubTokens) IssueExternalUserToken(ctx context.Context, principal auth.Principal) (string, time.Time, weknora.PlatformIdentity, error) {
	if !s.enabled {
		return "", time.Time{}, weknora.PlatformIdentity{}, errors.New("未配置 WEKNORA_EXTERNAL_USER_HMAC，无法签发终端用户令牌")
	}
	identity := weknora.PlatformIdentity{
		PlatformUserID: principal.UserID, PlatformTenantID: principal.TenantID, WeKnoraTenantID: "10000", AuthMethod: "local",
	}
	return "tok-" + principal.UserID, time.Now().Add(time.Minute), identity, nil
}

type stubMembers struct{}

func (stubMembers) ListMembershipTenantIDs(ctx context.Context, userID string) ([]string, error) {
	return []string{"tenant-a"}, nil
}

func TestAskRequiresTokenAndMapsKnowledgeIDs(t *testing.T) {
	remote := &stubRemote{}
	svc := NewService(remote, &stubKnowledge{remoteIDs: map[string]string{"kb1": "remote-1"}}, &stubTokens{enabled: true}, stubMembers{})
	principal := auth.Principal{UserID: "user-a", Username: "ada", TenantID: "tenant-a"}
	resp, err := svc.Ask(context.Background(), principal, "sess-1", "什么是擎知？", []string{"kb1"})
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if len(remote.askBody.KnowledgeBaseIDs) != 1 || remote.askBody.KnowledgeBaseIDs[0] != "remote-1" {
		t.Fatalf("remote ids = %#v", remote.askBody.KnowledgeBaseIDs)
	}
	if remote.askBody.AgentEnabled {
		t.Fatal("agent mode must stay disabled")
	}
}

func TestAskRejectsUnauthorizedKnowledgeBase(t *testing.T) {
	svc := NewService(&stubRemote{}, &stubKnowledge{deny: map[string]bool{"kb-x": true}}, &stubTokens{enabled: true}, stubMembers{})
	principal := auth.Principal{UserID: "user-a", TenantID: "tenant-a"}
	_, err := svc.Ask(context.Background(), principal, "sess-1", "q", []string{"kb-x"})
	if err == nil {
		t.Fatal("expected forbidden")
	}
	var knowledgeErr *knowledge.Error
	if !errors.As(err, &knowledgeErr) || !errors.Is(err, knowledge.ErrForbidden) {
		t.Fatalf("err = %v", err)
	}
}

func TestAskRejectsMissingHMAC(t *testing.T) {
	svc := NewService(&stubRemote{}, &stubKnowledge{remoteIDs: map[string]string{"kb1": "r1"}}, &stubTokens{enabled: false}, stubMembers{})
	_, err := svc.Ask(context.Background(), auth.Principal{UserID: "u1", TenantID: "t1"}, "sess-1", "q", []string{"kb1"})
	if err == nil || !errors.Is(err, ErrNotReady) {
		t.Fatalf("err = %v", err)
	}
}

func TestCreateSessionInjectsExternalToken(t *testing.T) {
	remote := &stubRemote{}
	svc := NewService(remote, &stubKnowledge{}, &stubTokens{enabled: true}, stubMembers{})
	item, err := svc.CreateSession(context.Background(), auth.Principal{UserID: "u1", TenantID: "t1"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if item.ID != "sess-1" || item.Title != "新对话" {
		t.Fatalf("item = %+v", item)
	}
}
