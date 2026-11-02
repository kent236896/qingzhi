# 用户工作台（user-workbench）

面向企业普通员工的知识与智能助手前端。与 `apps/admin-web` 管理后台相互独立，共用 `services/enterprise-api` 后端。

当前已完成：布局与路由、统一登录（本地 + Keycloak/SSO）、个人中心（资料与本地密码管理）、知识库管理（个人库增删改分享与文档上传；企业/共享库只读查看与检索）、AI 对话（知识库选择、会话、SSE 流式问答、引用片段）。智能体中心与个人记忆页仍为占位（记忆因隔离未通过暂不开放）。

## 技术栈

| 依赖 | 说明 |
| --- | --- |
| Vue 3 | Composition API + `<script setup>` |
| TypeScript | 严格模式 |
| Vite | 开发与构建 |
| Pinia | 状态管理 |
| Vue Router | 路由 |
| TDesign Vue Next | UI 组件 |
| Axios | JSON HTTP 客户端 |
| fetch + ReadableStream | Chat SSE（Axios 不用于吃流） |
| marked + DOMPurify | 助手消息安全 Markdown 渲染 |
| ESLint / Prettier | 代码规范 |
| pnpm | 包管理 |

版本与管理后台对齐，由本目录 `pnpm-lock.yaml` 锁定。

## 目录结构

```text
user-workbench/
├── src/
│   ├── api/           # auth / sso / chat / knowledge / HTTP 客户端
│   ├── components/    # 布局与个人中心组件
│   ├── layouts/
│   ├── router/
│   ├── stores/        # auth / chat 等
│   ├── types/
│   ├── utils/         # http / markdown / redirect / knowledge
│   └── views/         # login / chat / knowledge / profile / 占位
├── .env.example
├── package.json
└── README.md
```

## 认证说明

- 会话模型：Bearer JWT（`Authorization`），无 Cookie Session；工作台使用 `qz_wb_access_token` / `qz_wb_tenant_id`，与管理后台隔离。
- 本地登录：`POST /api/v1/auth/login`。
- SSO：`GET /api/v1/sso/login-options` → `.../oidc/start`（或 SAML）→ 回调 `/login/sso/callback` → `POST /api/v1/sso/oidc/exchange`。
- 退出：后端无 logout API；前端清除本地 JWT。联邦 IdP 登出未配置账号管理 URL 时不会拼造 Keycloak 地址。
- 本地密码修改：`PUT /api/v1/auth/password`，只改本平台哈希，**不会**改 Keycloak 密码。仅 `local_password_enabled=true` 的账号可改。
- 用户名：当前后端不允许用户自行修改。

## 知识库

- 路由：`/knowledge` 列表，`/knowledge/:id` 详情。
- 复用平台 `/api/v1/knowledge-bases*`（需 `knowledge:query`）。
- **我的知识库**：仅本人个人库；可新建 / 修改 / 删除 / 分享给用户 / 上传与删除文档。
- **企业知识库**：组织 / 岗位 / 角色库，以及别人分享给我的个人库；仅查看、Wiki 检索与「对话检索」，无改删分享上传。
- 后端 `CanWrite` 已保证组织库不可被普通用户写；前端隐藏操作作为补充。

## AI 对话

- 登录后默认进入 `/chat`（`/` 重定向到 `/chat`）。
- 平台代理：`/api/v1/chat/*`（见 [docs/api-conventions.md](../../docs/api-conventions.md)）。浏览器不持有 WeKnora API Key / HMAC。
- 知识库选择器只展示平台本地 id；服务端映射远程 ID 并做可见性校验。
- 从知识库详情可带 `?kb=` 进入对话并预选该库。
- 流式问答：`POST .../ask` 经 `fetch` SSE；停止生成需使用流事件中的 `message_id`。
- 引用面板展示检索片段与元数据；无安全原文代理时不生成永久公开链接。
- 会话标题搜索为前端本地过滤（上游列表无 keyword）。

### 人工配置（WeKnora）

1. WeKnora `api-principal-config.mode=signed_token`。
2. HMAC 与企业 API 的 `WEKNORA_EXTERNAL_USER_HMAC` 一致。
3. 系统设置中填写 WeKnora Base URL、API Key、`weknora_tenant_id`。
4. CORS 已包含工作台 Origin（如 `http://127.0.0.1:5174`）。

Session 跨用户隔离需实机 A/B 验证，结论见 [docs/api-reuse-analysis.md](../../docs/api-reuse-analysis.md)。

## 开发环境要求

- Node.js（见仓库根 README）
- pnpm `11.20.0`
- 本地或可访问的 `enterprise-api`（默认 `http://127.0.0.1:8080`）
- CORS 需包含工作台 Origin（如 `http://127.0.0.1:5174`）

## 安装与命令

```powershell
cd apps\user-workbench
pnpm install
Copy-Item .env.example .env   # 按需
pnpm dev                      # http://127.0.0.1:5174
pnpm typecheck
pnpm lint
pnpm build
```

| 变量 | 说明 |
| --- | --- |
| `VITE_API_BASE_URL` | API 根地址；本地可留空以走 Vite `/api` 代理。 |

`VITE_` 变量会进入浏览器产物，禁止存放密钥。Keycloak Client Secret 只存在服务端。

## 路由

| 路径 | 说明 |
| --- | --- |
| `/login` | 登录（本地 + SSO） |
| `/login/sso/callback` | SSO 票据换 JWT（后端硬编码路径） |
| `/` | 重定向到 `/chat` |
| `/chat` | AI 对话（需登录） |
| `/home` | 模块导览 |
| `/agents` | 业务占位（需登录） |
| `/memory` | 我的长期记忆（门禁状态页；隔离未通过时不开放 CRUD） |
| `/profile` | 个人中心 |

## 尚未实现

- 智能体中心
- 个人记忆 CRUD（隔离复测未通过，仅状态门禁页）
- 平台用户名自助修改
- Keycloak 账号控制台 URL 配置与联邦登出
- 本地凭证与 SSO 身份的绑定/初始化流程
- 引用「打开原文」安全代理
