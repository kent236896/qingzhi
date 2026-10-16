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
| 409 | 40900 | 唯一约束或依赖规则冲突 |
| 405 | 40500 | 方法不允许 |
| 413 | 41300 | 请求体超过 `HTTP_MAX_BODY_BYTES` |
| 429 | 42900 | 超过当前进程的客户端 IP 限流 |
| 500 | 50000 | 服务器内部错误 |
| 503 | 50301 | 依赖未就绪 |
| 504 | 50400 | 请求处理超过 `HTTP_REQUEST_TIMEOUT` |

校验失败时 `data.fields` 给出字段和可展示的说明，例如 `name为必填项`。响应不包含堆栈、SQL 或配置。

## Request ID

请求头 `X-Request-ID` 可选。值必须是 8 到 64 个字符，只含字母、数字、点、下划线和中划线。缺失或非法时服务端生成 UUID，并写回响应头和 `request_id`。

## 时间与分页

日期时间使用 UTC 的 RFC 3339，例如 `2026-10-14T03:34:00.000Z`。

列表接口使用查询参数 `page`（从 1 开始）和 `page_size`（默认 20，最大 100）。响应 `data` 包含 `items`、`page`、`page_size` 和 `total`。部门树接口返回整棵树，不使用分页。

## 已实现接口

- `GET /api/v1/health/live`：进程存活。
- `GET /api/v1/health/ready`：PostgreSQL 和 Redis 均为 `up` 时 HTTP 200，否则 HTTP 503。检查结果只有 `up` 或 `down`。
- `GET /api/v1/openapi.json`：返回 OpenAPI 3 文档，不使用统一信封。

账号、租户和部门管理接口同样走 `/api/v1`。除健康检查、OpenAPI 与登录外，管理接口需要 `Authorization: Bearer <token>`，并校验角色菜单权限；租户路径参数必须与令牌中的租户一致。认证失败返回 HTTP 401、业务码 `40100`；无权限返回 HTTP 403、业务码 `40300`。数据库未就绪时返回 HTTP 503、业务码 `50301`。

