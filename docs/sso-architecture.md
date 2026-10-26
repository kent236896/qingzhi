# 统一身份认证与 SSO：阶段 A 工程检查与架构设计

> 状态：阶段 A/B/C 已完成可验证交付。D–H（本平台作为 AS/OP/SAML IdP 等）尚未实现。

## 1. 现有工程结论

| 能力 | 现状 | 对本模块的影响 |
|---|---|---|
| 前端 | `apps/admin-web` Vue3 + TDesign + Pinia + Router | 新增「系统安全」导航与页面，复用布局/鉴权 |
| 后端 | `services/enterprise-api` Gin + GORM + migrate | 新增 `internal/sso` 等领域包与 Handler，不改基础设施 |
| 本地登录 | `identity.Service.Authenticate` + `auth.TokenIssuer`（HS256 JWT） | 保留为应急登录；SSO 成功后仍签发**本平台 JWT** |
| 会话 | **无 Redis 会话**（架构规则禁止）；JWT 自包含 + `LoadSessionAccess` | 协议授权码/state 等短态存 PostgreSQL，不引入 Redis 会话 |
| 用户/租户/角色 | `sys_users`、租户、组织、角色、菜单权限齐全 | 身份映射挂到现有 `sys_users`；禁止外部 Claim 直接提权 |
| 审计 | `sys_login_logs`、`sys_operate_logs`、`sys_permission_logs` | 新增 `sys_sso_auth_events` 专表；配置变更走操作审计 + SSO 事件 |
| 安全策略 | `sys_security_policy` 密码/锁定 | 本地登录继续遵守；外部失败不暴露是否存在账号 |
| Redis | 仅健康检查 | 协议状态**不**依赖 Redis |
| OAuth/OIDC/SAML | **不存在** | 全新建设 |
| go.mod | 无 fosite / go-oidc / crewjam | 阶段 C+ 按需引入，见 §5 |

关键文件：

- `internal/auth/{jwt,middleware,context}.go`
- `internal/identity/auth.go`（`Authenticate` / `LoadSessionAccess`）
- `internal/security/{audit,login}.go`
- `internal/handler/auth.go`
- `apps/admin-web/src/views/LoginView.vue`、`stores/auth.ts`

## 2. 角色边界（必须严格区分）

```text
┌─────────────────────────────────────────────────────────────┐
│ 角色 B：本平台 = 客户端 / SP                                 │
│  federation/oidc_client · oauth2_client · saml_sp           │
│  → 验证外部令牌/断言 → identity_mapping → 签发本平台 JWT     │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│ 角色 A：本平台 = AS / OP / IdP                               │
│  authorization_server/oauth2 · oidc · saml_idp              │
│  → 校验本平台用户会话 → 发授权码/令牌/断言给外部应用          │
└─────────────────────────────────────────────────────────────┘

共享：sys_users、菜单权限、操作审计、加密密钥服务、概览统计
不共享：协议 state 表、令牌表、签名密钥用途、Redirect/ACS 白名单
```

## 3. 拟议模块结构

```text
services/enterprise-api/internal/sso/
├── overview/          # 管理端概览统计（阶段 B）
├── provider/          # 外部 IdP 配置 CRUD（OIDC/OAuth2/SAML）（B→C/E）
├── clientreg/         # OAuth2/OIDC 客户端注册 + SAML SP 注册（B→D/F）
├── mapping/           # 外部身份 ↔ sys_users（C）
├── credential/        # Client Secret / 私钥加密存取与轮换（B 骨架，G 完善）
├── audit/             # sys_sso_auth_events（B）
├── oauth2as/          # Authorization Server（D，基于 Fosite）
├── oidcop/            # OIDC Provider 扩展（D）
├── oidcrp/            # OIDC Relying Party（C）
├── oauth2rp/          # 通用 OAuth2 登录适配（C，需可信 userinfo）
├── samlsp/            # SAML SP（E）
└── samlidp/           # SAML IdP（F，需单独验证 crewjam 能力）

apps/admin-web/src/
├── views/sso/         # 概览、外部提供方、客户端、映射、审计
├── api/sso.ts
└── router 增加 /security/sso/*
```

Handler 注册在 `internal/handler/sso_*.go`，由 `server.go` 挂载。

## 4. 数据库迁移规划

| 迁移 | 内容 | 阶段 |
|---|---|---|
| `000022_sso_foundation` | 菜单「系统安全」；`sys_sso_providers`、`sys_sso_oauth_clients`、`sys_sso_saml_sps`、`sys_sso_identity_links`、`sys_sso_auth_events`、`sys_sso_secrets`（密文元数据） | B |
| `000023_sso_oauth_runtime` | 授权码、同意记录、刷新令牌哈希、JWKS 密钥元数据 | D |
| `000024_sso_saml_runtime` | 断言重放缓存、可选 SLO 会话关联 | E/F |

原则：golang-migrate only；Secret 仅存密文/哈希；不整包 JSON 糊弄协议状态。

## 5. 第三方依赖（拟引入，阶段 C+ 才加入 go.mod）

| 库 | 用途 | 引入原因 |
|---|---|---|
| `github.com/ory/fosite` | OAuth2 AS + OIDC OP | 不自研授权码/PKCE/令牌引擎 |
| `github.com/coreos/go-oidc/v3` | OIDC RP：Discovery、JWKS、ID Token 校验 | 标准校验路径 |
| `golang.org/x/oauth2` | 授权码交换与 PKCE | 与 go-oidc 配套 |
| `github.com/crewjam/saml` | SAML SP/IdP | 成熟 XML+签名；**IdP 能力需单独互操作与安全评审** |

