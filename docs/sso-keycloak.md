# 擎知作为客户端：对接 Keycloak（OIDC）

本文说明如何把 **擎知（qinzhi）作为 OIDC 客户端 / Relying Party**，把 **Keycloak 作为认证授权端（OpenID Provider）**，完成管理后台的外部登录。

> 角色边界：擎知只作为 OIDC 客户端。校验 Keycloak 签发的 ID Token，映射到本平台用户后签发**本平台 JWT**。本平台不作为授权服务器 / OP；架构说明见 [sso-architecture.md](sso-architecture.md)。

## 1. 登录流程

```text
浏览器                擎知前端                 擎知 API                    Keycloak
  |                      |                        |                           |
  | 打开 /login         |                        |                           |
  |--------------------->|  GET /sso/login-options |                           |
  |                      |----------------------->|                           |
  | 点击「Keycloak 登录」|                        |                           |
  |--------------------->|  跳转 /sso/oidc/start  |                           |
  |                      |----------------------->|  302 → Keycloak /auth     |
  |                      |                        |-------------------------->|
  |  在 Keycloak 登录 / 同意授权                  |                           |
  |<--------------------------------------------------------------------------|
  |  回调 /api/v1/sso/oidc/callback?code&state    |                           |
  |---------------------------------------------->|  换码 + 校验 ID Token     |
  |                      |                        |  身份映射 / 可选自动开户  |
  |                      |                        |  签发一次性 ticket        |
  |  302 → 前端 /login/sso/callback?ticket=...    |                           |
  |<----------------------------------------------|                           |
  |  POST /sso/oidc/exchange                      |                           |
  |--------------------->|----------------------->|  兑换本平台 access_token  |
  |  进入管理后台（Bearer JWT）                   |                           |
```

关键协议端点：

| 步骤 | 方法 | 路径 |
| --- | --- | --- |
| 登录按钮列表 | `GET` | `/api/v1/sso/login-options` |
| 发起授权 | `GET` | `/api/v1/sso/oidc/start?provider_id=...&redirect=...&frontend_base=...` |
| IdP 回调 | `GET` | `/api/v1/sso/oidc/callback` |
| 兑票 | `POST` | `/api/v1/sso/oidc/exchange` |

本地默认地址：

| 组件 | 地址 |
| --- | --- |
| 擎知前端 | `http://127.0.0.1:5173` |
| 擎知 API | `http://127.0.0.1:8080` |
| OIDC 回调（填到 Keycloak） | `http://127.0.0.1:8080/api/v1/sso/oidc/callback` |

## 2. 前置条件

1. 已执行迁移到至少 `000023`（OIDC RP 运行时表）；建议同步执行到最新（含 `000024` 前端基址列）。
2. API 与前端已启动；库中至少有一个可用租户（全新环境可由 `BOOTSTRAP_*` 自动创建）。
3. `.env` 中：
   - `CORS_ALLOWED_ORIGINS` 包含前端 Origin（如 `http://127.0.0.1:5173`、`http://localhost:5173`）。
   - `SSO_SECRETS_KEY`（≥32 字符，或 32 字节 base64）用于加密 Client Secret；与 `JWT_SECRET` 分开。
   - `SSO_LOCAL_LOGIN_ENABLED=true` 建议联调阶段保留，便于应急本地登录。
4. 管理端「系统安全 → 身份认证与单点登录」中配置**前端基址**（可点「使用当前浏览器地址」）。登录发起时也会附带 `window.location.origin`，须与 CORS 白名单一致。一般不必再写 `SSO_FRONTEND_BASE_URL`。

## 3. 在 Keycloak 中配置

以下以 Keycloak 25+ 管理控制台为例；菜单名称因版本可能略有差异。

### 3.1 Realm 与用户

1. 创建或选择一个 Realm，例如 `qinzhi`。
2. 在 **Users** 中创建测试用户，设置密码，并保证邮箱等属性按需要填写（可选）。
3. 记录 Issuer（Realm 的 OpenID Provider 根地址），形如：

```text
http://127.0.0.1:8081/realms/qinzhi
```

可用浏览器打开：

```text
{Issuer}/.well-known/openid-configuration
```

确认能返回 Discovery JSON。擎知会用该 Issuer 做 Discovery，并自动取 Authorization / Token / JWKS 端点。

### 3.2 创建 Confidential Client

1. **Clients → Create client**
2. Client type：`OpenID Connect`
3. Client ID：例如 `qinzhi-admin`
4. Capability：
   - Client authentication：**On**（机密客户端）
   - Standard flow：**On**（Authorization Code）
   - Direct access grants：**Off**（擎知不使用 ROPC）
   - Implicit flow：**Off**
5. Login settings：
   - **Valid redirect URIs**（精确匹配，须与擎知登记的 Redirect URI 完全一致）：

