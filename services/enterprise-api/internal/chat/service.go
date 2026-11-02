package chat

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"qingzhi/enterprise-api/internal/auth"
	"qingzhi/enterprise-api/internal/knowledge"
	"qingzhi/enterprise-api/internal/weknora"
)

var (
	ErrNotReady     = errors.New("not ready")
	ErrForbidden    = errors.New("forbidden")
	ErrInvalid      = errors.New("invalid")
	ErrNotFound     = errors.New("not found")
	ErrUnauthorized = errors.New("unauthorized")
)

// Error is a domain error for chat APIs.
type Error struct {
	Kind    error
	Message string
}

func (e *Error) Error() string { return e.Message }
func (e *Error) Unwrap() error { return e.Kind }

func invalid(msg string) error   { return &Error{Kind: ErrInvalid, Message: msg} }
func notReady(msg string) error  { return &Error{Kind: ErrNotReady, Message: msg} }
func forbidden(msg string) error { return &Error{Kind: ErrForbidden, Message: msg} }
func notFound(msg string) error  { return &Error{Kind: ErrNotFound, Message: msg} }

// ChatRemote is the WeKnora session/QA surface used by this service.
type ChatRemote interface {
	CreateSession(ctx context.Context, title, description string) (weknora.ChatSession, error)
	GetSession(ctx context.Context, sessionID string) (weknora.ChatSession, error)
	ListSessions(ctx context.Context, page, pageSize int) ([]weknora.ChatSession, int, error)
	UpdateSession(ctx context.Context, sessionID, title, description string) (weknora.ChatSession, error)
	DeleteSession(ctx context.Context, sessionID string) error
	LoadMessages(ctx context.Context, sessionID string, limit int, before *time.Time) ([]weknora.ChatMessage, error)
	StopSession(ctx context.Context, sessionID, messageID string) error
	OpenKnowledgeChatStream(ctx context.Context, sessionID string, req weknora.KnowledgeQARequest) (*http.Response, error)
}

// KnowledgeAccess resolves authorized knowledge bases for chat.
type KnowledgeAccess interface {
	ListChatKnowledgeBases(ctx context.Context, actor knowledge.Actor, keyword string) ([]knowledge.ChatKnowledgeBase, error)
	ResolveRemoteKnowledgeBaseIDs(ctx context.Context, actor knowledge.Actor, platformIDs []string) ([]string, error)
}

// TokenIssuer signs X-External-User-Token for the current principal.
type TokenIssuer interface {
	IssueExternalUserToken(ctx context.Context, principal auth.Principal) (string, time.Time, weknora.PlatformIdentity, error)
}

// MembershipLookup loads tenant memberships for knowledge Actor construction.
type MembershipLookup interface {
	ListMembershipTenantIDs(ctx context.Context, userID string) ([]string, error)
}

// Service is the platform chat orchestrator.
type Service struct {
	remote     ChatRemote
	knowledge  KnowledgeAccess
	tokens     TokenIssuer
	membership MembershipLookup
	owners     SessionOwnerStore
	log        *slog.Logger
}

func NewService(remote ChatRemote, knowledge KnowledgeAccess, tokens TokenIssuer, membership MembershipLookup, owners SessionOwnerStore, log *slog.Logger) *Service {
	return &Service{remote: remote, knowledge: knowledge, tokens: tokens, membership: membership, owners: owners, log: log}
}

func (s *Service) ActorFromPrincipal(ctx context.Context, principal auth.Principal) (knowledge.Actor, error) {
	actor := knowledge.Actor{
		UserID:      principal.UserID,
		Username:    principal.Username,
		CanManage:   principal.HasPermission("knowledge:manage"),
		CanQueryAll: principal.HasPermission("system:tenant:query"),
	}
	if s.membership != nil {
		ids, err := s.membership.ListMembershipTenantIDs(ctx, principal.UserID)
		if err != nil {
			return knowledge.Actor{}, err
		}
		actor.TenantIDs = ids
	}
	if principal.TenantID != "" {
		found := false
		for _, id := range actor.TenantIDs {
			if id == principal.TenantID {
				found = true
				break
			}
		}
		if !found {
			actor.TenantIDs = append(actor.TenantIDs, principal.TenantID)
		}
	}
	return actor, nil
}

