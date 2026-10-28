# API 复用分析（阶段 03：WeKnora 身份 + AI 对话）

检查基准：`services/enterprise-api` 路由注册、`internal/weknora`、`internal/knowledge`、官方 SDK `github.com/Tencent/WeKnora/client`（模块版本 `v0.0.0-20261017152502-2c0d8384398c`）、`apps/user-workbench`。

| 业务功能 | 现有平台 API | 是否满足需求 | 处理方式 | 影响范围 |
|---|---|---|---|---|
| 用户身份获取 | `GET /api/v1/auth/me`；登录 `POST /api/v1/auth/login`、SSO exchange | 是（平台会话） | 复用 | 工作台 auth store |
| WeKnora 终端用户 Token | `POST /api/v1/weknora/external-user-token`（需 `model:manage`）；签发逻辑在 `ExternalUserTokenIssuer` | 部分（签发可用，但管理端调试接口不适合工作台；SDK 请求未注入 Token） | 优化适配层统一注入；工作台不直接调 issue 接口 | `internal/weknora`、Chat 服务 |
| WeKnora 空间 ID | 系统设置 `weknora_tenant_id` / `WEKNORA_TENANT_ID` | 是 | 复用 `settings.WeKnoraTenantID` | Chat 身份解析 |
| 知识库列表（授权） | `GET /api/v1/knowledge-bases`（需 `knowledge:query`） | 部分（可见性逻辑满足，但权限码可能挡住普通员工；且不暴露 RemoteID 给前端是正确的） | 新增 `GET /api/v1/chat/knowledge-bases`：复用 `knowledge.Service.List` 过滤，登录即可 | Chat 前端选择器 |
| 创建会话 | 无 | 否 | 新增 `POST /api/v1/chat/sessions` → 上游 `POST /api/v1/sessions` | Chat |
| 会话列表 | 无 | 否 | 新增 `GET /api/v1/chat/sessions` → 上游 `GET /api/v1/sessions` | Chat |
| 会话重命名 | 无 | 否 | 新增 `PUT /api/v1/chat/sessions/:id` → 上游 `PUT /api/v1/sessions/:id` | Chat |
| 会话删除 | 无 | 否 | 新增 `DELETE /api/v1/chat/sessions/:id` → 上游 `DELETE /api/v1/sessions/:id` | Chat |
| 历史消息 | 无 | 否 | 新增 `GET /api/v1/chat/sessions/:id/messages` → 上游 `GET /api/v1/messages/{id}/load` | Chat |
| 流式问答 | 无 | 否 | 新增 `POST /api/v1/chat/sessions/:id/ask`（SSE）→ 上游 `POST /api/v1/knowledge-chat/{id}` | Chat |
| 停止生成 | 无 | 否 | 新增 `POST /api/v1/chat/sessions/:id/stop` → 上游 `POST /api/v1/sessions/{id}/stop` | Chat |
| 引用来源 | 无专用平台 API；上游 SSE `references` / 消息 `knowledge_references` | 部分（展示片段可做；安全打开原文无已确认代理） | 复用流/历史中的引用字段展示；原文跳转标 PENDING | Chat UI |
| 会话关键词搜索 | 上游列表无 keyword 参数 | 否（上游无） | 前端本地过滤；不编造上游参数 | Chat UI |

## 新增或优化原因摘要

1. **优化** WeKnora HTTP 适配层：官方 SDK `applyAuthHeaders` 不设置 `X-External-User-Token`，必须在本项目注入，否则无法以终端用户主体调用 Session/QA。
2. **新增** `/api/v1/chat/*`：平台侧完全没有会话/问答代理；不能让浏览器直连 WeKnora 或持有 API Key。
3. **复用** 知识库可见性与 Token 签发，避免第二套授权或身份体系。

## 测试结果（阶段 03）

| 项 | 结果 |
|---|---|
| `go test ./internal/chat/... ./internal/weknora/... ./internal/knowledge/...` | 通过 |
| `apps/user-workbench` `pnpm typecheck` / `lint` / `build` | 通过 |
| 实机 A/B 两用户同 API Key 会话列表/消息互访 | **PENDING**（本环境未完成跨用户实测；不得宣称隔离已通过） |

## PENDING

- 实机验证 `signed_token` 下 Session/消息跨用户隔离。
- 引用「打开原文」安全代理（无确认上游则只展示片段）。
- WeKnora `api-principal-config` 是否已配置为 `signed_token` 且 HMAC 对齐。
