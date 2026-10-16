package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"qingzhi/enterprise-api/internal/auth"
	"qingzhi/enterprise-api/internal/identity"
	"qingzhi/enterprise-api/internal/response"
	"qingzhi/enterprise-api/internal/security"
)

// Auth serves login and current-session endpoints.
type Auth struct {
	Logger   *slog.Logger
	Database Pinger
	Service  *identity.Service
	Security *security.Service
	Tokens   *auth.TokenIssuer
	Timeout  time.Duration
}

// RegisterPublic mounts unauthenticated auth routes.
func (h Auth) RegisterPublic(group *gin.RouterGroup) {
	group.POST("/auth/login", h.login)
}

// RegisterProtected mounts authenticated auth routes.
func (h Auth) RegisterProtected(group *gin.RouterGroup) {
	group.GET("/auth/me", h.me)
	group.GET("/auth/menus", h.menus)
}

func (h Auth) login(c *gin.Context) {
	if !h.available(c) {
		return
	}
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
		TenantID string `json:"tenant_id"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	if h.Security != nil {
		if err := h.Security.CheckLoginAllowed(c.Request.Context(), body.Username); err != nil {
			h.recordLogin(c, body.Username, "", "", security.LoginLocked, err.Error())
			h.writeAuth(c, nil, err)
			return
		}
	}
	result, err := h.Service.Authenticate(c.Request.Context(), identity.LoginInput{
		Username: body.Username,
		Password: body.Password,
		TenantID: body.TenantID,
		LoginIP:  c.ClientIP(),
	})
	if err != nil {
		h.recordLogin(c, body.Username, "", body.TenantID, security.LoginCredentialFailure, loginMessage(err))
		h.writeAuth(c, nil, err)
		return
	}
	h.recordLogin(c, result.User.Username, result.User.ID, result.TenantID, security.LoginSuccess, "登录成功")
	token, expires, err := h.Tokens.Issue(result.User.ID, result.User.Username, result.TenantID, time.Now().UTC())
	if err != nil {
		if h.Logger != nil {
			h.Logger.Error("issue token failed", slog.String("component", "app"), slog.String("error", err.Error()))
		}
		response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误", nil)
		return
	}
	response.OK(c, map[string]any{
		"access_token": token,
		"token_type":   "Bearer",
		"expires_at":   expires.UTC().Format(time.RFC3339),
		"user": map[string]any{
			"id":       result.User.ID,
			"username": result.User.Username,
			"nickname": result.User.Nickname,
			"avatar":   result.User.Avatar,
		},
		"tenant_id":   result.TenantID,
		"permissions": result.Access.Permissions,
		"data_scope":  result.Access.Access.Scope,
	})
}

func (h Auth) me(c *gin.Context) {
	principal, ok := auth.GetPrincipal(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录或令牌无效", nil)
		return
	}
	permissions := make([]string, 0, len(principal.Permissions))
	for item := range principal.Permissions {
		permissions = append(permissions, item)
	}
	response.OK(c, map[string]any{
		"user": map[string]any{
			"id":       principal.UserID,
			"username": principal.Username,
			"nickname": principal.Nickname,
		},
		"tenant_id":   principal.TenantID,
		"permissions": permissions,
		"data_scope":  principal.Access.Scope,
	})
}

func (h Auth) menus(c *gin.Context) {
	if !h.available(c) {
		return
	}
	principal, ok := auth.GetPrincipal(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, "未登录或令牌无效", nil)
		return
	}
	items, err := h.Service.ListAuthorizedMenus(c.Request.Context(), principal.UserID, principal.TenantID)
	h.writeAuth(c, items, err)
}

func (h Auth) recordLogin(c *gin.Context, username, userID, tenantID string, result int, message string) {
	if h.Security == nil {
		return
	}
	agent := ""
	if c.Request != nil {
		agent = c.Request.UserAgent()
	}
	if err := h.Security.RecordLogin(c.Request.Context(), security.LoginLog{
		Username: username, UserID: userID, TenantID: tenantID, Result: result, Message: message,
		IP: c.ClientIP(), UserAgent: agent,
	}); err != nil && h.Logger != nil {
		h.Logger.Warn("login log failed", slog.String("component", "app"), slog.String("error", err.Error()))
	}
}

func loginMessage(err error) string {
	var domain *identity.Error
	if errors.As(err, &domain) && domain.Message != "" {
		return domain.Message
	}
	return "登录失败"
}

func (h Auth) available(c *gin.Context) bool {
	if h.Service == nil || h.Database == nil || h.Tokens == nil {
		response.Fail(c, http.StatusServiceUnavailable, response.CodeNotReady, "服务暂不可用", nil)
		return false
	}
	return true
}

func (h Auth) writeAuth(c *gin.Context, data any, err error) {
	if err == nil {
		response.OK(c, data)
		return
	}
	var domain *identity.Error
	var locked *security.Error
	if errors.As(err, &locked) && errors.Is(err, security.ErrLocked) {
		response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, locked.Message, nil)
		return
	}
	if errors.As(err, &domain) {
		switch {
		case errors.Is(err, identity.ErrUnauthorized):
			response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, domain.Message, nil)
		case errors.Is(err, identity.ErrForbidden):
			response.Fail(c, http.StatusForbidden, response.CodeForbidden, domain.Message, nil)
		case errors.Is(err, identity.ErrInvalid):
			response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, domain.Message, nil)
		case errors.Is(err, identity.ErrNotFound):
			response.Fail(c, http.StatusNotFound, response.CodeNotFound, domain.Message, nil)
		case errors.Is(err, identity.ErrConflict):
			response.Fail(c, http.StatusConflict, response.CodeConflict, domain.Message, nil)
		default:
			response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, domain.Message, nil)
		}
		return
	}
	if h.Logger != nil {
		h.Logger.Error("auth request failed", slog.String("component", "app"), slog.String("error", err.Error()))
	}
	response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误", nil)
}
