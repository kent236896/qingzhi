-- 000030 误用了 SSO 已占用的菜单 ID（000030–000035），智能体/工具箱菜单未插入，
-- 且把 SSO 菜单误授给了全部角色。本迁移用空闲 ID 补建并收回误授。

-- 收回：非超级管理员因 000030 误拿到的 SSO 菜单（原仅 super_admin 应有）
UPDATE sys_role_menus rm
SET deleted = true, update_time = now()
FROM sys_roles r
WHERE rm.role_id = r.id
  AND rm.tenant_id = r.tenant_id
  AND rm.deleted = false
  AND r.deleted = false
  AND r.code <> 'super_admin'
  AND rm.menu_id IN (
      'a1000000-0000-4000-8000-000000000030',
      'a1000000-0000-4000-8000-000000000031',
      'a1000000-0000-4000-8000-000000000033',
      'a1000000-0000-4000-8000-000000000034'
  );

INSERT INTO sys_menus (
    id, name, permission, type, sort, parent_id, path, icon,
    component, component_name, status, visible, keep_alive, always_show
) VALUES
    (
        'a1000000-0000-4000-8000-000000000040', '智能体', 'agent:query', 2, 16, NULL, '/agents', '#',
        'views/AgentsView.vue', 'AgentsView', 0, true, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000041', '查询', 'agent:query', 3, 1,
        'a1000000-0000-4000-8000-000000000040', '', '#', NULL, NULL, 0, false, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000042', '管理', 'agent:manage', 3, 2,
        'a1000000-0000-4000-8000-000000000040', '', '#', NULL, NULL, 0, false, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000043', '工具箱', 'toolbox:query', 2, 17, NULL, '/toolbox', '#',
        'views/ToolboxView.vue', 'ToolboxView', 0, true, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000044', '查询', 'toolbox:query', 3, 1,
        'a1000000-0000-4000-8000-000000000043', '', '#', NULL, NULL, 0, false, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000045', '管理', 'toolbox:manage', 3, 2,
        'a1000000-0000-4000-8000-000000000043', '', '#', NULL, NULL, 0, false, true, true
    )
ON CONFLICT (id) DO NOTHING;

-- 产物菜单若已由 000030 写入则保留；若缺则补齐
INSERT INTO sys_menus (
    id, name, permission, type, sort, parent_id, path, icon,
    component, component_name, status, visible, keep_alive, always_show
) VALUES
    (
        'a1000000-0000-4000-8000-000000000036', '产物', 'artifact:query', 2, 18, NULL, '/artifacts', '#',
        'views/ArtifactsView.vue', 'ArtifactsView', 0, true, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000037', '查询', 'artifact:query', 3, 1,
        'a1000000-0000-4000-8000-000000000036', '', '#', NULL, NULL, 0, false, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000038', '管理', 'artifact:manage', 3, 2,
        'a1000000-0000-4000-8000-000000000036', '', '#', NULL, NULL, 0, false, true, true
    )
ON CONFLICT (id) DO NOTHING;

INSERT INTO sys_role_menus (tenant_id, role_id, menu_id)
SELECT r.tenant_id, r.id, m.id
FROM sys_roles r
JOIN sys_menus m ON m.id IN (
    'a1000000-0000-4000-8000-000000000040',
    'a1000000-0000-4000-8000-000000000041',
    'a1000000-0000-4000-8000-000000000043',
    'a1000000-0000-4000-8000-000000000044',
    'a1000000-0000-4000-8000-000000000036',
    'a1000000-0000-4000-8000-000000000037'
)
WHERE r.deleted = false
  AND NOT EXISTS (
      SELECT 1 FROM sys_role_menus existing
      WHERE existing.tenant_id = r.tenant_id
        AND existing.role_id = r.id
        AND existing.menu_id = m.id
        AND existing.deleted = false
  );

INSERT INTO sys_role_menus (tenant_id, role_id, menu_id)
SELECT r.tenant_id, r.id, m.id
FROM sys_roles r
JOIN sys_menus m ON m.id IN (
    'a1000000-0000-4000-8000-000000000042',
    'a1000000-0000-4000-8000-000000000045',
    'a1000000-0000-4000-8000-000000000038'
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
