# 架构

擎知是一套通用企业后台的工程底座。当前阶段只有基础设施，没有组织、用户、权限等业务模块。

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

PostgreSQL 使用 GORM 连接池。阶段 1 的迁移 `000001_baseline` 不创建表，只验证迁移机制。Redis 只做连接检查，没有缓存或队列逻辑。
