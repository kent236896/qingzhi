package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"qingzhi/enterprise-api/internal/identity"
	"qingzhi/enterprise-api/internal/response"
	"qingzhi/enterprise-api/internal/security"
	"qingzhi/enterprise-api/internal/sso"
)

// RegisterPublic mounts unauthenticated SSO login routes.
func (h SSO) RegisterPublic(group *gin.RouterGroup) {
	group.GET("/sso/login-options", h.loginOptions)
	group.GET("/sso/oidc/start", h.startOIDC)
	group.GET("/sso/oauth2/start", h.startOAuth2)
	group.GET("/sso/oidc/callback", h.callbackOIDC)
	group.GET("/sso/oauth2/callback", h.callbackOAuth2)
	group.POST("/sso/oidc/exchange", h.exchangeTicket)
}

func (h SSO) loginOptions(c *gin.Context) {
	if !h.available(c) {
		return
	}
	items, settings, err := h.Service.LoginOptions(c.Request.Context())
	if err != nil {
		h.write(c, nil, err)
		return
	}
	response.OK(c, gin.H{
		"local_login_enabled": settings.LocalLoginEnabled,
		"providers":           items,
	})
}

func (h SSO) startOIDC(c *gin.Context) {
	h.startExternal(c)
}

func (h SSO) startOAuth2(c *gin.Context) {
	h.startExternal(c)
}

func (h SSO) startExternal(c *gin.Context) {
	if !h.available(c) {
		return
	}
	providerID := strings.TrimSpace(c.Query("provider_id"))
	if providerID == "" {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "缺少提供方", nil)
		return
	}
	authURL, err := h.Service.StartExternalLogin(
		c.Request.Context(),
		providerID,
		c.Query("redirect"),
		c.Query("tenant_id"),
	)
	if err != nil {
		h.write(c, nil, err)
		return
	}
	c.Redirect(http.StatusFound, authURL)
}

func (h SSO) callbackOIDC(c *gin.Context) {
	h.callbackExternal(c)
}

func (h SSO) callbackOAuth2(c *gin.Context) {
	h.callbackExternal(c)
}

func (h SSO) callbackExternal(c *gin.Context) {
	if !h.available(c) {
		return
	}
	if errParam := strings.TrimSpace(c.Query("error")); errParam != "" {
		desc := strings.TrimSpace(c.Query("error_description"))
		msg := "外部登录失败"
		if desc != "" {
			msg = desc
		}
		h.redirectFrontend(c, "/", "", msg)
		return
	}
	ticket, redirectPath, err := h.Service.CompleteExternalLogin(
		c.Request.Context(),
		c.Query("code"),
		c.Query("state"),
		c.ClientIP(),
	)
	if err != nil {
		msg := "外部登录失败"
		var domain *sso.Error
		if errors.As(err, &domain) && domain.Message != "" {
			msg = domain.Message
		}
		h.redirectFrontend(c, redirectPath, "", msg)
		return
	}
	h.redirectFrontend(c, redirectPath, ticket, "")
}

func (h SSO) exchangeTicket(c *gin.Context) {
	if !h.available(c) {
		return
	}
	if h.Tokens == nil {
		response.Fail(c, http.StatusServiceUnavailable, response.CodeNotReady, "服务暂不可用", nil)
		return
	}
	var body struct {
		Ticket string `json:"ticket"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	result, err := h.Service.ExchangeLoginTicket(c.Request.Context(), body.Ticket, c.ClientIP())
	if err != nil {
		h.writeLogin(c, err)
		return
	}
	h.recordSSOLogin(c, result, security.LoginSuccess, "外部身份登录成功")
	token, expires, err := h.Tokens.Issue(result.User.ID, result.User.Username, result.TenantID, time.Now().UTC())
	if err != nil {
		if h.Logger != nil {
			h.Logger.Error("issue federated token failed", slog.String("component", "app"), slog.String("error", err.Error()))
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

func (h SSO) redirectFrontend(c *gin.Context, path, ticket, errMsg string) {
	target, err := h.Service.FrontendRedirectURL(c.Request.Context(), path, ticket, errMsg)
	if err != nil {
		response.Fail(c, http.StatusServiceUnavailable, response.CodeNotReady, "前端回调地址未配置", nil)
		return
	}
	// Prevent open redirect amplification: FrontendRedirectURL already validates base URL.
	if _, parseErr := url.Parse(target); parseErr != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "回调地址不合法", nil)
		return
	}
	c.Redirect(http.StatusFound, target)
}

func (h SSO) writeLogin(c *gin.Context, err error) {
	var domain *sso.Error
	if errors.As(err, &domain) {
		response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, domain.Message, nil)
		return
	}
	var idErr *identity.Error
	if errors.As(err, &idErr) {
		response.Fail(c, http.StatusUnauthorized, response.CodeUnauthorized, idErr.Message, nil)
		return
	}
	h.write(c, nil, err)
}

func (h SSO) recordSSOLogin(c *gin.Context, result identity.LoginResult, code int, message string) {
	if h.Security == nil {
		return
	}
	_ = h.Security.RecordLogin(c.Request.Context(), security.LoginLog{
		Username:   result.User.Username,
		UserID:     result.User.ID,
		TenantID:   result.TenantID,
		Result:     code,
		Message:    message,
		IP:         c.ClientIP(),
		UserAgent:  c.Request.UserAgent(),
		CreateTime: time.Now().UTC(),
	})
}
