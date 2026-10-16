CREATE TABLE sys_login_logs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    username varchar(64) NOT NULL DEFAULT '',
    user_id varchar(64) NOT NULL DEFAULT '',
    tenant_id varchar(64) NOT NULL DEFAULT '',
    result smallint NOT NULL,
    message varchar(200) NOT NULL DEFAULT '',
    ip varchar(64) NOT NULL DEFAULT '',
    user_agent varchar(512) NOT NULL DEFAULT '',
    create_time timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT sys_login_logs_result_check CHECK (result IN (0, 1, 2))
);

CREATE INDEX sys_login_logs_username ON sys_login_logs (username, create_time DESC);
CREATE INDEX sys_login_logs_time ON sys_login_logs (create_time DESC);

COMMENT ON TABLE sys_login_logs IS '登录日志';
COMMENT ON COLUMN sys_login_logs.result IS '结果（0成功 1凭证失败 2锁定）';
COMMENT ON COLUMN sys_login_logs.message IS '结果说明';
COMMENT ON COLUMN sys_login_logs.ip IS '登录 IP';
COMMENT ON COLUMN sys_login_logs.user_agent IS '客户端';

CREATE TABLE sys_operate_logs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id varchar(64) NOT NULL DEFAULT '',
    username varchar(64) NOT NULL DEFAULT '',
    tenant_id varchar(64) NOT NULL DEFAULT '',
    module varchar(64) NOT NULL DEFAULT '',
    action varchar(32) NOT NULL DEFAULT '',
    request_method varchar(10) NOT NULL DEFAULT '',
    request_url varchar(255) NOT NULL DEFAULT '',
    ip varchar(64) NOT NULL DEFAULT '',
    user_agent varchar(512) NOT NULL DEFAULT '',
    status_code integer NOT NULL DEFAULT 0,
    duration_ms integer NOT NULL DEFAULT 0,
    request_id varchar(64) NOT NULL DEFAULT '',
    create_time timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX sys_operate_logs_time ON sys_operate_logs (create_time DESC);
CREATE INDEX sys_operate_logs_user ON sys_operate_logs (username, create_time DESC);

COMMENT ON TABLE sys_operate_logs IS '操作审计';
COMMENT ON COLUMN sys_operate_logs.module IS '业务模块';
COMMENT ON COLUMN sys_operate_logs.action IS '动作';
COMMENT ON COLUMN sys_operate_logs.request_url IS '请求路径，不含查询串和请求体';
COMMENT ON COLUMN sys_operate_logs.status_code IS 'HTTP 状态码';
COMMENT ON COLUMN sys_operate_logs.duration_ms IS '耗时毫秒';

CREATE TABLE sys_permission_logs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id varchar(64) NOT NULL DEFAULT '',
    operator_id varchar(64) NOT NULL DEFAULT '',
    operator_name varchar(64) NOT NULL DEFAULT '',
    role_id varchar(64) NOT NULL DEFAULT '',
    role_name varchar(64) NOT NULL DEFAULT '',
    change_type varchar(32) NOT NULL,
    summary varchar(500) NOT NULL DEFAULT '',
    before_text text NOT NULL DEFAULT '',
    after_text text NOT NULL DEFAULT '',
    create_time timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT sys_permission_logs_type_check CHECK (change_type IN ('menus', 'users', 'depts', 'data_scope'))
);

CREATE INDEX sys_permission_logs_time ON sys_permission_logs (create_time DESC);
CREATE INDEX sys_permission_logs_role ON sys_permission_logs (role_id, create_time DESC);

COMMENT ON TABLE sys_permission_logs IS '权限变更记录';
COMMENT ON COLUMN sys_permission_logs.change_type IS '变更类型（menus 菜单 users 用户 depts 部门 data_scope 数据范围）';
COMMENT ON COLUMN sys_permission_logs.before_text IS '变更前';
COMMENT ON COLUMN sys_permission_logs.after_text IS '变更后';

CREATE TABLE sys_security_policy (
    id smallint PRIMARY KEY,
    password_min_length integer NOT NULL DEFAULT 6,
    password_max_length integer NOT NULL DEFAULT 32,
    password_require_letter boolean NOT NULL DEFAULT false,
    password_require_number boolean NOT NULL DEFAULT false,
    password_require_symbol boolean NOT NULL DEFAULT false,
    login_fail_limit integer NOT NULL DEFAULT 0,
    login_lock_minutes integer NOT NULL DEFAULT 15,
    updater varchar(64) NOT NULL DEFAULT '',
    update_time timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT sys_security_policy_singleton CHECK (id = 1),
    CONSTRAINT sys_security_policy_password_check CHECK (
        password_min_length BETWEEN 6 AND 32
        AND password_max_length BETWEEN 6 AND 32
        AND password_max_length >= password_min_length
    ),
    CONSTRAINT sys_security_policy_lock_check CHECK (
        login_fail_limit BETWEEN 0 AND 20
        AND login_lock_minutes BETWEEN 1 AND 1440
    )
);

