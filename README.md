# 擎知

擎知是一套通用企业级后台管理系统的工程底座。当前阶段包含管理后台、Go API、PostgreSQL / Redis 开发环境，以及统一的配置、日志、错误、健康检查和 OpenAPI 机制。

此阶段没有登录、用户、部门、角色、权限等业务模块，也没有业务数据表。

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

`up` 执行全部未应用的迁移，`down` 只回滚一步。阶段 1 的基线迁移不创建表。

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
