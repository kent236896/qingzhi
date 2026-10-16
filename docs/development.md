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

`up` 应用全部未执行迁移。`down` 只回滚最近一步。`version` 查看当前版本。`000001_baseline` 不创建数据表。`000002_identity` 创建账号、租户和部门表。`000003_sys_prefix` 给这些表加上 `sys_` 前缀并调整字段。`000004_sys_posts` 创建岗位表 `sys_posts`。`000005_sys_menus` 创建菜单表 `sys_menus`。`000006_sys_menus_seed` 写入默认的三级菜单。`000007_sys_roles` 创建角色及相关关联表。`000008_auth_data_scope` 增加角色数据范围、`sys_role_depts`，并初始化管理员（`admin` / `Admin@123456`，需库中已有租户）。`000009_menu_crud_buttons` 在业务菜单下写入增删改查按钮权限，并同步给已拥有父菜单的角色。`000010_sys_dicts` 创建 `sys_dict_types`、`sys_dict_data`，并给已拥有「系统管理」的角色挂上字典菜单。`000011_sys_security` 创建登录日志、操作审计、权限变更和安全策略，并给已拥有「系统管理」的角色挂上安全审计菜单。`000012_sys_monitor` 创建 `sys_monitor_alerts`，并给已拥有「系统管理」的角色挂上系统监控菜单。`000013_menu_names` 把业务菜单改成四个字，并把「首页」调整为一级菜单。`000014_organization_root` 为每个租户约束唯一主体组织，并给缺少组织任职的成员补上该主体组织。若某个租户已经有多个主体组织，这一步会失败，需要先人工整理。`000015_subsidiary_root` 标记每个租户的主体组织，子公司主体可以挂在集团主体下面。配置需提供 `JWT_SECRET`（至少 16 位）与可选 `JWT_EXPIRE`（默认 `24h`）。可选 `HTTP_REQUEST_TIMEOUT`、`HTTP_MAX_BODY_BYTES`、`HTTP_RATE_LIMIT_RPS`、`HTTP_RATE_LIMIT_BURST` 控制请求防护，`MONITOR_WINDOW` 与 `MONITOR_*` 控制告警阈值。

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
