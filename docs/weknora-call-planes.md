# WeKnora 管理面 / 用户面调用矩阵

Qingzhi 经服务端调用 WeKnora 时必须二选一，**禁止**在同一会话/对话请求上混用 Admin Bearer 与 External Token（否则会话会记到管理员，后续 Ask 404）。完整盘点见 [agent-mcp-phase1-inventory.md](agent-mcp-phase1-inventory.md)。

## 鉴权头

| 平面 | 代码入口 | 头 |
| --- | --- | --- |
| 管理面 | `ExchangeAdmin` / `*RemoteAgent` CRUD、`PutRemoteMCPCredentials`、MCP test | `X-API-Key` + `Authorization: Bearer`（租户管理员 JWT） |
| 用户面 | `Exchange` / `ExchangeUser`、session、knowledge-chat、agent-chat、个人记忆、MCP OAuth | `X-API-Key` + `X-External-User-Token`（`signed_token`） |

`Exchange` / `ExchangeUser` 在发请求前执行 `AssertUserPlaneAllowed`：拒绝 Agent/MCP **定义写**与 **shared credentials** 写删。`ExchangeUser` 额外要求 ctx 已注入 External Token。

## 路径矩阵（Qingzhi 封装约定）

| 能力 | WeKnora 路径 | 平面 | 封装 |
| --- | --- | --- | --- |
| Agent 列表 / 读 / 建 / 改 / 删 / 复制 | `/api/v1/agents*` | **管理面** | `ListRemoteAgents`、`Get/Create/Update/Delete/CopyRemoteAgent` |
| MCP 建 / 改 / 删 / test | `/api/v1/mcp-services*` | **管理面** | `Create/Update/Delete/TestRemoteMCPService` |
| MCP 共享凭证 | `.../credentials` | **管理面** | `PutRemoteMCPCredentials`、`DeleteRemoteMCPCredential` |
| MCP OAuth 状态 / 授权 URL / 撤 token | `.../oauth/status`、`.../oauth/authorize-url`、`DELETE .../oauth/token` | **用户面** | `GetRemoteMCPOAuthStatus`、`CreateRemoteMCPAuthorizeURL`、`RevokeRemoteMCPOAuthToken` |
| 会话 / knowledge-chat / agent-chat | `/api/v1/sessions*`、`/knowledge-chat/*`、`/agent-chat/*` | **用户面** | `chat_client.go`；Ask：有 owners.`agent_id` → `OpenAgentChatStream`，否则 `OpenKnowledgeChatStream` + `agent_enabled=false` |
| 个人记忆 | `/api/v1/memory/*` | **用户面** | `memory_*.go` |
| 空间 principal / 记忆配置 KV | `api-principal-config`、`tenants/kv/*` | **管理面** | `EnsureSignedTokenPrincipal`、`memory_config.go` |

## 用户面硬拒绝（即使误用 Ext 也不得从 Qingzhi 发出）

- `POST/PUT/PATCH/DELETE /api/v1/agents...`（含 `.../copy`）
- `POST/PUT/PATCH/DELETE /api/v1/mcp-services` 与 `.../{id}` 定义变更、`.../test`、`.../metadata/refresh`
- `PUT/PATCH/POST/DELETE .../credentials*`
- **任意方法** `.../tool-approvals`、`.../tools/approvals`（第六阶段起不做工具审批代理，平台不封装、用户面 `Exchange` 直接拒绝）

允许：`GET` 诊断（除 approvals）、用户 OAuth、session/chat/memory。

企业 ACL 真相仍是平台 `sys_agents` / `sys_mcp_*` + `scoped`；WeKnora 列表对 Ext 用户不提供可靠隔离。

## MCP `auth_mode`

| 值 | 含义 | 写凭证 |
| --- | --- | --- |
| `shared`（默认） | 空间级静态凭证 | 仅管理面 |
| `user_oauth` | 每平台用户 OAuth | 用户面 authorize / status / revoke；**不做**个人静态 API Key Gateway |

平台目录 `sys_mcp_services.auth_mode` / `url` 与 `sys_agents.config_json` 已由迁移 `000033` 持久化；创建/更新经管理面同步 WeKnora。

## 用户 OAuth 时序（第四阶段）

1. 工作台对可见且 `auth_mode=user_oauth` 的 MCP 调 `POST /api/v1/toolbox/mcp/{id}/oauth/authorize-url`。  
2. Qingzhi 用 External Token 调 WeKnora `authorize-url`，`redirect_uri` = WeKnora 自有 `/api/v1/mcp-oauth/callback`。  
3. 浏览器打开返回的 `authorization_url`；第三方授权后回调 **WeKnora**（不经 Qingzhi）。  
4. WeKnora 写 per-principal token 后重定向到 `frontend_redirect`（工作台 `/toolbox`，可能带 `#mcp_oauth_result=`）。  
5. 工作台轮询 `GET .../oauth/status?authorization_attempt=` 或刷新列表；撤销走 `DELETE .../oauth/token`。
