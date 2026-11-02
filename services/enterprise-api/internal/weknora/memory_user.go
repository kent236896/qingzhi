package weknora

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"qingzhi/enterprise-api/internal/auth"
)

// PersonalMemoryRemote is the WeKnora personal memory surface (requires External User Token in ctx).
type PersonalMemoryRemote interface {
	GetPersonalMemorySettings(ctx context.Context) (PersonalMemorySettings, error)
	UpdatePersonalMemorySettings(ctx context.Context, userEnabled bool) (PersonalMemorySettings, error)
	ListPersonalMemoryItems(ctx context.Context, page, pageSize int, kind, status string) (PersonalMemoryList, error)
	CreatePersonalMemoryItem(ctx context.Context, kind, content string, importance int) (PersonalMemoryItem, error)
	UpdatePersonalMemoryItem(ctx context.Context, id, kind, content string, importance int) (PersonalMemoryItem, error)
	DeletePersonalMemoryItem(ctx context.Context, id string) error
	ConfirmPersonalMemoryItem(ctx context.Context, id string) error
	RejectPersonalMemoryItem(ctx context.Context, id string) error
}

func (s *MemoryService) withCaller(ctx context.Context, principal auth.Principal) (context.Context, error) {
	if !PersonalMemoryIsolation().Ready {
		return ctx, fmt.Errorf("个人记忆能力未开放：%s", PersonalMemoryIsolation().Reason)
	}
	token, _, _, err := s.IssueExternalUserToken(ctx, principal)
	if err != nil {
		return ctx, err
	}
	return WithExternalUserToken(ctx, token), nil
}

func (s *MemoryService) personalRemote() (PersonalMemoryRemote, error) {
	if s == nil || s.remote == nil {
		return nil, fmt.Errorf("WeKnora 未配置")
	}
	p, ok := s.remote.(PersonalMemoryRemote)
	if !ok || p == nil {
		return nil, fmt.Errorf("当前 WeKnora 客户端不支持个人记忆接口")
	}
	return p, nil
}

// GetPersonalSettings returns settings for the authenticated platform user.
func (s *MemoryService) GetPersonalSettings(ctx context.Context, principal auth.Principal) (PersonalMemorySettings, error) {
	remote, err := s.personalRemote()
	if err != nil {
		return PersonalMemorySettings{}, err
	}
	ctx, err = s.withCaller(ctx, principal)
	if err != nil {
		return PersonalMemorySettings{}, err
	}
	return remote.GetPersonalMemorySettings(ctx)
}

// UpdatePersonalSettings updates only the caller's personal switch.
func (s *MemoryService) UpdatePersonalSettings(ctx context.Context, principal auth.Principal, userEnabled bool) (PersonalMemorySettings, error) {
	remote, err := s.personalRemote()
	if err != nil {
		return PersonalMemorySettings{}, err
	}
	ctx, err = s.withCaller(ctx, principal)
	if err != nil {
		return PersonalMemorySettings{}, err
	}
	return remote.UpdatePersonalMemorySettings(ctx, userEnabled)
}

// ListPersonalItems lists the caller's memories.
func (s *MemoryService) ListPersonalItems(ctx context.Context, principal auth.Principal, page, pageSize int, kind, status string) (PersonalMemoryList, error) {
	remote, err := s.personalRemote()
	if err != nil {
		return PersonalMemoryList{}, err
	}
	ctx, err = s.withCaller(ctx, principal)
	if err != nil {
		return PersonalMemoryList{}, err
	}
	return remote.ListPersonalMemoryItems(ctx, page, pageSize, kind, status)
}

// CreatePersonalItem creates a memory for the caller.
func (s *MemoryService) CreatePersonalItem(ctx context.Context, principal auth.Principal, kind, content string, importance int) (PersonalMemoryItem, error) {
	kind, content, importance, err := validatePersonalItem(kind, content, importance)
	if err != nil {
		return PersonalMemoryItem{}, err
	}
	remote, err := s.personalRemote()
	if err != nil {
		return PersonalMemoryItem{}, err
	}
	ctx, err = s.withCaller(ctx, principal)
	if err != nil {
		return PersonalMemoryItem{}, err
	}
	return remote.CreatePersonalMemoryItem(ctx, kind, content, importance)
}

