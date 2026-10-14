# 测试

## 后端

在 `services/enterprise-api` 执行：

```powershell
go test ./...
go vet ./...
```

覆盖范围：

- 环境变量和 `.env` 加载，包括进程环境优先、缺少必填项、生产环境拒绝示例密码。
- 成功信封、内部错误信封、字段校验和非法 JSON。
- 存活检查、就绪成功、依赖失败时的 503、Request ID、404 和 CORS。
- panic 只记录一次，响应中没有堆栈。
- PostgreSQL 和 Redis 连接失败时，返回文本和诊断文本都不含密码。
- 迁移命令在数据库不可达时失败，且错误不含密码。
- 基线 SQL 不包含 `CREATE TABLE`。
- OpenAPI 路径与已注册路由一致。

这些测试不要求 Docker。它们对关闭的端口发起真实连接，用来确认失败被收敛成安全错误。

## 前端

在 `apps/admin-web` 执行：

```powershell
pnpm typecheck
pnpm lint
pnpm test
pnpm build
```

Vitest 覆盖 HTTP 客户端的成功解包、Request ID、超时、网络错误、服务端安全消息，以及校验和未就绪响应。

## 基础设施

Docker、真实 PostgreSQL 和真实 Redis 的启动与迁移成功，需要本机安装 Docker 后再执行 `scripts/check-infra.ps1`、`scripts/infra-up.ps1` 和 `scripts/migrate.ps1`。不要把未执行的基础设施检查记为通过。
