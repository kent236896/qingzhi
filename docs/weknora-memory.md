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

请求 WeKnora 时由服务端同时携带现有 `X-API-Key` 与（在需要终端主体时）`X-External-User-Token`。浏览器不持有 API Key / HMAC。

管理端调试接口：`POST /api/v1/weknora/external-user-token`（需 `model:manage`）。

## 3. 空间级长期记忆配置（已实现）

| 本平台 | WeKnora |
| --- | --- |
| `GET /api/v1/weknora/memory-config` | `GET /api/v1/tenants/kv/memory-config` |
| `PUT /api/v1/weknora/memory-config` | `PUT /api/v1/tenants/kv/memory-config` |

权限：`model:query` / `model:manage`。保存流程：先读完整配置 → 合并表单 → 校验 → PUT 完整对象 → 再读确认。操作走现有 `OperateAudit`。

管理页：系统管理 → **长期记忆**（`/settings/memory`）。

字段：`enabled`、`write_mode`、`extract_model_id`、`max_items`、`extract_delay_seconds`、`extract_min_interval_seconds`、`extract_instructions`、`interest_threshold`、`embedding_model_id`、`vector_recall`、`retrieval_conditioning`。

## 4. 个人长期记忆（待集成）

公开文档要求个人 `/memory/*` 作用域由凭证决定，且 **API Key Principal 当前明确隔离的是 Session/MCP**，未保证 `signed_token` 对个人记忆接口实现跨用户隔离。

因此：

- **未实现**「我的长期记忆」读写页。  
- `GET /api/v1/weknora/memory/personal-status` 返回 `ready=false` 与原因说明。  
- **禁止**用超级管理员 API Key 冒充任意用户读写个人记忆。  

### 4.1 真实环境验证记录（2026-10-28）

对 `WEKNORA_BASE_URL` 指向的 WeKnora **0.8.2 / standard**、空间 `10000` 实测：

| 检查项 | 结果 |
| --- | --- |
| 空间配置 `GET/PUT tenants/kv/memory-config` | **通过**：可读写，`enabled` 切换后能还原 |
| 缺 `X-External-User-Token` 访问 `/memory/settings` | **200**（未拒绝，回退到 API Key 租户主体） |
| 用户 A / B 不同 signed JWT 改 `user_enabled` | **未隔离**：改 A 后读 B 看到相同值 |
| 用户 A / B 写入不同 memory items | **未隔离**：双方列表都能看到对方条目；返回 `subject_id` 均为 `api_tenant:10000`（不是 JWT `sub`） |
| `GET /tenants/10000/api-principal-config` | API Key **403**（需 Owner JWT，本次未能读取 mode） |

结论：当前部署下，**不能**用「共享空间 API Key + 外部用户 Token」安全实现个人记忆；个人功能保持关闭，直至 WeKnora 侧 Principal 对 `/memory/*` 按外部用户生效并可复测通过。

## 5. WeKnora 侧必配

1. API Key 具备 `manage_tenant_settings` 或 `full-access`（空间配置）。  
2. 若使用终端用户主体：空间 `api-principal-config` 设为 `signed_token`，配置与本平台相同的 `hmac_secret`。  
3. 本平台在 **系统配置** 填写 WeKnora 地址、API Key、空间 ID（默认 `10000`）；`.env` 仅作首次种子。  
4. 终端用户令牌仍用环境变量：`WEKNORA_EXTERNAL_USER_HMAC`、`WEKNORA_EXTERNAL_USER_TOKEN_TTL`（密钥不进前端）。

## 6. 智能体 `memory_enabled`

本阶段未集成智能体管理 API。后续若接入，仅在实际 Agent API 支持该字段时写入；空间开关开启不代表所有智能体允许记忆。
