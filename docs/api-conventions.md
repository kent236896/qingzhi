# API 约定

普通 JSON API 使用统一信封。文件下载、流式响应和 OpenAPI 文档本身可以不使用该信封。

## 版本与路径

- 当前版本前缀：`/api/v1`。
- 路径使用小写名词，多个单词用中划线，例如 `/api/v1/health/live`。
- 不在路径里放动词。`GET` 读取，`POST` 创建，`PUT` 整体替换，`PATCH` 局部更新，`DELETE` 删除。
- `GET` 不产生副作用。

## 成功响应

```json
{
  "code": 0,
  "message": "ok",
  "data": {},
  "request_id": "example-request-id"
}
```

`code` 是业务码，与 HTTP 状态码不是同一个数字。

## 错误响应

| HTTP | code | 含义 |
| --- | --- | --- |
| 400 | 40000 | 字段校验失败 |
| 400 | 40001 | 请求体不是合法 JSON，或其他无法解析的请求 |
| 404 | 40400 | 资源不存在 |
| 405 | 40500 | 方法不允许 |
| 500 | 50000 | 服务器内部错误 |
| 503 | 50301 | 依赖未就绪 |

校验失败时 `data.fields` 给出字段和可展示的说明，例如 `name为必填项`。响应不包含堆栈、SQL 或配置。

## Request ID

请求头 `X-Request-ID` 可选。值必须是 8 到 64 个字符，只含字母、数字、点、下划线和中划线。缺失或非法时服务端生成 UUID，并写回响应头和 `request_id`。

## 时间与分页

日期时间使用 UTC 的 RFC 3339，例如 `2026-10-14T03:34:00.000Z`。

以后的列表接口使用查询参数 `page`（从 1 开始）和 `page_size`（默认 20，最大 100）。响应 `data` 包含 `items`、`page`、`page_size` 和 `total`。当前没有分页接口。

## 已实现接口

- `GET /api/v1/health/live`：进程存活。
- `GET /api/v1/health/ready`：PostgreSQL 和 Redis 均为 `up` 时 HTTP 200，否则 HTTP 503。检查结果只有 `up` 或 `down`。
- `GET /api/v1/openapi.json`：返回 OpenAPI 3 文档，不使用统一信封。

接口变更时同步修改 `services/enterprise-api/api/openapi.json`。