```text
http://127.0.0.1:8080/api/v1/sso/oidc/callback
```

   - **Valid post logout redirect URIs**（退出后允许回到的前端地址，精确匹配或按 Keycloak 版本允许的通配）：

```text
http://127.0.0.1:5173/login
http://localhost:5173/login
```

   - **Web origins**（CORS，给浏览器直连 Keycloak 时用；本流程由 API 服务端换码，仍建议写上前端 Origin）：

```text
http://127.0.0.1:5173
http://localhost:5173
```

6. 保存后到 **Credentials** 页复制 **Client secret**。
7. **Advanced**（或客户端策略）中确认允许 **PKCE S256**。擎知发起授权时始终带 `code_challenge` / `code_challenge_method=S256`。
8. （可选）**Logout settings / Front-channel logout**：若需 IdP 侧会话同步退出，可填前端登出回跳页；当前管理端「退出」默认只清除本平台 JWT，见下方「退出与登出」。

### 3.3 Client Scopes（建议）

默认 Scope 使用：

```text
openid profile email
```

确保 ID Token 或可解析 Claims 中至少包含：

| Claim | 用途 |
| --- | --- |
| `sub` | 外部主体，身份映射主键（必需） |
| `email` | 首次登录时按邮箱匹配本平台账号；自动开户时写入新用户邮箱 |
| `preferred_username` / `username` | 邮箱未命中时按用户名匹配本平台账号 |
| `name` 或 `preferred_username` | 自动开户时的昵称回退（可选） |

Keycloak 默认会在 ID Token / UserInfo 中提供上述字段；若自定义 Client Scope 过严，请保证 `sub` 始终存在。

## 4. 在擎知中登记 Keycloak

使用具备 `system:sso:provider:manage` 权限的管理员登录本地账号。

### 4.1 认证策略

打开 **系统安全 → 身份认证与单点登录**：

1. 确认「本地账号登录」仍开启（联调失败时可回退）。
2. 「前端基址」设为当前管理端 Origin，例如 `http://127.0.0.1:5173`。
3. 「外部自动开户默认租户 ID」：若开启自动开户，建议填明确租户 UUID；留空则使用最早可用租户或 `SSO_DEFAULT_TENANT_ID`。
4. 保存策略。

### 4.2 外部身份提供方

打开 **系统安全 → 外部身份提供方 → 新建**：

| 字段 | 示例 / 说明 |
| --- | --- |
| 名称 | `Keycloak` |
| 协议 | `OIDC` |
| Issuer | 你的 Keycloak Realm 根地址，例如 `http://192.168.0.170:32548/realms/qinzhi-admin`（**不要照抄文档里的 127.0.0.1:8081**；须能被 **API 进程**访问并打开 `/.well-known/openid-configuration`） |
| Client ID | `qinzhi-admin` |
| Client Secret | Keycloak Credentials 中的 Secret |
| Redirect URI | `http://127.0.0.1:8080/api/v1/sso/oidc/callback` |
| Scopes | `openid profile email` |
| 登录页显示 | 开 |
| 自动开户 | 见下一节 |

创建时默认**立即启用**且「登录页显示」开启；若列表状态仍为停用，点行内「启用」。只有 **status=enabled 且登录页显示=开** 才会出现在 `/login`。

> SAML 提供方登记页暂有入口，但 SAML SP 运行时尚未交付；对接 Keycloak 请使用 **OIDC**，不要选 SAML。

### 4.4 退出与登出地址（Keycloak）

Keycloak Realm 的 OIDC Discovery 中会给出：

| 用途 | 地址（用你的 Issuer 替换） |
| --- | --- |
| 结束会话（RP-Initiated Logout） | `{Issuer}/protocol/openid-connect/logout` |
| Discovery | `{Issuer}/.well-known/openid-configuration` 中的 `end_session_endpoint` |

在 Keycloak Client 中至少配置：

| Keycloak 字段 | 本地示例 | 说明 |
| --- | --- | --- |
| Valid post logout redirect URIs | `http://127.0.0.1:5173/login` | 联邦登出完成后允许跳回的前端登录页；须与实际 `post_logout_redirect_uri` 一致 |
| Valid redirect URIs | `http://127.0.0.1:8080/api/v1/sso/oidc/callback` | 登录回调，与退出无关，但常一起核对 |

浏览器侧发起联邦登出的典型 URL（可选 `id_token_hint`，Keycloak 多数版本建议带上）：

```text
{Issuer}/protocol/openid-connect/logout
  ?client_id=qinzhi-admin
  &post_logout_redirect_uri=http%3A%2F%2F127.0.0.1%3A5173%2Flogin
```

说明：

