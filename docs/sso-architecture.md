# 统一身份认证与 SSO：客户端登录架构

> 状态：本平台**仅作为客户端**对接外部 IdP。支持 **OIDC RP** 与 **SAML SP** 登录；不再实现本平台作为 OAuth2 AS / OIDC OP / SAML IdP。

## 1. 角色边界

```text
外部 IdP（Keycloak / Azure AD / 企业 SAML IdP …）
        │  OIDC 授权码 + PKCE  /  SAML AuthnRequest
        ▼
┌─────────────────────────────────────────────────────────────┐
│ 擎知 = 客户端 / SP                                           │
│  校验 ID Token 或 SAML 断言 → 身份映射 → 签发本平台 JWT      │
└─────────────────────────────────────────────────────────────┘
```

本平台登录成功后仍签发 **本平台 HS256 JWT**；无 Redis 会话。

## 2. 已交付能力

| 能力 | 说明 |
|---|---|
| 外部身份提供方 CRUD | 协议仅 `oidc` / `saml` |
| OIDC 登录 | `/api/v1/sso/oidc/start` → IdP → `/oidc/callback` → 前端兑票 |
| SAML 登录 | `/api/v1/sso/saml/start` → IdP → `POST /saml/acs` → 前端兑票 |
| SP Metadata | `GET /api/v1/sso/saml/metadata?provider_id=` |
| 身份映射 / 认证审计 / 概览 | 管理端「系统安全」 |
| 本地应急登录 | `sys_sso_settings.local_login_enabled` |

## 3. 明确不做（已移除）

| 能力 | 说明 |
|---|---|
| OAuth / OIDC 客户端登记 | 原「本平台作为 AS/OP」登记表与菜单已删除 |
| SAML 服务提供方登记 | 原「本平台作为 IdP」登记表与菜单已删除 |
| 通用 OAuth2（非 OIDC）外部登录 | 协议不再接受 `oauth2` |
| Discovery / `/oauth2/*` 作为 OP | 不实现 |

## 4. 模块与表

```text
services/enterprise-api/internal/sso/
├── login.go        # OIDC RP
├── saml_login.go   # SAML SP（crewjam/saml）
├── service.go      # 管理 API
└── …

apps/admin-web/src/views/sso/
├── OverviewView / ProvidersView / IdentityLinksView / AuthEventsView
```

关键表：`sys_sso_settings`、`sys_sso_providers`、`sys_sso_identity_links`、`sys_sso_auth_events`、`sys_sso_oidc_auth_requests`、`sys_sso_login_tickets`、`sys_sso_secrets`。  
SAML SP 签名私钥密文存于 `sys_sso_secrets`（kind=`saml_signing_key`），证书 PEM 在 `sys_sso_settings.sp_certificate_pem`（首次 SAML 登录时自动生成）。

## 5. 身份映射

1. 唯一键：`(provider_id, external_subject)` → `user_id`（OIDC `sub` / SAML NameID）；已有映射直接登录。  
2. 无映射时：先按 IdP **邮箱**（忽略大小写），再按 **用户名**（OIDC `preferred_username` / SAML uid 等）匹配本平台正常状态账号；命中则自动写入映射并登录。  
3. 仍无匹配时：自动开户开启则新建最小权限用户并映射；关闭则拒绝并提示「外部身份尚未绑定本平台账号」。  
4. 外部 Claim / Attribute 不得直接写入角色。

## 6. 联调入口

- Keycloak（OIDC）：[sso-keycloak.md](sso-keycloak.md)  
- SAML：在「外部身份提供方」登记 IdP Metadata（或 Entity ID + SSO URL + 证书），ACS 填 `http://127.0.0.1:8080/api/v1/sso/saml/acs`，把 SP Metadata URL 交给 IdP。
