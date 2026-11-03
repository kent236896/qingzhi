# WeKnora 长期记忆与统一身份

本平台通过 **HTTP API / 官方 Go SDK** 集成 WeKnora，不依赖 WeKnora 内部源码。

## 1. 统一身份

本地账号登录与 Keycloak/OIDC 登录在签发本平台 JWT 时汇合：

| 登录方式 | JWT `amethod` | WeKnora `sub` |
| --- | --- | --- |
| 用户名密码 | `local` | 本平台 `sys_users.id` |
| SSO（含 Keycloak） | `sso` | **同一** `sys_users.id`（经身份映射后） |

不会仅因邮箱相同自动合并账号；SSO 绑定仍遵循现有 `sys_sso_identity_links` 规则。

统一身份对象（服务端内部）：

- `PlatformUserID`：本平台用户 ID  
- `PlatformTenantID`：本平台租户 ID  
- `WeKnoraTenantID`：优先系统配置「WeKnora 空间 ID」；空则默认 `10000`；首次启动可用 `WEKNORA_TENANT_ID` 写入空库
- `AuthenticationMethod`：`local` / `sso`  

终端用户身份**只**从后端已认证 `Principal` 构造，不接受前端传入的 user_id / tenant_id。

## 2. 短期 `X-External-User-Token`

当 WeKnora 空间 `api-principal-config.mode=signed_token` 时，后端使用 `WEKNORA_EXTERNAL_USER_HMAC`（须与 WeKnora 侧 `hmac_secret` 一致）签发 HS256 JWT：

- `sub`：PlatformUserID  
- `tenant_id`：WeKnoraTenantID  
- `aud`：`weknora`  
- `iat` / `exp`：默认 TTL 10 分钟，最长 24 小时  

请求 WeKnora 时由服务端同时携带现有 `X-API-Key` 与（在需要终端主体时）`X-External-User-Token`。浏览器不持有 API Key / HMAC。管理面 Admin JWT 与用户面 External Token 的分工见 [weknora-call-planes.md](weknora-call-planes.md)。

管理端调试接口：`POST /api/v1/weknora/external-user-token`（需 `model:manage`）。

## 3. 空间级长期记忆配置（已实现）

| 本平台 | WeKnora |
| --- | --- |
| `GET /api/v1/weknora/memory-config` | `GET /api/v1/tenants/kv/memory-config` |
| `PUT /api/v1/weknora/memory-config` | `PUT /api/v1/tenants/kv/memory-config` |

权限：`model:query` / `model:manage`。保存流程：先读完整配置 → 合并表单 → 校验 → PUT 完整对象 → 再读确认。操作走现有 `OperateAudit`。

管理页：系统管理 → **长期记忆**（`/settings/memory`）。

字段：`enabled`、`write_mode`、`extract_model_id`、`max_items`、`extract_delay_seconds`、`extract_min_interval_seconds`、`extract_instructions`、`interest_threshold`、`embedding_model_id`、`vector_recall`、`retrieval_conditioning`。

## 4. 个人长期记忆（已开放）

前置：WeKnora 空间 `api-principal-config.mode=signed_token`，且与平台 `WEKNORA_EXTERNAL_USER_HMAC` 一致。可用系统配置页「启用 signed_token 主体」：用已保存的 WeKnora 管理员邮箱/密码登录换 JWT 后写入配置。

平台代理（登录即可；服务端注入 External User Token，**禁止**前端传 user_id）：

| 本平台 | WeKnora |
| --- | --- |
| `GET /api/v1/memory/status` | 门禁 + 只读空间摘要 |
| `GET/PUT /api/v1/memory/settings` | `GET/PUT /api/v1/memory/settings` |
| `GET/POST /api/v1/memory/items` | 同路径 |
| `PUT/DELETE /api/v1/memory/items/{id}` | 同路径 |
| `POST .../confirm`、`.../reject` | 同路径 |

工作台：`/memory` 开关 + 列表 CRUD。门禁 `PersonalMemoryIsolation().Ready=true`（2026-10-30 复测 PASS）。

### 4.1 真实环境验证记录

#### 2026-10-28 / 2026-10-29

在未正确配置 `api-principal-config=signed_token` 时，`/memory/*` 回退 API Key 主体，跨用户共享。详见历史记录；当时保持 `ready=false`。

#### 2026-10-30（开放依据）

脚本：`scripts/memory-isolation-retest.go`（Step0 写入 `signed_token` + HMAC；同一 API Key + 用户 A/B External Token）。

| 检查项 | 结果 |
| --- | --- |
| 缺 Token `GET /api/v1/memory/settings` | **401** |
| A PUT `user_enabled=false`、B PUT `true` 后各自 GET | **互不影响** |
| A POST item 后 B list | **不可见**；`subject_id=api_external_user:<空间>:<sub>` |
| B DELETE A 的 item | **404** |
| **RESULT** | **PASS isolation_ok** |

结论：可安全开放个人记忆代理；必须注入 `X-External-User-Token`，禁止回退管理员主体。

## 5. WeKnora 侧必配

1. API Key 具备 `manage_tenant_settings` 或 `full-access`（空间配置）。  
2. 空间 `api-principal-config` 设为 `signed_token`，`hmac_secret` 与本平台 `WEKNORA_EXTERNAL_USER_HMAC` 相同。  
3. 本平台在 **系统配置** 填写 WeKnora 地址、API Key、空间 ID、管理员邮箱/密码；`.env` 仅作首次种子。管理员密码不回显。  
4. **API Key ≠ 管理员 JWT**：`WEKNORA_API_KEY` / 配置页「API Key」是工作区密钥；`WEKNORA_ADMIN_TOKEN` 须由 `POST /api/v1/auth/login`（邮箱+密码）换取，不要把 JWT 填进 API Key。开启 `signed_token` 后，仅 API Key 访问 `/system/info`、空间配置等会 401（`invalid external user token`），连通探测与管理接口需管理员 JWT。  
   - 终端用户路径（会话 / 问答 / 个人记忆）：**只**使用 API Key + `X-External-User-Token`，**禁止**再挂管理员 Bearer，否则会话会记到管理员名下，用户再问会 404。  
   - 管理路径（system info、models、memory-config、principal-config）：使用管理员 JWT（`ExchangeAdmin`）。  
5. 终端用户令牌仍用环境变量：`WEKNORA_EXTERNAL_USER_HMAC`、`WEKNORA_EXTERNAL_USER_TOKEN_TTL`（密钥不进前端）。  
6. 可选：`POST /api/v1/system/settings/ensure-principal`（或配置页按钮）用管理员账号登录并写入 signed_token。

## 6. 智能体 `memory_enabled`

平台智能体目录（`/api/v1/agents`）的 `config.memory_enabled` 可在管理端/工作台编辑；默认 `true`，创建/更新时写入 WeKnora 远端 config。空间记忆总开关仍在 `weknora/memory-config`；二者同时开启时个人记忆才会生效。
