# 架构

擎知是一套通用企业后台。当前包含工程底座，以及账号、租户、部门组织、角色、字典、JWT 登录鉴权、个人资料与密码、数据权限、操作审计、登录日志、权限变更记录、安全策略、系统监控和异常告警。

## 工程边界

| 目录 | 职责 |
| --- | --- |
| `apps/admin-web` | Vue 3 管理后台。独立安装、启动、构建和测试。 |
| `services/enterprise-api` | Go API。独立启动和测试。 |
| `deploy` | 本地 PostgreSQL、Redis 和 Nginx 骨架。 |
| `docs` | 架构、接口、开发和测试约定。 |
| `scripts` | Windows 下可直接执行的开发命令。 |

前端通过 HTTP 调用后端。浏览器不连接 PostgreSQL 或 Redis。数据库结构只由 golang-migrate 的 SQL 文件变更。

## 请求路径

开发时，Vite 把 `/api` 代理到 `http://127.0.0.1:8080`。后端路由前缀是 `/api/v1`。

可选的 Nginx 配置位于 `deploy/nginx`，使用 Compose profile `edge`。它把 `/api/` 转到宿主机的 8080 端口，把其他路径转到 Vite 的 5173 端口。

## 后端组成

`cmd/server` 读取配置并组装日志、PostgreSQL、Redis 和 HTTP 服务。依赖暂时不可达时进程仍然启动，`/health/live` 返回成功，`/health/ready` 返回 503。

日志是 JSON，写到标准输出。访问日志和应用日志用 `component` 区分。

## 数据

PostgreSQL 使用 GORM 连接池。`000001_baseline` 不创建表。`000002_identity` 创建身份表，`000003_sys_prefix` 把它们改成 `sys_users`、`sys_tenants`、`sys_organizations`、`sys_tenant_memberships`、`sys_organization_memberships`。`000004_sys_posts` 增加租户内的岗位表 `sys_posts`。`000005_sys_menus` 增加系统菜单表 `sys_menus`，用 `parent_id` 组成最多三级的树。`000006_sys_menus_seed` 初始化「系统管理」和「基础设施」两个一级目录，并把已开发页面挂在「系统管理」下。`000013_menu_names` 把这些页面改成四个字的名称，并把「首页」提升为单独的一级菜单。`000014_organization_root` 保证每个未删除租户只有一个主体组织；没有主体组织的租户会补一条，已有多个主体组织时迁移会中止。没有组织任职的既有成员会补挂到该主体组织，不删除账号。`000015_subsidiary_root` 用 `is_tenant_root` 标记主体组织，使子公司主体可以挂在集团主体下面。`000007_sys_roles` 增加租户角色表 `sys_roles`、角色菜单表 `sys_role_menus`、角色用户表 `sys_user_roles`，并挂上「角色」菜单。`000008_auth_data_scope` 为角色增加 `data_scope`、创建 `sys_role_depts`，并在已有租户下初始化管理员账号与超级管理员角色。`000009_menu_crud_buttons` 在业务菜单下增加增删改查按钮权限，接口按 `query/create/update/delete` 鉴权。`000010_sys_dicts` 增加全局字典类型表 `sys_dict_types` 和字典数据表 `sys_dict_data`，并挂上「字典」菜单。`000011_sys_security` 增加登录日志、操作审计、权限变更记录和单行安全策略，并挂上「安全审计」目录。`000012_sys_monitor` 增加告警表 `sys_monitor_alerts`，并挂上「系统监控」。请求指标保留在进程内存中；告警写入 PostgreSQL。CPU、内存和磁盘由 `gopsutil` 采集。限流、请求超时和请求体大小限制由 API 进程执行，不依赖 Redis。组织数据用 `tenant_id` 隔离。字典和安全策略不按租户隔离。`000016_user_posts` 增加 `sys_user_posts`，记录人员在租户内的岗位任职。`000017_knowledge` 增加知识库、租户发布、个人分享和文件镜像，并挂上一级菜单「知识管理」。知识库通过官方 Go SDK `github.com/Tencent/WeKnora/client` 对接 WeKnora。创建知识库时的索引与 Wiki 配置、Wiki 读写仍走 SDK 的原始请求，因为类型化创建参数不包含这些字段，SDK 也没有 Wiki 方法。个人知识库由具备查询权限的用户创建并分享给指定用户；组织、岗位和角色知识库需要 `knowledge:manage`。可见范围是本部门及以下、岗位和角色的并集，并且必须与知识库发布的租户有交集。登录仍使用 JWT，不引入 Redis 会话。操作审计不保存请求体。人员身份在 `sys_users` 中全局唯一。一个租户对应一个主体组织，主体组织下可以有多级部门，集团和子公司各有一个主体组织，部门留在本公司租户内；子公司主体可以挂在集团主体下面。人员通过 `sys_organization_memberships` 在多个组织任职，也可以跨租户任职；`sys_tenant_memberships` 表示由这些任职推导出的租户归属，角色仍挂在这条归属上。新建任职必须选择该租户内的组织。移除某租户的最后一条任职时，删除该租户归属和该租户内的角色，不删除人员身份，其他租户任职保持不变。在每个租户里最多有一个主部门。状态用 `0` 表示正常、`1` 表示停用，删除标记是 `deleted`。密码以 bcrypt 存入 `sys_users.password`，接口响应不返回密码。登录签发 HS256 JWT（不经 Redis 会话）；除健康检查、OpenAPI 与登录外，管理接口需 Bearer Token，并按角色菜单权限与数据范围过滤。侧边栏读取 `/api/v1/auth/menus`。Redis 只做连接检查。

`000018_model_bindings` 增加 `sys_weknora_model_bindings`，只保存场景默认模型 ID。模型内容和密钥仍在 WeKnora。

数据库未打开时，身份与组织接口仍然注册，并返回 HTTP 503、业务码 `50301`。领域规则放在 `internal/identity`，鉴权中间件在 `internal/auth`，由 `internal/server` 注入。
