CREATE TABLE sys_dict_types (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name varchar(100) NOT NULL DEFAULT '',
    type varchar(100) NOT NULL DEFAULT '',
    status smallint NOT NULL DEFAULT 0,
    remark varchar(500),
    creator varchar(64) NOT NULL DEFAULT '',
    create_time timestamptz NOT NULL DEFAULT now(),
    updater varchar(64) NOT NULL DEFAULT '',
    update_time timestamptz NOT NULL DEFAULT now(),
    deleted boolean NOT NULL DEFAULT false,
    deleted_time timestamptz,
    CONSTRAINT sys_dict_types_status_check CHECK (status IN (0, 1))
);

CREATE UNIQUE INDEX sys_dict_types_type_unique ON sys_dict_types (type) WHERE deleted = false;
CREATE INDEX sys_dict_types_name ON sys_dict_types (name) WHERE deleted = false;

COMMENT ON TABLE sys_dict_types IS '字典类型';
COMMENT ON COLUMN sys_dict_types.name IS '字典名称';
COMMENT ON COLUMN sys_dict_types.type IS '字典类型';
COMMENT ON COLUMN sys_dict_types.status IS '状态（0正常 1停用）';
COMMENT ON COLUMN sys_dict_types.remark IS '备注';
COMMENT ON COLUMN sys_dict_types.creator IS '创建者';
COMMENT ON COLUMN sys_dict_types.create_time IS '创建时间';
COMMENT ON COLUMN sys_dict_types.updater IS '更新者';
COMMENT ON COLUMN sys_dict_types.update_time IS '更新时间';
COMMENT ON COLUMN sys_dict_types.deleted IS '是否删除';
COMMENT ON COLUMN sys_dict_types.deleted_time IS '删除时间';

CREATE TABLE sys_dict_data (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    sort integer NOT NULL DEFAULT 0,
    label varchar(100) NOT NULL DEFAULT '',
    value varchar(100) NOT NULL DEFAULT '',
    dict_type varchar(100) NOT NULL DEFAULT '',
    status smallint NOT NULL DEFAULT 0,
    color_type varchar(100) NOT NULL DEFAULT '',
    css_class varchar(100) NOT NULL DEFAULT '',
    remark varchar(500),
    creator varchar(64) NOT NULL DEFAULT '',
    create_time timestamptz NOT NULL DEFAULT now(),
    updater varchar(64) NOT NULL DEFAULT '',
    update_time timestamptz NOT NULL DEFAULT now(),
    deleted boolean NOT NULL DEFAULT false,
    CONSTRAINT sys_dict_data_status_check CHECK (status IN (0, 1))
);

CREATE UNIQUE INDEX sys_dict_data_value_unique ON sys_dict_data (dict_type, value) WHERE deleted = false;
CREATE INDEX sys_dict_data_type ON sys_dict_data (dict_type, sort, label) WHERE deleted = false;

COMMENT ON TABLE sys_dict_data IS '字典数据';
COMMENT ON COLUMN sys_dict_data.sort IS '字典排序';
COMMENT ON COLUMN sys_dict_data.label IS '字典标签';
COMMENT ON COLUMN sys_dict_data.value IS '字典键值';
COMMENT ON COLUMN sys_dict_data.dict_type IS '字典类型';
COMMENT ON COLUMN sys_dict_data.status IS '状态（0正常 1停用）';
COMMENT ON COLUMN sys_dict_data.color_type IS '颜色类型';
COMMENT ON COLUMN sys_dict_data.css_class IS 'css 样式';
COMMENT ON COLUMN sys_dict_data.remark IS '备注';
COMMENT ON COLUMN sys_dict_data.creator IS '创建者';
COMMENT ON COLUMN sys_dict_data.create_time IS '创建时间';
COMMENT ON COLUMN sys_dict_data.updater IS '更新者';
COMMENT ON COLUMN sys_dict_data.update_time IS '更新时间';
COMMENT ON COLUMN sys_dict_data.deleted IS '是否删除';

INSERT INTO sys_menus (
    id, name, permission, type, sort, parent_id, path, icon,
    component, component_name, status, visible, keep_alive, always_show
) VALUES
    (
        'a1000000-0000-4000-8000-000000000018', '字典管理', 'system:dict:query', 2, 70,
        'a1000000-0000-4000-8000-000000000001', '/dicts', '#',
        'views/DictsView.vue', 'DictsView', 0, true, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000118', '查询', 'system:dict:query', 3, 1,
        'a1000000-0000-4000-8000-000000000018', '', '#', NULL, NULL, 0, false, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000128', '新增', 'system:dict:create', 3, 2,
        'a1000000-0000-4000-8000-000000000018', '', '#', NULL, NULL, 0, false, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000138', '修改', 'system:dict:update', 3, 3,
        'a1000000-0000-4000-8000-000000000018', '', '#', NULL, NULL, 0, false, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000148', '删除', 'system:dict:delete', 3, 4,
        'a1000000-0000-4000-8000-000000000018', '', '#', NULL, NULL, 0, false, true, true
    );

-- 已拥有「系统管理」目录的角色同步获得字典菜单及按钮。
INSERT INTO sys_role_menus (tenant_id, role_id, menu_id)
SELECT rm.tenant_id, rm.role_id, m.id
FROM sys_role_menus rm
JOIN sys_menus m ON m.id IN (
    'a1000000-0000-4000-8000-000000000018',
    'a1000000-0000-4000-8000-000000000118',
    'a1000000-0000-4000-8000-000000000128',
    'a1000000-0000-4000-8000-000000000138',
    'a1000000-0000-4000-8000-000000000148'
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
