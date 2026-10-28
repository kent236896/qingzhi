package sso

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"encoding/xml"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/crewjam/saml"
)

// StartSAMLLogin begins a SAML AuthnRequest (HTTP-Redirect) against an external IdP.
func (s *Service) StartSAMLLogin(ctx context.Context, providerID, redirectPath, tenantID, frontendBase string) (string, error) {
	provider, err := s.store.GetProvider(ctx, providerID)
	if err != nil {
		return "", err
	}
	if provider.Status != StatusEnabled {
		return "", invalid("身份提供方未启用")
	}
	if provider.Protocol != ProtocolSAML {
		return "", invalid("该提供方不支持 SAML 登录")
	}
	redirectPath = sanitizeRedirectPath(redirectPath)
	tenantID = strings.TrimSpace(tenantID)
	frontendBase, err = s.normalizeFrontendBase(ctx, frontendBase)
	if err != nil {
		return "", err
	}

	state, err := randomURLString(32)
	if err != nil {
		return "", err
	}

	sp, err := s.buildSAMLServiceProvider(ctx, provider)
	if err != nil {
		return "", err
	}
	idpSSOURL := sp.GetSSOBindingLocation(saml.HTTPRedirectBinding)
	if idpSSOURL == "" {
		idpSSOURL = sp.GetSSOBindingLocation(saml.HTTPPostBinding)
	}
	if idpSSOURL == "" {
		idpSSOURL = strings.TrimSpace(provider.SSOURL)
	}
	if idpSSOURL == "" {
		return "", invalid("无法确定 IdP SSO URL，请配置 Metadata URL 或 SSO URL")
	}

	authnReq, err := sp.MakeAuthenticationRequest(idpSSOURL, saml.HTTPRedirectBinding, saml.HTTPPostBinding)
	if err != nil {
		return "", invalid("无法构造 SAML AuthnRequest")
	}

	req := AuthRequest{
		State:           state,
		Nonce:           authnReq.ID,
		CodeVerifier:    authnReq.ID,
		ProviderID:      provider.ID,
		Protocol:        ProtocolSAML,
		RedirectPath:    redirectPath,
		TenantID:        tenantID,
		FrontendBaseURL: frontendBase,
		ExpiresAt:       s.now().UTC().Add(10 * time.Minute),
	}
	if _, err := s.store.CreateAuthRequest(ctx, req); err != nil {
		return "", err
	}

	u, err := authnReq.Redirect(state, sp)
	if err != nil {
		return "", invalid("无法生成 SAML 跳转地址")
	}
	return u.String(), nil
}

