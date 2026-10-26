package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"qingzhi/enterprise-api/internal/auth"
	"qingzhi/enterprise-api/internal/response"
	"qingzhi/enterprise-api/internal/security"
	"qingzhi/enterprise-api/internal/sso"
)

// SSO serves identity federation administration and login APIs.
type SSO struct {
	Logger   *slog.Logger
	Database Pinger
	Service  *sso.Service
	Security *security.Service
	Tokens   *auth.TokenIssuer
	Timeout  time.Duration
}

// Register mounts SSO management routes.
func (h SSO) Register(group *gin.RouterGroup, enforce bool) {
	perm := func(permission string) gin.HandlerFunc {
		if !enforce {
			return func(c *gin.Context) { c.Next() }
		}
		return auth.RequirePermission(permission)
	}

	group.GET("/sso/overview", perm("system:sso:overview"), h.overview)
	group.GET("/sso/settings", perm("system:sso:overview"), h.getSettings)
	group.PUT("/sso/settings", perm("system:sso:provider:manage"), h.updateSettings)

	group.GET("/sso/providers", perm("system:sso:provider:query"), h.listProviders)
	group.POST("/sso/providers", perm("system:sso:provider:manage"), h.createProvider)
	group.GET("/sso/providers/:id", perm("system:sso:provider:query"), h.getProvider)
	group.PUT("/sso/providers/:id", perm("system:sso:provider:manage"), h.updateProvider)
	group.PATCH("/sso/providers/:id/status", perm("system:sso:provider:manage"), h.setProviderStatus)

	group.GET("/sso/oauth-clients", perm("system:sso:oauth-client:query"), h.listOAuthClients)
	group.POST("/sso/oauth-clients", perm("system:sso:oauth-client:manage"), h.createOAuthClient)
	group.GET("/sso/oauth-clients/:id", perm("system:sso:oauth-client:query"), h.getOAuthClient)
	group.PUT("/sso/oauth-clients/:id", perm("system:sso:oauth-client:manage"), h.updateOAuthClient)
	group.POST("/sso/oauth-clients/:id/rotate-secret", perm("system:sso:oauth-client:manage"), h.rotateOAuthSecret)
	group.PATCH("/sso/oauth-clients/:id/status", perm("system:sso:oauth-client:manage"), h.setOAuthClientStatus)

	group.GET("/sso/saml-sps", perm("system:sso:saml-sp:query"), h.listSamlSPs)
	group.POST("/sso/saml-sps", perm("system:sso:saml-sp:manage"), h.createSamlSP)
	group.GET("/sso/saml-sps/:id", perm("system:sso:saml-sp:query"), h.getSamlSP)
	group.PUT("/sso/saml-sps/:id", perm("system:sso:saml-sp:manage"), h.updateSamlSP)
	group.PATCH("/sso/saml-sps/:id/status", perm("system:sso:saml-sp:manage"), h.setSamlSPStatus)

	group.GET("/sso/identity-links", perm("system:sso:identity-link:query"), h.listIdentityLinks)
	group.DELETE("/sso/identity-links/:id", perm("system:sso:identity-link:manage"), h.deleteIdentityLink)

	group.GET("/sso/auth-events", perm("system:sso:auth-event:query"), h.listAuthEvents)
}

func (h SSO) overview(c *gin.Context) {
	if !h.available(c) {
		return
	}
	item, err := h.Service.Overview(c.Request.Context())
	h.write(c, item, err)
}

func (h SSO) getSettings(c *gin.Context) {
	if !h.available(c) {
		return
	}
	item, err := h.Service.GetSettings(c.Request.Context())
	h.write(c, item, err)
}

func (h SSO) updateSettings(c *gin.Context) {
	if !h.available(c) {
		return
	}
	var body sso.Settings
	if !response.BindJSON(c, &body) {
		return
	}
	item, err := h.Service.UpdateSettings(c.Request.Context(), body, ssoActor(c))
	h.write(c, item, err)
}

func (h SSO) listProviders(c *gin.Context) {
	if !h.available(c) {
		return
	}
	query, ok := ssoListQuery(c)
	if !ok {
		return
	}
	page, err := h.Service.ListProviders(c.Request.Context(), query)
	h.write(c, page, err)
}