1. 管理端右上角「退出」当前只清本平台会话（本地 JWT），**不会**自动调 Keycloak logout；用户在 Keycloak 侧会话可能仍在，再次点 OIDC 登录可能免密。
2. 若需要「退出即清 Keycloak 会话」，应在清本地 token 后跳转到上面的 `end_session_endpoint`，且 Client 已登记对应的 **Valid post logout redirect URIs**。
3. 生产环境把 `127.0.0.1` 换成对外 HTTPS 域名，登录回调与登出回跳分别指向 API / 前端正式地址。

### 4.3 身份映射与自动开户

映射规则：`(provider_id, Keycloak sub)` → `sys_users.id`。

| 策略 | 行为 |
| --- | --- |
| 已有身份映射 | 直接登录对应本平台用户。 |
| 无映射，邮箱或用户名能匹配到本平台正常账号 | 自动建立映射并登录（与自动开户开关无关）。 |
| 自动开户 = 关，且无映射、邮箱/用户名也匹配不到 | 拒绝，提示「外部身份尚未绑定本平台账号」。 |
| 自动开户 = 开，且无映射、邮箱/用户名也匹配不到 | 新建最小权限用户并写入映射；新用户名形如联邦用户名（与 `provider_id` + `sub` 相关）。 |
| 权限 | 外部 Claim **不会**映射成角色；提权只在本平台角色管理中操作。 |

联调建议：Keycloak 用户邮箱或 `preferred_username` 与本平台 `admin` 等账号对齐后，即使关闭自动开户也可首次登录并自动绑定。

## 5. 验证步骤

1. 退出管理端，打开 `http://127.0.0.1:5173/login`。
2. 应看到外部登录按钮（提供方名称）。
3. 点击后跳转到 Keycloak 登录页；用测试用户登录。
4. 成功后应回到擎知前端并进入后台；请求头携带本平台 JWT。
5. 在 **系统安全 → 认证审计** 中应能看到成功的 `login.*` / `identity_link.create` 等事件。
6. 在 **身份映射** 中应能看到 Keycloak `sub` 与本平台用户的绑定。

### 5.1 负例（建议各走一遍）

| 场景 | 期望 |
| --- | --- |
| Redirect URI 与 Keycloak 登记不一致 | Keycloak 拒绝或回调失败 |
| 提供方停用 | 登录页不再展示或 start 失败 |
| 自动开户关闭且无映射、邮箱/用户名也不匹配 | 回调后提示外部身份未绑定 |
| 邮箱或 preferred_username 与本平台一致 | 即使关闭自动开户也会自动绑定并登录 |
| 错误 Client Secret | 授权码交换失败，审计记失败 |

## 6. 生产注意

1. 全站 HTTPS；Issuer、Redirect URI、前端基址一律使用对外可达的正式域名。
2. Keycloak Valid redirect URIs **精确匹配**，不要用通配。
3. Client Secret 只在创建/轮换时由管理员粘贴；接口永不回读明文。
4. `SSO_SECRETS_KEY` / `JWT_SECRET` 使用独立高强度密钥，并纳入密钥轮换流程。
5. CORS 只放行真实管理端 Origin；`frontend_base` 必须落在白名单内，防止开放重定向。
6. 保留可关闭的本地应急登录，直到确认 Keycloak 与运维流程可靠。
7. Keycloak 与擎知之间的时钟偏差会影响 ID Token `exp`/`iat` 校验，请保证 NTP 同步。

## 7. 排障清单

| 现象 | 排查 |
| --- | --- |
| 登录页没有外部按钮 | 提供方是否**启用**（新建默认启用；旧数据可能仍是停用）、是否「登录页显示」；`GET /api/v1/sso/login-options` 的 `providers` 应非空 |
| 退出后点 OIDC 仍免密进 Keycloak | 属预期：本平台退出未清 IdP 会话；需配置 Valid post logout redirect URIs 并走 `end_session_endpoint` |
| `无法加载 OIDC Discovery` | Issuer 是否可被 **API 进程**访问（容器网络 / hosts）；`/.well-known/openid-configuration` |
| Keycloak 报 invalid redirect_uri | 两端 Redirect URI 字符级一致（含 `127.0.0.1` vs `localhost`） |
| `授权码交换失败` | Client Secret、PKCE、Client authentication 是否机密客户端 |
| `ID Token 校验失败` / `nonce` | Issuer 与 Client ID 是否匹配；是否被中间人改写；时钟 |
| 回调后仍停在登录页 | 前端基址 / CORS；浏览器控制台与认证审计 |
| `自动开户需要先创建租户` | 先 bootstrap 或手工建租户，或配置默认租户 ID |
| 从概览页点进子菜单异常 | 已修复空 `certificate_warnings` 导致表格崩溃；请更新到含该修复的版本并硬刷新 |

## 8. 相关文档与代码

- 总体架构与阶段： [sso-architecture.md](sso-architecture.md)
- 开发环境与 env： [development.md](development.md)
- 协议实现：`services/enterprise-api/internal/sso/login.go`
- 管理页：`apps/admin-web/src/views/sso/*`、登录页 `LoginView.vue`