// CompleteSAMLLogin finishes ACS callback, maps identity, and returns a one-time ticket.
func (s *Service) CompleteSAMLLogin(ctx context.Context, r *http.Request, loginIP string) (ticket, redirectPath, frontendBase string, err error) {
	if err := r.ParseForm(); err != nil {
		return "", "/", "", invalid("SAML 回调参数无效")
	}
	relayState := strings.TrimSpace(r.Form.Get("RelayState"))
	if relayState == "" {
		return "", "/", "", invalid("缺少 RelayState")
	}
	authReq, err := s.store.ConsumeAuthRequest(ctx, relayState)
	if err != nil {
		return "", "/", "", err
	}
	frontendBase = strings.TrimSpace(authReq.FrontendBaseURL)
	redirectPath = authReq.RedirectPath
	if authReq.Protocol != ProtocolSAML {
		return "", redirectPath, frontendBase, invalid("登录请求协议不匹配")
	}
	provider, err := s.store.GetProvider(ctx, authReq.ProviderID)
	if err != nil {
		return "", redirectPath, frontendBase, err
	}
	if provider.Status != StatusEnabled {
		return "", redirectPath, frontendBase, invalid("身份提供方未启用")
	}

	sp, err := s.buildSAMLServiceProvider(ctx, provider)
	if err != nil {
		return "", redirectPath, frontendBase, err
	}
	assertion, err := sp.ParseResponse(r, []string{authReq.Nonce, authReq.CodeVerifier})
	if err != nil {
		_ = s.store.MarkProviderAuthResult(ctx, provider.ID, false, "SAML 断言校验失败")
		_ = s.RecordEvent(ctx, AuthEvent{
			EventType:  "login.callback",
			Result:     ResultFailure,
			Protocol:   ProtocolSAML,
			ProviderID: provider.ID,
			Message:    "SAML 断言校验失败",
			IP:         loginIP,
		})
		return "", redirectPath, frontendBase, invalid("SAML 断言校验失败")
	}
	subject := ""
	if assertion.Subject != nil && assertion.Subject.NameID != nil {
		subject = strings.TrimSpace(assertion.Subject.NameID.Value)
	}
	if subject == "" {
		return "", redirectPath, frontendBase, invalid("SAML 断言缺少 NameID")
	}
	email := samlAttribute(assertion, provider.ClaimMapping, "email", "email", "mail", "Email")
	username := samlAttribute(assertion, provider.ClaimMapping, "username", "uid", "sAMAccountName", "username", "preferred_username")
	name := samlAttribute(assertion, provider.ClaimMapping, "name", "displayName", "cn", "name", "givenName")

	userID, err := s.resolveMappedUser(ctx, provider, subject, email, username, name, authReq.TenantID)
	if err != nil {
		_ = s.store.MarkProviderAuthResult(ctx, provider.ID, false, safeErr(err))
		_ = s.RecordEvent(ctx, AuthEvent{
			EventType:  "login.map",
			Result:     ResultFailure,
			Protocol:   ProtocolSAML,
			ProviderID: provider.ID,
			Message:    safeErr(err),
			IP:         loginIP,
		})
		return "", redirectPath, frontendBase, err
	}
	if s.identity == nil {
		return "", redirectPath, frontendBase, notReady("身份服务未就绪")
	}
	tenantID := authReq.TenantID
	if tenantID == "" {
		tenantID = strings.TrimSpace(s.runtime.DefaultTenantID)
	}
	login, err := s.identity.LoginFederated(ctx, userID, tenantID, loginIP)
	if err != nil {
		_ = s.store.MarkProviderAuthResult(ctx, provider.ID, false, "本地会话建立失败")
		_ = s.RecordEvent(ctx, AuthEvent{
			EventType:  "login.session",
			Result:     ResultFailure,
			Protocol:   ProtocolSAML,
			ProviderID: provider.ID,
			UserID:     userID,
			Message:    "本地会话建立失败",
			IP:         loginIP,
		})
		return "", redirectPath, frontendBase, mapIdentityErr(err)
	}

	ticketValue, err := randomURLString(32)
	if err != nil {
		return "", redirectPath, frontendBase, err
	}
	_, err = s.store.CreateLoginTicket(ctx, LoginTicket{
		Ticket:     ticketValue,
		UserID:     login.User.ID,
		TenantID:   login.TenantID,
		ProviderID: provider.ID,
		ExpiresAt:  s.now().UTC().Add(2 * time.Minute),
	})
	if err != nil {
		return "", redirectPath, frontendBase, err
	}
	_ = s.store.MarkProviderAuthResult(ctx, provider.ID, true, "")
	_ = s.RecordEvent(ctx, AuthEvent{
		EventType:  "login.callback",
		Result:     ResultSuccess,
		Protocol:   ProtocolSAML,
		ProviderID: provider.ID,
		UserID:     login.User.ID,
		Username:   login.User.Username,
		Message:    "SAML 外部登录成功",
		IP:         loginIP,
	})
	return ticketValue, redirectPath, frontendBase, nil
}

