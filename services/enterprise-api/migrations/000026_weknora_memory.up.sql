-- WeKnora 空间 ID（与 API Key 绑定的租户/空间），供终端用户 signed_token 的 tenant_id claim 使用。
ALTER TABLE sys_weknora_settings
    ADD COLUMN IF NOT EXISTS weknora_tenant_id varchar(64) NOT NULL DEFAULT '';

COMMENT ON COLUMN sys_weknora_settings.weknora_tenant_id IS 'WeKnora 空间 ID；须与 API Key 所属空间一致，供 X-External-User-Token.tenant_id';

-- 长期记忆配置页：挂在系统配置同级，权限复用 model:manage（超级管理员）。
INSERT INTO sys_menus (
    id, name, permission, type, sort, parent_id, path, icon,
    component, component_name, status, visible, keep_alive, always_show
) VALUES
    (
        'a1000000-0000-4000-8000-000000000028', '长期记忆', 'model:manage', 2, 81,
        'a1000000-0000-4000-8000-000000000001', '/settings/memory', '#',
        'views/MemorySettingsView.vue', 'MemorySettingsView', 0, true, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000280', '查看', 'model:query', 3, 1,
        'a1000000-0000-4000-8000-000000000028', '', '#', NULL, NULL, 0, false, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000281', '管理', 'model:manage', 3, 2,
        'a1000000-0000-4000-8000-000000000028', '', '#', NULL, NULL, 0, false, true, true
    )
ON CONFLICT (id) DO NOTHING;

INSERT INTO sys_role_menus (tenant_id, role_id, menu_id)
SELECT r.tenant_id, r.id, m.id
FROM sys_roles r
JOIN sys_menus m ON m.id IN (
    'a1000000-0000-4000-8000-000000000028',
    'a1000000-0000-4000-8000-000000000280',
    'a1000000-0000-4000-8000-000000000281'
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