若 crewjam IdP 无法满足 Audience/重放/签名要求，阶段 F 将报告缺口并提出可替换适配器，**不会删校验硬过**。

加密：AES-GCM，密钥来自环境变量 `SSO_SECRETS_KEY`（≥32 字节），与 `JWT_SECRET` 分离。

## 6. API 清单（规划）

### 管理端（需菜单权限）

| 方法 | 路径 | 阶段 |
|---|---|---|
| GET | `/api/v1/sso/overview` | B |
| CRUD | `/api/v1/sso/providers` | B/C |
| POST | `/api/v1/sso/providers/:id/test` | C/E |
| CRUD | `/api/v1/sso/oauth-clients` | B/D |
| POST | `/api/v1/sso/oauth-clients/:id/rotate-secret` | G |
| CRUD | `/api/v1/sso/saml-sps` | B/F |
| GET | `/api/v1/sso/saml-sps/:id/metadata` | F |
| GET | `/api/v1/sso/identity-links` | C |
| GET | `/api/v1/sso/auth-events` | B |

### 协议端点（公共或半公共）

| 角色 | 端点 | 阶段 |
|---|---|---|
| OP | `/.well-known/openid-configuration`、`/oauth2/auth`、`/oauth2/token`、`/oauth2/revoke`、`/oauth2/introspect`、`/userinfo`、`/jwks.json` | D |
| RP | `/api/v1/sso/oidc/start`、`/api/v1/sso/oidc/callback` | C |
| OAuth2 RP | `/api/v1/sso/oauth2/start`、`.../callback` | C |
| SAML SP | `/saml/metadata`、`/saml/acs`、`/saml/slo`（若启用） | E |
| SAML IdP | `/saml/idp/metadata`、`/saml/idp/sso` | F |

Issuer / EntityID / 回调基址来自受控配置（如 `SSO_ISSUER`），**禁止**按未信任 Host 动态拼 Issuer。

## 7. 身份映射规则（与现有用户衔接）

1. 唯一键：`(provider_id, external_subject)` → `user_id`。  
2. **禁止**仅凭邮箱自动合并两个已存在账号。  
3. 自动开户默认关；开启后新用户仅最小权限（无 `knowledge:manage` / 超级管理员菜单）。  
4. 外部 Claim 不得直接写入角色；提权仅管理员在本平台操作。  
5. 提供方停用 / 用户停用 / 解绑后：拒绝新会话；已签发 JWT 按 TTL 自然过期（阶段 G 可加黑名单若需要）。

衔接点：`identity.Service` 增加「按 ID 加载用户 / 创建最小权限用户」内部方法，不新造用户体系。

## 8. 安全决策摘要

- 不实现 Implicit / ROPC。  
- Redirect URI / ACS **精确匹配**，无通配。  
- PKCE S256（公开客户端强制；机密客户端按注册策略）。  
- state / nonce / InResponseTo / 断言重放防护。  
- 管理 API 永不返回完整 Client Secret / 私钥；轮换时一次性展示。  
- 日志脱敏：无密码、完整令牌、Secret、私钥、完整断言。  
- 本地管理员应急登录始终可配置保留。  
- 架构规则：不引入 Redis 会话。

## 9. 分阶段验收标准

| 阶段 | 可独立验证的完成定义 |
|---|---|
| A | 本文档 + 依赖/表/API 清单 |
| B | 迁移可 up/down；概览 API 真实计数；菜单可进；无假数据 |
| C | 外部 OIDC 真登录一轮；映射与审计落库；测试客户端/手工脚本可复现 |
| D | Discovery+授权码+ID Token；独立测试客户端互操作通过 |
| E | 外部 IdP → 本平台 SP 登录成功；负例测试（坏签名等） |
| F | 外部 SP → 本平台 IdP SSO；元数据下载；IdP-initiated 默认关 |
| G | Secret 轮换、证书到期提醒、完整审计查询 |
| H | 全量自动化 + 部署/密钥文档 + 未支持项清单 |

## 10. 交付进度

### 已完成

| 阶段 | 内容 |
|---|---|
| A | 工程检查、角色边界、模块/API/迁移/依赖清单（本文） |
| B | `000022_sso_foundation` 迁移；`internal/sso` 配置/审计/密钥加密骨架；管理 API；「系统安全」菜单与 6 个真实数据页面 |
| C | `000023_sso_oidc_rp`；外部 OIDC/OAuth2 授权码+PKCE；ID Token / userinfo 校验；身份映射；一次性票据兑换本平台 JWT；登录页对接 |

### 未完成（不得标为已交付）

| 阶段 | 缺口 |
|---|---|
| D | OAuth2 AS / OIDC OP（Discovery、授权码、JWKS、UserInfo；Fosite） |
| E | SAML SP（Metadata、ACS、断言校验） |
| F | SAML IdP（SSO、签名断言；crewjam 能力评审） |
| G | 密钥轮换完善、证书诊断、完整生命周期策略 |
| H | 协议互操作自动化、部署与验收文档终稿 |

### Keycloak 联调要点（角色 B）

1. 迁移执行到 `000023`。  
2. `.env` 设置 `SSO_FRONTEND_BASE_URL`（如 `http://localhost:5173`）与可选 `SSO_DEFAULT_TENANT_ID`（自动开户时必填）。  
3. 管理端登记 OIDC 提供方：Issuer / Client ID / Secret / Redirect URI=`http://127.0.0.1:8080/api/v1/sso/oidc/callback`，启用并打开「登录页显示」。  
4. 登录页点击外部登录 → Keycloak → 回调 API → 前端 `/login/sso/callback` 兑票。  
5. 默认**不**按邮箱合并账号；未绑定且未开自动开户时拒绝登录。
