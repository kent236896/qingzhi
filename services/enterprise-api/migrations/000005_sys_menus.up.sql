CREATE TABLE sys_menus (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name varchar(50) NOT NULL,
    permission varchar(100) NOT NULL DEFAULT '',
    type smallint NOT NULL,
    sort integer NOT NULL DEFAULT 0,
    parent_id uuid REFERENCES sys_menus (id),
    path varchar(200) NOT NULL DEFAULT '',
    icon varchar(100) NOT NULL DEFAULT '#',
    component varchar(255),
    component_name varchar(255),
    status smallint NOT NULL DEFAULT 0,
    visible boolean NOT NULL DEFAULT true,
    keep_alive boolean NOT NULL DEFAULT true,
    always_show boolean NOT NULL DEFAULT true,
    creator varchar(64) NOT NULL DEFAULT '',
    create_time timestamptz NOT NULL DEFAULT now(),
    updater varchar(64) NOT NULL DEFAULT '',
    update_time timestamptz NOT NULL DEFAULT now(),
    deleted boolean NOT NULL DEFAULT false,
    CONSTRAINT sys_menus_type_check CHECK (type IN (1, 2, 3)),
    CONSTRAINT sys_menus_status_check CHECK (status IN (0, 1))
);

CREATE INDEX sys_menus_parent ON sys_menus (parent_id, sort, name) WHERE deleted = false;

COMMENT ON TABLE sys_menus IS '菜单';
COMMENT ON COLUMN sys_menus.name IS '菜单名称';
COMMENT ON COLUMN sys_menus.permission IS '权限标识';
COMMENT ON COLUMN sys_menus.type IS '菜单类型（1目录 2菜单 3按钮）';
COMMENT ON COLUMN sys_menus.sort IS '显示顺序';
COMMENT ON COLUMN sys_menus.parent_id IS '父菜单ID';
COMMENT ON COLUMN sys_menus.path IS '路由地址';
COMMENT ON COLUMN sys_menus.icon IS '菜单图标';
COMMENT ON COLUMN sys_menus.component IS '组件路径';
COMMENT ON COLUMN sys_menus.component_name IS '组件名';
COMMENT ON COLUMN sys_menus.status IS '菜单状态（0正常 1停用）';
COMMENT ON COLUMN sys_menus.visible IS '是否可见';
COMMENT ON COLUMN sys_menus.keep_alive IS '是否缓存';
COMMENT ON COLUMN sys_menus.always_show IS '是否总是显示';
COMMENT ON COLUMN sys_menus.creator IS '创建者';
COMMENT ON COLUMN sys_menus.create_time IS '创建时间';
COMMENT ON COLUMN sys_menus.updater IS '更新者';
COMMENT ON COLUMN sys_menus.update_time IS '更新时间';
COMMENT ON COLUMN sys_menus.deleted IS '是否删除';