- `POST /api/v1/auth/login`：用户名、密码、可选 `tenant_id`。成功返回 access token、当前用户、权限列表与数据范围。
- `GET /api/v1/auth/me`、`GET /api/v1/auth/menus`：当前会话与按角色过滤的菜单树。`me` 同时返回系统版本。侧边栏应使用菜单接口，而不是完整菜单管理接口。
- `GET/PATCH /api/v1/auth/profile` 读取或修改当前用户的昵称、备注、邮箱、手机、性别和头像。`PUT /api/v1/auth/password` 用当前密码修改登录密码。这两项只作用于登录者本人。
- `GET/POST /api/v1/tenants`，`GET/PATCH /api/v1/tenants/{tenantId}`
- `GET/POST /api/v1/tenants/{tenantId}/organizations`，`PATCH/DELETE /api/v1/tenants/{tenantId}/organizations/{organizationId}`。每个租户只有一个主体组织。集团和子公司的主体可以同级，也可以把子公司主体挂到集团主体下面；部门必须留在本公司租户内。不能删除主体组织。`GET` 按数据范围过滤。令牌租户可以直接访问；维护其他租户的组织还需要 `system:tenant:query`。
- `GET/POST /api/v1/tenants/{tenantId}/posts`，`GET/PATCH/DELETE /api/v1/tenants/{tenantId}/posts/{postId}`。岗位按租户隔离，同一租户内编码唯一。
- `GET/POST /api/v1/menus`，`GET/PATCH/DELETE /api/v1/menus/{menuId}`。`GET` 返回整棵菜单树，最多使用三级。类型用 `1` 表示目录、`2` 表示菜单、`3` 表示按钮。`permission` 是权限标识。业务菜单下通常挂查询/新增/修改/删除按钮（如 `system:tenant:create`）。角色勾选按钮后，对应用户才能调用对应写接口；无权限返回 HTTP 403、业务码 `40300`。
- `GET/POST /api/v1/dict-types`，`GET/PATCH/DELETE /api/v1/dict-types/{typeId}`。字典类型全局唯一，不按租户隔离。`type` 为 1 到 100 位字母、数字或下划线。修改 `type` 时同步更新其下数据的 `dict_type`。类型下仍有数据时不能删除，删除会写入 `deleted_time`。
- `GET/POST /api/v1/dict-data`，`GET/PATCH/DELETE /api/v1/dict-data/{dataId}`。`GET` 必须带 `dict_type`。同一类型下 `value` 唯一。`color_type` 可为空，或为 `default`、`primary`、`success`、`info`、`warning`、`danger`。
- `GET /api/v1/dicts/{dictType}` 返回该类型下状态正常的数据；类型停用时返回空数组。权限标识为 `system:dict:query|create|update|delete`。
- `GET /api/v1/security/login-logs` 查询登录日志。`result`：`0` 成功、`1` 凭证失败、`2` 锁定。日志不记录密码。连续凭证失败达到安全策略阈值后，窗口期内拒绝登录。
- `GET /api/v1/security/operate-logs` 查询已登录用户的写操作。只保存方法、路径、状态和耗时，不保存请求体。
- `GET /api/v1/security/permission-logs` 查询角色菜单、角色用户、数据权限部门和数据范围的变更前后内容。
- `GET/PATCH /api/v1/security/policy` 读取或更新全局安全策略。密码长度限制在 6 到 32。`login_fail_limit` 为 0 时不锁定。创建和修改账号密码时按该策略校验。权限标识为 `system:login-log:query`、`system:operate-log:query`、`system:permission-log:query`、`system:security-policy:query|update`。
- `GET /api/v1/monitor/overview` 返回 API、数据库和 Redis 连通性，窗口内的请求量、错误、超时和响应时间，CPU、内存、磁盘，当前防护参数，以及未恢复告警。健康检查和监控接口自身不计入该窗口。
- `GET /api/v1/monitor/alerts` 分页查询告警历史，`status` 可取 `firing` 或 `resolved`。条件消失后告警自动变为已恢复。权限标识为 `system:monitor:query`。
- `GET/POST /api/v1/tenants/{tenantId}/roles`，`GET/PATCH/DELETE /api/v1/tenants/{tenantId}/roles/{roleId}`。角色按租户隔离。类型用 `1` 表示系统、`2` 表示自定义。`data_scope`：`1` 全部、`2` 自定义、`3` 本部门、`4` 本部门及以下、`5` 仅本人。
- `GET/PUT /api/v1/tenants/{tenantId}/roles/{roleId}/menus` 覆盖角色菜单；`GET/PUT .../depts` 覆盖自定义数据权限部门；`GET/PUT .../users` 覆盖角色用户。角色用户必须已是该租户成员。
- `GET/POST /api/v1/users`，`GET/PATCH /api/v1/users/{userId}`。创建账号需提供密码；更新时可改密码。响应不包含密码。列表受数据范围约束。
- `GET/POST /api/v1/tenants/{tenantId}/members`，`PUT /api/v1/tenants/{tenantId}/members/{userId}` 覆盖该租户内的组织任职，`DELETE` 移除该租户下的全部任职、租户归属和该租户内的角色，不删除账号。创建和覆盖都必须至少选择一个属于该租户的组织。成员列表受数据范围约束。

状态使用 `0`（正常）和 `1`（停用）。部门、岗位、菜单、角色和字典删除把 `deleted` 标为真。删除部门前不能有下级或任职，删除菜单前不能有下级，删除角色前不能有用户，删除字典类型前不能有字典数据，系统角色不能删除。有成员或部门的租户没有删除接口。

接口变更时同步修改 `services/enterprise-api/api/openapi.json`。
