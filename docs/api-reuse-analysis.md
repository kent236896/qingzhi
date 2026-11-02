# API 复用分析

检查基准：`services/enterprise-api` 路由注册、`internal/weknora`、官方 SDK、`apps/user-workbench`、实机 WeKnora（`.env` 中 `WEKNORA_BASE_URL`）。

## 阶段 03：WeKnora 身份 + AI 对话

| 业务功能 | 现有平台 API | 是否满足需求 | 处理方式 | 影响范围 |
|---|---|---|---|---|
| 用户身份获取 | `GET /api/v1/auth/me`；登录 `POST /api/v1/auth/login`、SSO exchange | 是（平台会话） | 复用 | 工作台 auth store |
| WeKnora 终端用户 Token | `POST /api/v1/weknora/external-user-token`（需 `model:manage`）；签发逻辑在 `ExternalUserTokenIssuer` | 部分 | 优化适配层统一注入；工作台不直接调 issue 接口 | `internal/weknora`、Chat 服务 |
| WeKnora 空间 ID | 系统设置 `weknora_tenant_id` / `WEKNORA_TENANT_ID` | 是 | 复用 | Chat / 记忆身份解析 |
| 知识库列表（授权） | `GET /api/v1/knowledge-bases`（需 `knowledge:query`） | 部分 | 新增 `GET /api/v1/chat/knowledge-bases` | Chat |
| 会话 / 流式问答 / 停止 | `/api/v1/chat/*` + 平台 `chat_session_owners` | 是 | 复用（P0 已补平台侧归属） | Chat |

## 阶段 04：个人长期记忆

| 功能 | 现有平台 API | WeKnora 上游（实机） | 能力是否满足 | 处理方式 |
|---|---|---|---|---|
| 获取个人记忆设置 | `GET /api/v1/memory/settings` | `GET /api/v1/memory/settings` | 是（signed_token 隔离） | **代理** |
| 修改个人记忆开关 | `PUT /api/v1/memory/settings` | `PUT /api/v1/memory/settings` | 是 | **代理** |
| 查询个人记忆列表 | `GET /api/v1/memory/items` | 同路径 | 是 | **代理** |
| 新增 / 编辑 / 删除 | `POST/PUT/DELETE .../items` | 同路径 | 是 | **代理** |
| 确认 / 拒绝 | `POST .../confirm|reject` | 同路径 | 是 | **代理** |
| 清空全部记忆 | 无 | `DELETE /api/v1/memory/items` | — | 本阶段不做 |
| 个人能力门禁状态 | `GET /api/v1/memory/status` | — | 是 | **复用**；`ready=true`（2026-10-30） |
| 空间级记忆配置 | `GET/PUT /api/v1/weknora/memory-config` | `tenants/kv/memory-config` | 是 | **复用** |
| WeKnora 管理员账号 | 系统配置 `admin_email` / `admin_password` | `POST /api/v1/auth/login` | 是 | 存服务端；`ensure-principal` 换 JWT 写 signed_token |

### 上游实测字段

`GET /api/v1/memory/settings`：`workspace_enabled`、`user_enabled`、`effective`、`write_mode`、`item_count`、`max_items`。

`POST /api/v1/memory/items`：`subject_id` 在 signed_token 下为 `api_external_user:<空间>:<sub>`。

### 隔离复测

脚本：`scripts/memory-isolation-retest.go`（Step0 需管理员 JWT：`WEKNORA_ADMIN_TOKEN` 或配置页管理员账号）。

| 检查项 | 2026-10-29 | 2026-10-30 |
|---|---|---|
| 缺 Token | 200 回退 | **401** |
| settings 跨用户 | 共享 | **隔离** |
| items 跨用户可见 | 可见 | **不可见** |
| 跨用户删除 | 200 | **404** |
| **RESULT** | FAIL | **PASS** |

## 测试结果摘要

| 项 | 结果 |
|---|---|
| 记忆隔离复测 | **PASS isolation_ok**（2026-10-30） |
| 个人记忆 CRUD 代理 | **已开放**（`ready=true`） |
| 门禁单测 / settings 单测 / 工作台 typecheck | 以当次命令输出为准 |