func (s *Service) withUser(ctx context.Context, principal auth.Principal) (context.Context, weknora.PlatformIdentity, error) {
	if s == nil || s.remote == nil {
		return ctx, weknora.PlatformIdentity{}, notReady("WeKnora 未配置")
	}
	if s.tokens == nil {
		return ctx, weknora.PlatformIdentity{}, notReady("未配置 WEKNORA_EXTERNAL_USER_HMAC，无法签发终端用户令牌")
	}
	token, _, identity, err := s.tokens.IssueExternalUserToken(ctx, principal)
	if err != nil {
		return ctx, weknora.PlatformIdentity{}, notReady(err.Error())
	}
	return weknora.WithExternalUserToken(ctx, token), identity, nil
}

func (s *Service) assertOwned(ctx context.Context, principal auth.Principal, sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return invalid("会话标识不合法")
	}
	if s.owners == nil {
		return notReady("会话归属存储不可用")
	}
	ok, err := s.owners.OwnedBy(ctx, principal.UserID, sessionID)
	if err != nil {
		return err
	}
	if !ok {
		return notFound("会话不存在或无权访问")
	}
	return nil
}

func (s *Service) ListKnowledgeBases(ctx context.Context, principal auth.Principal, keyword string) ([]knowledge.ChatKnowledgeBase, error) {
	actor, err := s.ActorFromPrincipal(ctx, principal)
	if err != nil {
		return nil, err
	}
	if s.knowledge == nil {
		return nil, notReady("知识库服务不可用")
	}
	return s.knowledge.ListChatKnowledgeBases(ctx, actor, keyword)
}

func (s *Service) ListSessions(ctx context.Context, principal auth.Principal, page, pageSize int) ([]weknora.ChatSession, int, error) {
	if s.owners == nil {
		return nil, 0, notReady("会话归属存储不可用")
	}
	// Platform ownership is the source of truth for list visibility.
	// Still require a valid terminal-user token so the caller is ready for subsequent ops.
	if _, _, err := s.withUser(ctx, principal); err != nil {
		return nil, 0, err
	}
	owners, total, err := s.owners.ListByUser(ctx, principal.UserID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	items := make([]weknora.ChatSession, 0, len(owners))
	for _, owner := range owners {
		items = append(items, weknora.ChatSession{
			ID:        owner.SessionID,
			Title:     owner.Title,
			CreatedAt: owner.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt: owner.UpdatedAt.UTC().Format(time.RFC3339),
		})
	}
	return items, total, nil
}

func (s *Service) CreateSession(ctx context.Context, principal auth.Principal, title string) (weknora.ChatSession, error) {
	if s.owners == nil {
		return weknora.ChatSession{}, notReady("会话归属存储不可用")
	}
	ctx, identity, err := s.withUser(ctx, principal)
	if err != nil {
		return weknora.ChatSession{}, err
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = "新对话"
	}
	created, err := s.remote.CreateSession(ctx, title, "")
	if err != nil {
		return weknora.ChatSession{}, mapRemoteErr(err)
	}
	now := time.Now().UTC()
	owner := SessionOwner{
		SessionID:       created.ID,
		UserID:          principal.UserID,
		TenantID:        principal.TenantID,
		WeKnoraTenantID: identity.WeKnoraTenantID,
		Title:           created.Title,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if owner.Title == "" {
		owner.Title = title
	}
	if err := s.owners.Insert(ctx, owner); err != nil {
		if s.log != nil {
			s.log.Error("chat session ownership insert failed; compensating remote delete",
				slog.String("component", "chat"),
				slog.String("session_id", created.ID),
				slog.String("user_id", principal.UserID),
				slog.String("error", err.Error()),
			)
		}
		if delErr := s.remote.DeleteSession(ctx, created.ID); delErr != nil && s.log != nil {
			s.log.Error("chat session compensate delete failed",
				slog.String("component", "chat"),
				slog.String("session_id", created.ID),
				slog.String("user_id", principal.UserID),
				slog.String("error", delErr.Error()),
			)
		}
		return weknora.ChatSession{}, err
	}
	if created.Title == "" {
		created.Title = owner.Title
	}
	return created, nil
}

func (s *Service) RenameSession(ctx context.Context, principal auth.Principal, sessionID, title string) (weknora.ChatSession, error) {
	sessionID = strings.TrimSpace(sessionID)
	title = strings.TrimSpace(title)
	if sessionID == "" {
		return weknora.ChatSession{}, invalid("会话标识不合法")
	}
	if title == "" {
		return weknora.ChatSession{}, invalid("请填写会话标题")
	}
	if err := s.assertOwned(ctx, principal, sessionID); err != nil {
		return weknora.ChatSession{}, err
	}
	ctx, _, err := s.withUser(ctx, principal)
	if err != nil {
		return weknora.ChatSession{}, err
	}
	updated, err := s.remote.UpdateSession(ctx, sessionID, title, "")
	if err != nil {
		return weknora.ChatSession{}, mapRemoteErr(err)
	}
	if err := s.owners.UpdateTitle(ctx, principal.UserID, sessionID, title); err != nil && !errors.Is(err, ErrNotFound) {
		return weknora.ChatSession{}, err
	}
	updated.Title = title
	return updated, nil
}

func (s *Service) DeleteSession(ctx context.Context, principal auth.Principal, sessionID string) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return invalid("会话标识不合法")
	}
	if err := s.assertOwned(ctx, principal, sessionID); err != nil {
		return err
	}
	ctx, _, err := s.withUser(ctx, principal)
	if err != nil {
		return err
	}
	if err := s.remote.DeleteSession(ctx, sessionID); err != nil {
		mapped := mapRemoteErr(err)
		// Remote already gone: still drop ownership so the id cannot be reused against this user.
		if !errors.Is(mapped, ErrNotFound) {
			return mapped
		}
	}
	if err := s.owners.Delete(ctx, principal.UserID, sessionID); err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}
	return nil
}