// SAMLMetadataXML returns this platform's SP metadata for a configured IdP provider.
func (s *Service) SAMLMetadataXML(ctx context.Context, providerID string) ([]byte, error) {
	provider, err := s.store.GetProvider(ctx, providerID)
	if err != nil {
		return nil, err
	}
	if provider.Protocol != ProtocolSAML {
		return nil, invalid("仅 SAML 提供方可导出 SP Metadata")
	}
	sp, err := s.buildSAMLServiceProvider(ctx, provider)
	if err != nil {
		return nil, err
	}
	return xml.MarshalIndent(sp.Metadata(), "", "  ")
}

func (s *Service) buildSAMLServiceProvider(ctx context.Context, provider Provider) (*saml.ServiceProvider, error) {
	key, cert, err := s.ensureSAMLSPKey(ctx)
	if err != nil {
		return nil, err
	}
	acsURL, err := s.resolveSAMLACSURL(provider)
	if err != nil {
		return nil, err
	}
	entityID := strings.TrimSpace(provider.SPEntityID)
	if entityID == "" {
		entityID = strings.TrimRight(acsURL.String(), "/")
	}
	metadataURL := *acsURL
	metadataURL.Path = strings.TrimSuffix(acsURL.Path, "/acs") + "/metadata"
	q := metadataURL.Query()
	q.Set("provider_id", provider.ID)
	metadataURL.RawQuery = q.Encode()

	idpMeta, err := s.loadIDPMetadata(ctx, provider)
	if err != nil {
		return nil, err
	}

	sp := &saml.ServiceProvider{
		EntityID:    entityID,
		Key:         key,
		Certificate: cert,
		MetadataURL: metadataURL,
		AcsURL:      *acsURL,
		IDPMetadata: idpMeta,
		HTTPClient:  s.client(),
	}
	if format := strings.TrimSpace(provider.NameIDFormat); format != "" {
		sp.AuthnNameIDFormat = saml.NameIDFormat(format)
	}
	return sp, nil
}

func (s *Service) resolveSAMLACSURL(provider Provider) (*url.URL, error) {
	raw := strings.TrimSpace(provider.ACSURL)
	if raw == "" {
		raw = "http://127.0.0.1:8080/api/v1/sso/saml/acs"
	}
	if err := validateAbsoluteURL(raw, "ACS URL"); err != nil {
		return nil, err
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, invalid("ACS URL 不合法")
	}
	return u, nil
}

func (s *Service) loadIDPMetadata(ctx context.Context, provider Provider) (*saml.EntityDescriptor, error) {
	if u := strings.TrimSpace(provider.MetadataURL); u != "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}
		resp, err := s.client().Do(req)
		if err != nil {
			return nil, invalid("无法下载 IdP Metadata")
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, invalid("IdP Metadata 返回异常")
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		if err != nil {
			return nil, err
		}
		var ed saml.EntityDescriptor
		if err := xml.Unmarshal(body, &ed); err == nil && ed.EntityID != "" {
			return &ed, nil
		}
		var entities saml.EntitiesDescriptor
		if err := xml.Unmarshal(body, &entities); err == nil {
			for i := range entities.EntityDescriptors {
				if len(entities.EntityDescriptors[i].IDPSSODescriptors) > 0 {
					return &entities.EntityDescriptors[i], nil
				}
			}
			if len(entities.EntityDescriptors) > 0 {
				return &entities.EntityDescriptors[0], nil
			}
		}
		return nil, invalid("无法解析 IdP Metadata")
	}

	ssoURL := strings.TrimSpace(provider.SSOURL)
	entityID := strings.TrimSpace(provider.EntityID)
	if ssoURL == "" || entityID == "" {
		return nil, invalid("未配置 Metadata URL 时必须同时提供 Entity ID 与 SSO URL")
	}
	certPEM := strings.TrimSpace(provider.IDPCertificatePEM)
	if certPEM == "" {
		return nil, invalid("未配置 Metadata URL 时必须提供 IdP 证书 PEM")
	}
	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return nil, invalid("IdP 证书 PEM 无效")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, invalid("IdP 证书解析失败")
	}
	return &saml.EntityDescriptor{
		EntityID: entityID,
		IDPSSODescriptors: []saml.IDPSSODescriptor{{
			SSODescriptor: saml.SSODescriptor{
				RoleDescriptor: saml.RoleDescriptor{
					KeyDescriptors: []saml.KeyDescriptor{{
						Use: "signing",
						KeyInfo: saml.KeyInfo{
							X509Data: saml.X509Data{
								X509Certificates: []saml.X509Certificate{{
									Data: base64.StdEncoding.EncodeToString(cert.Raw),
								}},
							},
						},
					}},
				},
			},
			SingleSignOnServices: []saml.Endpoint{{
				Binding:  saml.HTTPRedirectBinding,
				Location: ssoURL,
			}},
		}},
	}, nil
}

