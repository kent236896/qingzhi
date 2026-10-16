CREATE TABLE sys_roles (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES sys_tenants (id),
    name varchar(30) NOT NULL,
    code varchar(100) NOT NULL,
    sort integer NOT NULL DEFAULT 0,
    status smallint NOT NULL DEFAULT 0,
    type smallint NOT NULL DEFAULT 2,
    remark varchar(500),
    creator varchar(64) NOT NULL DEFAULT '',
    create_time timestamptz NOT NULL DEFAULT now(),
    updater varchar(64) NOT NULL DEFAULT '',
    update_time timestamptz NOT NULL DEFAULT now(),
    deleted boolean NOT NULL DEFAULT false,
    CONSTRAINT sys_roles_status_check CHECK (status IN (0, 1)),
    CONSTRAINT sys_roles_type_check CHECK (type IN (1, 2))
);

CREATE UNIQUE INDEX sys_roles_code_unique ON sys_roles (tenant_id, code) WHERE deleted = false;
CREATE INDEX sys_roles_tenant ON sys_roles (tenant_id, sort, name);

COMMENT ON TABLE sys_roles IS '角色';
COMMENT ON COLUMN sys_roles.name IS '角色名称';
COMMENT ON COLUMN sys_roles.code IS '角色权限字符串';
COMMENT ON COLUMN sys_roles.sort IS '显示顺序';
COMMENT ON COLUMN sys_roles.status IS '角色状态（0正常 1停用）';
COMMENT ON COLUMN sys_roles.type IS '角色类型（1系统 2自定义）';
COMMENT ON COLUMN sys_roles.remark IS '备注';
COMMENT ON COLUMN sys_roles.creator IS '创建者';
COMMENT ON COLUMN sys_roles.create_time IS '创建时间';
COMMENT ON COLUMN sys_roles.updater IS '更新者';
COMMENT ON COLUMN sys_roles.update_time IS '更新时间';
COMMENT ON COLUMN sys_roles.deleted IS '是否删除';
COMMENT ON COLUMN sys_roles.tenant_id IS '租户编号';

CREATE TABLE sys_role_menus (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES sys_tenants (id),
    role_id uuid NOT NULL REFERENCES sys_roles (id),
    menu_id uuid NOT NULL REFERENCES sys_menus (id),
    creator varchar(64) NOT NULL DEFAULT '',
    create_time timestamptz NOT NULL DEFAULT now(),
    updater varchar(64) NOT NULL DEFAULT '',
    update_time timestamptz NOT NULL DEFAULT now(),
    deleted boolean NOT NULL DEFAULT false
);

CREATE UNIQUE INDEX sys_role_menus_unique ON sys_role_menus (tenant_id, role_id, menu_id) WHERE deleted = false;
CREATE INDEX sys_role_menus_role ON sys_role_menus (role_id) WHERE deleted = false;
CREATE INDEX sys_role_menus_menu ON sys_role_menus (menu_id) WHERE deleted = false;

COMMENT ON TABLE sys_role_menus IS '角色菜单';
COMMENT ON COLUMN sys_role_menus.role_id IS '角色ID';
COMMENT ON COLUMN sys_role_menus.menu_id IS '菜单ID';
COMMENT ON COLUMN sys_role_menus.creator IS '创建者';
COMMENT ON COLUMN sys_role_menus.create_time IS '创建时间';
COMMENT ON COLUMN sys_role_menus.updater IS '更新者';
COMMENT ON COLUMN sys_role_menus.update_time IS '更新时间';
COMMENT ON COLUMN sys_role_menus.deleted IS '是否删除';
COMMENT ON COLUMN sys_role_menus.tenant_id IS '租户编号';

CREATE TABLE sys_user_roles (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES sys_tenants (id),
    user_id uuid NOT NULL REFERENCES sys_users (id),
    role_id uuid NOT NULL REFERENCES sys_roles (id),
    creator varchar(64) NOT NULL DEFAULT '',
    create_time timestamptz NOT NULL DEFAULT now(),
    updater varchar(64) NOT NULL DEFAULT '',
    update_time timestamptz NOT NULL DEFAULT now(),
    deleted boolean NOT NULL DEFAULT false
);

CREATE UNIQUE INDEX sys_user_roles_unique ON sys_user_roles (tenant_id, user_id, role_id) WHERE deleted = false;
CREATE INDEX sys_user_roles_role ON sys_user_roles (role_id) WHERE deleted = false;
CREATE INDEX sys_user_roles_user ON sys_user_roles (user_id) WHERE deleted = false;

COMMENT ON TABLE sys_user_roles IS '角色用户';
COMMENT ON COLUMN sys_user_roles.user_id IS '用户ID';
COMMENT ON COLUMN sys_user_roles.role_id IS '角色ID';
COMMENT ON COLUMN sys_user_roles.creator IS '创建者';
COMMENT ON COLUMN sys_user_roles.create_time IS '创建时间';
COMMENT ON COLUMN sys_user_roles.updater IS '更新者';
COMMENT ON COLUMN sys_user_roles.update_time IS '更新时间';
COMMENT ON COLUMN sys_user_roles.deleted IS '是否删除';
COMMENT ON COLUMN sys_user_roles.tenant_id IS '租户编号';

INSERT INTO sys_menus (
    id, name, permission, type, sort, parent_id, path, icon,
    component, component_name, status, visible, keep_alive, always_show
) VALUES (
    'a1000000-0000-4000-8000-000000000017', '角色管理', 'system:role:query', 2, 45,
    'a1000000-0000-4000-8000-000000000001', '/roles', '#',
    'views/RolesView.vue', 'RolesView', 0, true, true, true
);
