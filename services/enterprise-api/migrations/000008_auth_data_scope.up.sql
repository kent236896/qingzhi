ALTER TABLE sys_roles
    ADD COLUMN data_scope smallint NOT NULL DEFAULT 1;

ALTER TABLE sys_roles
    ADD CONSTRAINT sys_roles_data_scope_check CHECK (data_scope IN (1, 2, 3, 4, 5));

COMMENT ON COLUMN sys_roles.data_scope IS '数据范围（1全部 2自定义 3本部门 4本部门及以下 5仅本人）';

CREATE TABLE sys_role_depts (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES sys_tenants (id),
    role_id uuid NOT NULL REFERENCES sys_roles (id),
    dept_id uuid NOT NULL REFERENCES sys_organizations (id),
    creator varchar(64) NOT NULL DEFAULT '',
    create_time timestamptz NOT NULL DEFAULT now(),
    updater varchar(64) NOT NULL DEFAULT '',
    update_time timestamptz NOT NULL DEFAULT now(),
    deleted boolean NOT NULL DEFAULT false
);

CREATE UNIQUE INDEX sys_role_depts_unique ON sys_role_depts (tenant_id, role_id, dept_id) WHERE deleted = false;
CREATE INDEX sys_role_depts_role ON sys_role_depts (role_id) WHERE deleted = false;
CREATE INDEX sys_role_depts_dept ON sys_role_depts (dept_id) WHERE deleted = false;

COMMENT ON TABLE sys_role_depts IS '角色部门（自定义数据权限）';
COMMENT ON COLUMN sys_role_depts.role_id IS '角色ID';
COMMENT ON COLUMN sys_role_depts.dept_id IS '部门ID';
COMMENT ON COLUMN sys_role_depts.creator IS '创建者';
COMMENT ON COLUMN sys_role_depts.create_time IS '创建时间';
COMMENT ON COLUMN sys_role_depts.updater IS '更新者';
COMMENT ON COLUMN sys_role_depts.update_time IS '更新时间';
COMMENT ON COLUMN sys_role_depts.deleted IS '是否删除';
COMMENT ON COLUMN sys_role_depts.tenant_id IS '租户编号';

-- 默认管理员：Admin@123456（bcrypt）。挂到已有首个租户并授予全部菜单。
DO $$
DECLARE
    tid uuid;
    uid uuid := 'b1000000-0000-4000-8000-000000000001';
    rid uuid := 'b1000000-0000-4000-8000-000000000002';
BEGIN
    SELECT id INTO tid FROM sys_tenants WHERE deleted = false ORDER BY create_time ASC LIMIT 1;
    IF tid IS NULL THEN
        RETURN;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM sys_users WHERE id = uid OR (username = 'admin' AND deleted = false)) THEN
        INSERT INTO sys_users (
            id, username, password, nickname, remark, email, mobile, sex, avatar, status
        ) VALUES (
            uid, 'admin',
            '$2a$10$a8mCcVpyRa0L1IYKrkX9NelFKhgCGCExJBIR4K2vzB.q1e46lbePa',
            '管理员', '系统初始化管理员', '', '', 0, '', 0
        );
    ELSE
        SELECT id INTO uid FROM sys_users WHERE username = 'admin' AND deleted = false LIMIT 1;
        UPDATE sys_users
        SET password = '$2a$10$a8mCcVpyRa0L1IYKrkX9NelFKhgCGCExJBIR4K2vzB.q1e46lbePa',
            nickname = COALESCE(NULLIF(nickname, ''), '管理员'),
            update_time = now()
        WHERE id = uid;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM sys_roles WHERE id = rid OR (tenant_id = tid AND code = 'super_admin' AND deleted = false)) THEN
        INSERT INTO sys_roles (
            id, tenant_id, name, code, sort, status, type, data_scope, remark
        ) VALUES (
            rid, tid, '超级管理员', 'super_admin', 0, 0, 1, 1, '系统初始化角色'
        );
    ELSE
        SELECT id INTO rid FROM sys_roles WHERE tenant_id = tid AND code = 'super_admin' AND deleted = false LIMIT 1;
        UPDATE sys_roles
        SET data_scope = 1, type = 1, status = 0, update_time = now()
        WHERE id = rid;
    END IF;

    INSERT INTO sys_tenant_memberships (tenant_id, user_id, status)
    SELECT tid, uid, 0
    WHERE NOT EXISTS (
        SELECT 1 FROM sys_tenant_memberships
        WHERE tenant_id = tid AND user_id = uid
    );

    INSERT INTO sys_user_roles (tenant_id, user_id, role_id)
    SELECT tid, uid, rid
    WHERE NOT EXISTS (
        SELECT 1 FROM sys_user_roles
        WHERE tenant_id = tid AND user_id = uid AND role_id = rid AND deleted = false
    );

    INSERT INTO sys_role_menus (tenant_id, role_id, menu_id)
    SELECT tid, rid, m.id
    FROM sys_menus m
    WHERE m.deleted = false
      AND NOT EXISTS (
          SELECT 1 FROM sys_role_menus rm
          WHERE rm.tenant_id = tid AND rm.role_id = rid AND rm.menu_id = m.id AND rm.deleted = false
      );
END $$;