func (s *Service) LoadMessages(ctx context.Context, principal auth.Principal, sessionID string, limit int, before *time.Time) ([]weknora.ChatMessage, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, invalid("会话标识不合法")
	}
	if err := s.assertOwned(ctx, principal, sessionID); err != nil {
		return nil, err
	}
	ctx, _, err := s.withUser(ctx, principal)
	if err != nil {
		return nil, err
	}
	items, err := s.remote.LoadMessages(ctx, sessionID, limit, before)
	return items, mapRemoteErr(err)
}

func (s *Service) Stop(ctx context.Context, principal auth.Principal, sessionID, messageID string) error {
	sessionID = strings.TrimSpace(sessionID)
	messageID = strings.TrimSpace(messageID)
	if sessionID == "" {
		return invalid("会话标识不合法")
	}
	if messageID == "" {
		return invalid("缺少 message_id，无法停止生成")
	}
	if err := s.assertOwned(ctx, principal, sessionID); err != nil {
		return err
	}
	ctx, _, err := s.withUser(ctx, principal)
	if err != nil {
		return err
	}
	return mapRemoteErr(s.remote.StopSession(ctx, sessionID, messageID))
}

// Ask opens an SSE stream for knowledge Q&A. Caller must close resp.Body.
func (s *Service) Ask(ctx context.Context, principal auth.Principal, sessionID, query string, platformKBIDs []string) (*http.Response, error) {
	sessionID = strings.TrimSpace(sessionID)
	query = strings.TrimSpace(query)
	if sessionID == "" {
		return nil, invalid("会话标识不合法")
	}
	if query == "" {
		return nil, invalid("请输入问题")
	}
	if err := s.assertOwned(ctx, principal, sessionID); err != nil {
		return nil, err
	}
	actor, err := s.ActorFromPrincipal(ctx, principal)
	if err != nil {
		return nil, err
	}
	if s.knowledge == nil {
		return nil, notReady("知识库服务不可用")
	}
	remoteIDs, err := s.knowledge.ResolveRemoteKnowledgeBaseIDs(ctx, actor, platformKBIDs)
	if err != nil {
		return nil, err
	}
	ctx, _, err = s.withUser(ctx, principal)
	if err != nil {
		return nil, err
	}
	resp, err := s.remote.OpenKnowledgeChatStream(ctx, sessionID, weknora.KnowledgeQARequest{
		Query:            query,
		KnowledgeBaseIDs: remoteIDs,
		AgentEnabled:     false,
	})
	if err != nil {
		return nil, mapRemoteErr(err)
	}
	return resp, nil
}

func mapRemoteErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, weknora.ErrNotFound) {
		return notFound("会话不存在或无权访问")
	}
	var apiErr *weknora.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case http.StatusNotFound:
			return notFound("会话不存在或无权访问")
		case http.StatusForbidden:
			return notFound("会话不存在或无权访问")
		case http.StatusUnauthorized:
			return &Error{Kind: ErrUnauthorized, Message: "WeKnora 认证失败"}
		}
	}
	return err
}

// Ready reports whether chat can run.
func (s *Service) Ready() error {
	if s == nil || s.remote == nil {
		return notReady("WeKnora 未配置")
	}
	if s.tokens == nil {
		return notReady("未配置 WEKNORA_EXTERNAL_USER_HMAC")
	}
	if s.knowledge == nil {
		return notReady("知识库服务不可用")
	}
	if s.owners == nil {
		return notReady("会话归属存储不可用")
	}
	return nil
}