func (h SSO) createProvider(c *gin.Context) {
	if !h.available(c) {
		return
	}
	var body sso.ProviderInput
	if !response.BindJSON(c, &body) {
		return
	}
	item, err := h.Service.CreateProvider(c.Request.Context(), body, ssoActor(c))
	h.write(c, item, err)
}

func (h SSO) getProvider(c *gin.Context) {
	if !h.available(c) {
		return
	}
	item, err := h.Service.GetProvider(c.Request.Context(), c.Param("id"))
	h.write(c, item, err)
}

func (h SSO) updateProvider(c *gin.Context) {
	if !h.available(c) {
		return
	}
	var body sso.ProviderInput
	if !response.BindJSON(c, &body) {
		return
	}
	item, err := h.Service.UpdateProvider(c.Request.Context(), c.Param("id"), body, ssoActor(c))
	h.write(c, item, err)
}

func (h SSO) setProviderStatus(c *gin.Context) {
	if !h.available(c) {
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	err := h.Service.SetProviderStatus(c.Request.Context(), c.Param("id"), body.Status, ssoActor(c))
	h.write(c, gin.H{"ok": true}, err)
}

func (h SSO) listOAuthClients(c *gin.Context) {
	if !h.available(c) {
		return
	}
	query, ok := ssoListQuery(c)
	if !ok {
		return
	}
	page, err := h.Service.ListOAuthClients(c.Request.Context(), query)
	h.write(c, page, err)
}

func (h SSO) createOAuthClient(c *gin.Context) {
	if !h.available(c) {
		return
	}
	var body sso.OAuthClientInput
	if !response.BindJSON(c, &body) {
		return
	}
	item, err := h.Service.CreateOAuthClient(c.Request.Context(), body, ssoActor(c))
	h.write(c, item, err)
}

func (h SSO) getOAuthClient(c *gin.Context) {
	if !h.available(c) {
		return
	}
	item, err := h.Service.GetOAuthClient(c.Request.Context(), c.Param("id"))
	h.write(c, item, err)
}

func (h SSO) updateOAuthClient(c *gin.Context) {
	if !h.available(c) {
		return
	}
	var body sso.OAuthClientInput
	if !response.BindJSON(c, &body) {
		return
	}
	item, err := h.Service.UpdateOAuthClient(c.Request.Context(), c.Param("id"), body, ssoActor(c))
	h.write(c, item, err)
}

func (h SSO) rotateOAuthSecret(c *gin.Context) {
	if !h.available(c) {
		return
	}
	item, err := h.Service.RotateOAuthClientSecret(c.Request.Context(), c.Param("id"), ssoActor(c))
	h.write(c, item, err)
}

func (h SSO) setOAuthClientStatus(c *gin.Context) {
	if !h.available(c) {
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	err := h.Service.SetOAuthClientStatus(c.Request.Context(), c.Param("id"), body.Status, ssoActor(c))
	h.write(c, gin.H{"ok": true}, err)
}

func (h SSO) listSamlSPs(c *gin.Context) {
	if !h.available(c) {
		return
	}
	query, ok := ssoListQuery(c)
	if !ok {
		return
	}
	page, err := h.Service.ListSamlSPs(c.Request.Context(), query)
	h.write(c, page, err)
}

func (h SSO) createSamlSP(c *gin.Context) {
	if !h.available(c) {
		return
	}
	var body sso.SamlSPInput
	if !response.BindJSON(c, &body) {
		return
	}
	item, err := h.Service.CreateSamlSP(c.Request.Context(), body, ssoActor(c))
	h.write(c, item, err)
}

func (h SSO) getSamlSP(c *gin.Context) {
	if !h.available(c) {
		return
	}
	item, err := h.Service.GetSamlSP(c.Request.Context(), c.Param("id"))
	h.write(c, item, err)
}

func (h SSO) updateSamlSP(c *gin.Context) {
	if !h.available(c) {
		return
	}
	var body sso.SamlSPInput
	if !response.BindJSON(c, &body) {
		return
	}
	item, err := h.Service.UpdateSamlSP(c.Request.Context(), c.Param("id"), body, ssoActor(c))
	h.write(c, item, err)
}

func (h SSO) setSamlSPStatus(c *gin.Context) {
	if !h.available(c) {
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if !response.BindJSON(c, &body) {
		return
	}
	err := h.Service.SetSamlSPStatus(c.Request.Context(), c.Param("id"), body.Status, ssoActor(c))
	h.write(c, gin.H{"ok": true}, err)
}

func (h SSO) listIdentityLinks(c *gin.Context) {
	if !h.available(c) {
		return
	}
	query, ok := ssoListQuery(c)
	if !ok {
		return
	}
	page, err := h.Service.ListIdentityLinks(c.Request.Context(), query)
	h.write(c, page, err)
}

func (h SSO) deleteIdentityLink(c *gin.Context) {
	if !h.available(c) {
		return
	}
	err := h.Service.DeleteIdentityLink(c.Request.Context(), c.Param("id"), ssoActor(c))
	h.write(c, gin.H{"ok": true}, err)
}

func (h SSO) listAuthEvents(c *gin.Context) {
	if !h.available(c) {
		return
	}
	query, ok := ssoListQuery(c)
	if !ok {
		return
	}
	page, err := h.Service.ListAuthEvents(c.Request.Context(), query)
	h.write(c, page, err)
}

func ssoActor(c *gin.Context) string {
	principal, ok := auth.GetPrincipal(c)
	if !ok {
		return ""
	}
	return principal.Username
}

func (h SSO) available(c *gin.Context) bool {
	if h.Service == nil {
		response.Fail(c, http.StatusServiceUnavailable, response.CodeNotReady, "服务暂不可用", nil)
		return false
	}
	if h.Database == nil {
		return true
	}
	timeout := h.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()
	if err := h.Database.Ping(ctx); err != nil {
		if h.Logger != nil {
			h.Logger.Warn("sso database unavailable", slog.String("component", "app"), slog.String("error", err.Error()))
		}
		response.Fail(c, http.StatusServiceUnavailable, response.CodeNotReady, "服务暂不可用", nil)
		return false
	}
	return true
}

func (h SSO) write(c *gin.Context, data any, err error) {
	if err == nil {
		response.OK(c, data)
		return
	}
	var domain *sso.Error
	if errors.As(err, &domain) {
		switch {
		case errors.Is(domain, sso.ErrNotFound):
			response.Fail(c, http.StatusNotFound, response.CodeNotFound, domain.Message, nil)
		case errors.Is(domain, sso.ErrConflict):
			response.Fail(c, http.StatusConflict, response.CodeInvalidParams, domain.Message, nil)
		case errors.Is(domain, sso.ErrNotReady):
			response.Fail(c, http.StatusServiceUnavailable, response.CodeNotReady, domain.Message, nil)
		default:
			response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, domain.Message, nil)
		}
		return
	}
	if h.Logger != nil {
		h.Logger.Error("sso request failed", slog.String("component", "app"), slog.String("error", err.Error()))
	}
	response.Fail(c, http.StatusInternalServerError, response.CodeInternal, "服务器内部错误", nil)
}

func ssoListQuery(c *gin.Context) (sso.ListQuery, bool) {
	page, err := ssoOptionalInt(c.Query("page"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "分页参数不合法", nil)
		return sso.ListQuery{}, false
	}
	pageSize, err := ssoOptionalInt(c.Query("page_size"))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "分页参数不合法", nil)
		return sso.ListQuery{}, false
	}
	keyword := strings.TrimSpace(c.Query("keyword"))
	if len(keyword) > 64 {
		response.Fail(c, http.StatusBadRequest, response.CodeInvalidParams, "请求参数不合法", nil)
		return sso.ListQuery{}, false
	}
	return sso.ListQuery{
		Page:     page,
		PageSize: pageSize,
		Keyword:  keyword,
		Protocol: strings.TrimSpace(c.Query("protocol")),
		Status:   strings.TrimSpace(c.Query("status")),
		Result:   strings.TrimSpace(c.Query("result")),
	}, true
}

func ssoOptionalInt(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	return strconv.Atoi(raw)
}
