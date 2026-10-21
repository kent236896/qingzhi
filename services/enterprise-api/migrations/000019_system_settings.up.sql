CREATE TABLE sys_weknora_settings (
    id smallint PRIMARY KEY,
    base_url varchar(256) NOT NULL DEFAULT '',
    api_key text NOT NULL DEFAULT '',
    updated_by varchar(64) NOT NULL DEFAULT '',
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT sys_weknora_settings_singleton CHECK (id = 1)
);

INSERT INTO sys_weknora_settings (id) VALUES (1);

COMMENT ON TABLE sys_weknora_settings IS 'WeKnora 连接配置，只在服务端保存 API Key';
COMMENT ON COLUMN sys_weknora_settings.base_url IS '主机和端口，不带 /api/v1';
COMMENT ON COLUMN sys_weknora_settings.api_key IS '服务端保存的 API Key，接口不返回明文';

INSERT INTO sys_menus (
    id, name, permission, type, sort, parent_id, path, icon,
    component, component_name, status, visible, keep_alive, always_show
) VALUES
    (
        'a1000000-0000-4000-8000-000000000020', '系统配置', 'system:settings:query', 2, 80,
        'a1000000-0000-4000-8000-000000000001', '/settings', '#',
        'views/SettingsView.vue', 'SettingsView', 0, true, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000220', '查看', 'system:settings:query', 3, 1,
        'a1000000-0000-4000-8000-000000000020', '', '#', NULL, NULL, 0, false, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000221', '修改', 'system:settings:update', 3, 2,
        'a1000000-0000-4000-8000-000000000020', '', '#', NULL, NULL, 0, false, true, true
    );

INSERT INTO sys_role_menus (tenant_id, role_id, menu_id)
SELECT r.tenant_id, r.id, m.id
FROM sys_roles r
JOIN sys_menus m ON m.id IN (
    'a1000000-0000-4000-8000-000000000020',
    'a1000000-0000-4000-8000-000000000220',
    'a1000000-0000-4000-8000-000000000221'
)
WHERE r.deleted = false
  AND r.code = 'super_admin'
  AND NOT EXISTS (
      SELECT 1 FROM sys_role_menus existing
      WHERE existing.tenant_id = r.tenant_id
        AND existing.role_id = r.id
        AND existing.menu_id = m.id
        AND existing.deleted = false
  );