// UpdatePersonalItem updates a memory owned by the caller (WeKnora enforces ownership).
func (s *MemoryService) UpdatePersonalItem(ctx context.Context, principal auth.Principal, id, kind, content string, importance int) (PersonalMemoryItem, error) {
	if strings.TrimSpace(id) == "" {
		return PersonalMemoryItem{}, fmt.Errorf("缺少记忆标识")
	}
	kind, content, importance, err := validatePersonalItem(kind, content, importance)
	if err != nil {
		return PersonalMemoryItem{}, err
	}
	remote, err := s.personalRemote()
	if err != nil {
		return PersonalMemoryItem{}, err
	}
	ctx, err = s.withCaller(ctx, principal)
	if err != nil {
		return PersonalMemoryItem{}, err
	}
	return remote.UpdatePersonalMemoryItem(ctx, id, kind, content, importance)
}

// DeletePersonalItem deletes a memory owned by the caller.
func (s *MemoryService) DeletePersonalItem(ctx context.Context, principal auth.Principal, id string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("缺少记忆标识")
	}
	remote, err := s.personalRemote()
	if err != nil {
		return err
	}
	ctx, err = s.withCaller(ctx, principal)
	if err != nil {
		return err
	}
	return remote.DeletePersonalMemoryItem(ctx, id)
}

// ConfirmPersonalItem confirms a pending memory.
func (s *MemoryService) ConfirmPersonalItem(ctx context.Context, principal auth.Principal, id string) error {
	remote, err := s.personalRemote()
	if err != nil {
		return err
	}
	ctx, err = s.withCaller(ctx, principal)
	if err != nil {
		return err
	}
	return remote.ConfirmPersonalMemoryItem(ctx, id)
}

// RejectPersonalItem rejects a pending memory.
func (s *MemoryService) RejectPersonalItem(ctx context.Context, principal auth.Principal, id string) error {
	remote, err := s.personalRemote()
	if err != nil {
		return err
	}
	ctx, err = s.withCaller(ctx, principal)
	if err != nil {
		return err
	}
	return remote.RejectPersonalMemoryItem(ctx, id)
}

var allowedMemoryKinds = map[string]struct{}{
	"profile": {}, "preference": {}, "fact": {}, "task": {}, "interest": {},
}

func validatePersonalItem(kind, content string, importance int) (string, string, int, error) {
	kind = strings.TrimSpace(strings.ToLower(kind))
	content = strings.TrimSpace(content)
	if kind == "" {
		return "", "", 0, fmt.Errorf("请选择记忆类型")
	}
	if _, ok := allowedMemoryKinds[kind]; !ok {
		return "", "", 0, fmt.Errorf("不支持的记忆类型")
	}
	if content == "" {
		return "", "", 0, fmt.Errorf("请填写记忆内容")
	}
	if utf8.RuneCountInString(content) > 300 {
		return "", "", 0, fmt.Errorf("记忆内容不能超过 300 个字符")
	}
	if importance < 1 {
		importance = 1
	}
	if importance > 5 {
		importance = 5
	}
	return kind, content, importance, nil
}

// MapPersonalMemoryError converts upstream errors to HTTP-ish status for handlers.
func MapPersonalMemoryError(err error) (int, string) {
	if err == nil {
		return http.StatusOK, ""
	}
	var apiErr *MemoryAPIError
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case http.StatusUnauthorized:
			return http.StatusUnauthorized, "WeKnora 认证失败，请确认已配置 External User HMAC"
		case http.StatusForbidden:
			return http.StatusForbidden, "无权操作该记忆"
		case http.StatusNotFound:
			return http.StatusNotFound, "记忆不存在或无权访问"
		case http.StatusConflict:
			return http.StatusConflict, apiErr.Message
		case http.StatusBadRequest:
			return http.StatusBadRequest, apiErr.Message
		default:
			if apiErr.Message != "" {
				return http.StatusBadGateway, apiErr.Message
			}
		}
	}
	msg := err.Error()
	if strings.Contains(msg, "未开放") || strings.Contains(msg, "未配置 WEKNORA_EXTERNAL_USER_HMAC") {
		return http.StatusServiceUnavailable, msg
	}
	if strings.Contains(msg, "请") || strings.Contains(msg, "不能") || strings.Contains(msg, "缺少") || strings.Contains(msg, "不支持") {
		return http.StatusBadRequest, msg
	}
	return http.StatusBadGateway, "记忆服务暂时不可用"
}