INSERT INTO sys_security_policy (id) VALUES (1);

COMMENT ON TABLE sys_security_policy IS '安全策略';
COMMENT ON COLUMN sys_security_policy.password_min_length IS '密码最小长度';
COMMENT ON COLUMN sys_security_policy.password_max_length IS '密码最大长度';
COMMENT ON COLUMN sys_security_policy.password_require_letter IS '密码必须包含字母';
COMMENT ON COLUMN sys_security_policy.password_require_number IS '密码必须包含数字';
COMMENT ON COLUMN sys_security_policy.password_require_symbol IS '密码必须包含符号';
COMMENT ON COLUMN sys_security_policy.login_fail_limit IS '连续失败锁定阈值，0 表示不锁定';
COMMENT ON COLUMN sys_security_policy.login_lock_minutes IS '锁定统计窗口（分钟）';

INSERT INTO sys_menus (
    id, name, permission, type, sort, parent_id, path, icon,
    component, component_name, status, visible, keep_alive, always_show
) VALUES
    (
        'a1000000-0000-4000-8000-000000000003', '安全审计', '', 1, 30, NULL, '', '#',
        NULL, NULL, 0, true, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000019', '操作审计', 'system:operate-log:query', 2, 10,
        'a1000000-0000-4000-8000-000000000003', '/audit/operates', '#',
        'views/OperateLogsView.vue', 'OperateLogsView', 0, true, true, true
    ),
    (
        'a1000000-0000-4000-8000-00000000001a', '登录日志', 'system:login-log:query', 2, 20,
        'a1000000-0000-4000-8000-000000000003', '/audit/logins', '#',
        'views/LoginLogsView.vue', 'LoginLogsView', 0, true, true, true
    ),
    (
        'a1000000-0000-4000-8000-00000000001b', '权限变更', 'system:permission-log:query', 2, 30,
        'a1000000-0000-4000-8000-000000000003', '/audit/permissions', '#',
        'views/PermissionLogsView.vue', 'PermissionLogsView', 0, true, true, true
    ),
    (
        'a1000000-0000-4000-8000-00000000001c', '安全策略', 'system:security-policy:query', 2, 40,
        'a1000000-0000-4000-8000-000000000003', '/audit/policy', '#',
        'views/SecurityPolicyView.vue', 'SecurityPolicyView', 0, true, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000219', '查询', 'system:operate-log:query', 3, 1,
        'a1000000-0000-4000-8000-000000000019', '', '#', NULL, NULL, 0, false, true, true
    ),
    (
        'a1000000-0000-4000-8000-00000000021a', '查询', 'system:login-log:query', 3, 1,
        'a1000000-0000-4000-8000-00000000001a', '', '#', NULL, NULL, 0, false, true, true
    ),
    (
        'a1000000-0000-4000-8000-00000000021b', '查询', 'system:permission-log:query', 3, 1,
        'a1000000-0000-4000-8000-00000000001b', '', '#', NULL, NULL, 0, false, true, true
    ),
    (
        'a1000000-0000-4000-8000-00000000021c', '查询', 'system:security-policy:query', 3, 1,
        'a1000000-0000-4000-8000-00000000001c', '', '#', NULL, NULL, 0, false, true, true
    ),
    (
        'a1000000-0000-4000-8000-00000000022c', '修改', 'system:security-policy:update', 3, 2,
        'a1000000-0000-4000-8000-00000000001c', '', '#', NULL, NULL, 0, false, true, true
    );

INSERT INTO sys_role_menus (tenant_id, role_id, menu_id)
SELECT rm.tenant_id, rm.role_id, m.id
FROM sys_role_menus rm
JOIN sys_menus m ON m.id IN (
    'a1000000-0000-4000-8000-000000000003',
    'a1000000-0000-4000-8000-000000000019',
    'a1000000-0000-4000-8000-00000000001a',
    'a1000000-0000-4000-8000-00000000001b',
    'a1000000-0000-4000-8000-00000000001c',
    'a1000000-0000-4000-8000-000000000219',
    'a1000000-0000-4000-8000-00000000021a',
    'a1000000-0000-4000-8000-00000000021b',
    'a1000000-0000-4000-8000-00000000021c',
    'a1000000-0000-4000-8000-00000000022c'
)
WHERE rm.menu_id = 'a1000000-0000-4000-8000-000000000001'
  AND rm.deleted = false
  AND NOT EXISTS (
      SELECT 1 FROM sys_role_menus existing
      WHERE existing.tenant_id = rm.tenant_id
        AND existing.role_id = rm.role_id
        AND existing.menu_id = m.id
        AND existing.deleted = false
  );
