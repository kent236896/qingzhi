# 开发

命令默认在仓库根目录执行。Windows 使用 PowerShell 脚本，不依赖 GNU Make。已安装 Make 的环境也可以使用根目录 `Makefile`。

前端 TypeScript 使用 5.9。不要把它升到 7.x：当前 `vue-tsc` 3.3 和 `typescript-eslint` 8 与 TypeScript 7 不兼容。后端 `go.mod` 保持 Go 1.21，运行 Go 命令时建议设置 `GOTOOLCHAIN=local`，避免自动下载更新的工具链。

## 环境变量

```powershell
Copy-Item .env.example .env
```

`.env` 已被 Git 忽略。示例密码 `change-me` 只用于本机。生产环境必须更换密码，并且 `POSTGRES_SSLMODE` 不能是 `disable`。

随附的 Compose 文件启动的 Redis 没有密码，因此本地 `REDIS_PASSWORD` 留空。

后端从当前目录的 `.env`、上一级或上两级 `.env` 读取变量，且不覆盖已经存在的进程环境变量。设置空的 `ENV_FILE` 可以禁止读取文件。

## 基础设施

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\infra-up.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\infra-logs.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\infra-down.ps1
```

PostgreSQL 映射到 `127.0.0.1` 上的 `POSTGRES_PORT`（默认 5432），Redis 映射到 `REDIS_PORT`（默认 6379）。数据保存在 Compose 卷 `postgres_data` 和 `redis_data` 中。

Nginx 骨架不随默认启动：

```powershell
docker compose --env-file .env --profile edge -f deploy/docker-compose.yml up -d
```

它监听 `127.0.0.1` 的 `NGINX_PORT`（默认 8088）。`/nginx-health` 返回纯文本 `ok`。

## 数据库迁移

在基础设施就绪后执行：

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\migrate.ps1 up
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\migrate.ps1 version
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\migrate.ps1 down
```

`up` 应用全部未执行迁移。`down` 只回滚最近一步。`version` 查看当前版本。阶段 1 的基线迁移不创建数据表。

## 启动应用

两个进程分两个终端启动。

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\api-dev.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\web-dev.ps1
```

后端监听 `http://127.0.0.1:8080`。前端开发服务器是 `http://127.0.0.1:5173`，并把 `/api` 代理到后端。

也可以进入各自目录：

```powershell
cd services\enterprise-api
go run ./cmd/server
```

```powershell
cd apps\admin-web
pnpm dev
```

## 检查

```powershell
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\check.ps1
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\check-infra.ps1
```

`check.ps1` 运行 Go 测试、`go vet`、前端类型检查、ESLint、Vitest 和生产构建。`check-infra.ps1` 需要本机已安装 Docker，并验证 Compose 配置。