func (s *Service) ensureSAMLSPKey(ctx context.Context) (*rsa.PrivateKey, *x509.Certificate, error) {
	if s.vault == nil {
		return nil, nil, notReady("联邦密钥服务未配置")
	}
	settings, err := s.store.Settings(ctx)
	if err != nil {
		return nil, nil, err
	}
	if settings.SPKeySecretID != "" && strings.TrimSpace(settings.SPCertificatePEM) != "" {
		ciphertext, err := s.store.GetSecretCiphertext(ctx, settings.SPKeySecretID)
		if err == nil {
			plain, err := s.vault.Decrypt(ciphertext)
			if err == nil {
				key, cert, err := parseSPKeyMaterial(plain, settings.SPCertificatePEM)
				if err == nil {
					return key, cert, nil
				}
			}
		}
	}

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject:      pkix.Name{CommonName: "qinzhi-saml-sp"},
		NotBefore:    time.Now().UTC().Add(-time.Hour),
		NotAfter:     time.Now().UTC().AddDate(10, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, nil, err
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, nil, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	secretID, err := s.storeSecret(ctx, "saml_signing_key", string(keyPEM))
	if err != nil {
		return nil, nil, err
	}
	settings.SPKeySecretID = secretID
	settings.SPCertificatePEM = string(certPEM)
	settings.UpdatedAt = s.now().UTC()
	if err := s.store.SaveSettings(ctx, settings); err != nil {
		return nil, nil, err
	}
	return key, cert, nil
}

func parseSPKeyMaterial(keyPEM []byte, certPEM string) (*rsa.PrivateKey, *x509.Certificate, error) {
	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return nil, nil, invalid("SAML SP 私钥无效")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		parsed, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, nil, invalid("SAML SP 私钥解析失败")
		}
		var ok bool
		key, ok = parsed.(*rsa.PrivateKey)
		if !ok {
			return nil, nil, invalid("SAML SP 私钥必须是 RSA")
		}
	}
	cblock, _ := pem.Decode([]byte(certPEM))
	if cblock == nil {
		return nil, nil, invalid("SAML SP 证书无效")
	}
	cert, err := x509.ParseCertificate(cblock.Bytes)
	if err != nil {
		return nil, nil, invalid("SAML SP 证书解析失败")
	}
	return key, cert, nil
}

func samlAttribute(assertion *saml.Assertion, mapping map[string]any, logical string, defaults ...string) string {
	keys := defaults
	if mapping != nil {
		if raw, ok := mapping[logical]; ok {
			if s, ok := raw.(string); ok && strings.TrimSpace(s) != "" {
				keys = []string{strings.TrimSpace(s)}
			}
		}
	}
	for _, stmt := range assertion.AttributeStatements {
		for _, attr := range stmt.Attributes {
			for _, key := range keys {
				if strings.EqualFold(attr.Name, key) || strings.EqualFold(attr.FriendlyName, key) {
					for _, v := range attr.Values {
						if strings.TrimSpace(v.Value) != "" {
							return strings.TrimSpace(v.Value)
						}
					}
				}
			}
		}
	}
	return ""
}
