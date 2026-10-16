# 擎知

擎知是一套通用企业级后台管理系统的工程底座。当前阶段包含管理后台、Go API、PostgreSQL / Redis 开发环境，以及统一的配置、日志、错误、健康检查和 OpenAPI 机制。

当前可以维护账号、租户、部门、岗位、菜单、角色和字典，并支持 JWT 登录、菜单授权、数据权限、操作审计、登录日志、权限变更记录、安全策略、系统监控和异常告警。页头账号菜单可以查看系统版本，并修改当前登录人的个人资料和密码。默认管理员（迁移后、且库中已有租户）：`admin` / `Admin@123456`。

## 环境

已在本机核对的工具：

| 工具 | 版本 |
| --- | --- |
| Node.js | 25.8.0 |
| pnpm | 11.20.0 |
| Go | 1.21.5 |

Docker、GNU Make、本机 PostgreSQL 和 Redis 不是编译前端或后端单元测试的前提。启动数据库和 Redis 需要 Docker。

不要为了这个仓库升级全局 Node.js、pnpm 或 Go。后端保持 Go 1.21。前端 TypeScript 锁定在 5.9，因为当前的 `vue-tsc` 和 `typescript-eslint` 不能使用 TypeScript 7。

## 技术版本

前端（`apps/admin-web`，由 `pnpm-lock.yaml` 锁定）：

| 依赖 | 版本 |
| --- | --- |
| Vue | 3.5.43 |
| TypeScript | 5.9.3 |
| Vite | 8.3.4 |
| Pinia | 4.0.3 |
| Vue Router | 5.4.0 |
| TDesign Vue Next | 1.20.9 |
| TDesign Icons Vue Next | 0.4.11 |
| Axios | 1.20.0 |
| ESLint | 10.12.0 |
| Prettier | 3.9.10 |
| Vitest | 5.0.4 |

后端（`services/enterprise-api`，由 `go.sum` 锁定）：

| 依赖 | 版本 |
| --- | --- |
| Gin | 1.10.0 |
| gin-contrib/cors | 1.7.2 |
| GORM | 1.25.12 |
| GORM PostgreSQL driver | 1.5.11 |
| go-redis | 9.7.0 |
| golang-migrate | 4.17.1 |
| validator | 10.20.0 |
| 结构化日志 | Go 1.21 `log/slog` |

开发环境镜像：`postgres:16-alpine`、`redis:7-alpine`、`nginx:1.27-alpine`。

## 首次准备

在仓库根目录执行：

```powershell
Copy-Item .env.example .env
cd apps\admin-web
pnpm install
cd ..\..
```

`.env` 已被 Git 忽略。`change-me` 只是本机示例密码。

## 启动与停止

基础设施：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\infra-up.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\infra-logs.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\infra-down.ps1
```

PostgreSQL 和 Redis 只绑定 `127.0.0.1`。数据在 Compose 卷中。Nginx 使用 profile `edge`，说明见 [docs/development.md](docs/development.md)。

数据库迁移：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\migrate.ps1 up
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\migrate.ps1 version
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\migrate.ps1 down
```

`up` 执行全部未应用的迁移，`down` 只回滚一步。基线迁移不创建表；`000002_identity` 创建账号、租户和部门表，`000003_sys_prefix` 加上 `sys_` 前缀，`000004_sys_posts` 创建岗位表，`000005_sys_menus` 创建菜单表，`000006_sys_menus_seed` 写入默认菜单，`000007_sys_roles` 创建角色表，`000008_auth_data_scope` 增加数据范围并初始化管理员，`000009_menu_crud_buttons` 增加菜单增删改查按钮权限，`000010_sys_dicts` 创建字典类型表、字典数据表和「字典」菜单，`000011_sys_security` 创建登录日志、操作审计、权限变更记录和安全策略，`000012_sys_monitor` 创建告警表和「系统监控」菜单，`000013_menu_names` 把业务菜单改成四个字，并把「首页」调整为一级菜单。`000014_organization_root` 约束每个租户只有一个主体组织。配置需要 `JWT_SECRET`。请求防护和告警阈值见 `.env.example` 中的 `HTTP_*` 与 `MONITOR_*`。系统 CPU、内存和磁盘通过 `github.com/shirou/gopsutil/v3` 采集，因为 Go 标准库没有可移植的这些指标。限流只作用于当前 API 进程。

后端和前端分两个终端启动：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\api-dev.ps1
```

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\web-dev.ps1
```

也可以进入各自目录：

```powershell
cd services\enterprise-api
go run ./cmd/server
```

```powershell
cd apps\admin-web
pnpm dev
```

- 后端：<http://127.0.0.1:8080>
- 前端：<http://127.0.0.1:5173>
- 存活检查：`GET /api/v1/health/live`
- 就绪检查：`GET /api/v1/health/ready`
- OpenAPI：`GET /api/v1/openapi.json`

前端开发服务器把 `/api` 代理到后端。首页会请求这两个健康检查。

## 检查与测试

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\check.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\check-infra.ps1
```

`check.ps1` 运行 Go 测试、`go vet`、前端类型检查、ESLint、Vitest 和生产构建。`check-infra.ps1` 需要 Docker，用来验证 Compose 配置；Docker 不存在时命令会失败，这不代表单元测试失败。

已安装 Make 时，可用根目录 `Makefile` 中的 `infra-up`、`infra-down`、`api-dev`、`web-dev`、`migrate-up`、`test` 和 `compose-config`。Windows 上不安装 Make 也能完成上述 PowerShell 流程。

## 文档

- [架构](docs/architecture.md)
- [API 约定](docs/api-conventions.md)
- [开发](docs/development.md)
- [测试](docs/testing.md)
