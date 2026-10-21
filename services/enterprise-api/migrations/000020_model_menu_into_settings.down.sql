INSERT INTO sys_menus (
    id, name, permission, type, sort, parent_id, path, icon,
    component, component_name, status, visible, keep_alive, always_show
) VALUES
    (
        'a1000000-0000-4000-8000-00000000001f', '模型配置', 'model:query', 2, 16, NULL, '/models', '#',
        'views/ModelsView.vue', 'ModelsView', 0, true, true, true
    ),
    (
        'a1000000-0000-4000-8000-00000000021f', '查看', 'model:query', 3, 1,
        'a1000000-0000-4000-8000-00000000001f', '', '#', NULL, NULL, 0, false, true, true
    ),
    (
        'a1000000-0000-4000-8000-00000000022f', '管理', 'model:manage', 3, 2,
        'a1000000-0000-4000-8000-00000000001f', '', '#', NULL, NULL, 0, false, true, true
    );

INSERT INTO sys_role_menus (tenant_id, role_id, menu_id)
SELECT existing.tenant_id, existing.role_id, 'a1000000-0000-4000-8000-00000000021f'
FROM sys_role_menus existing
WHERE existing.menu_id = 'a1000000-0000-4000-8000-000000000222'
  AND existing.deleted = false
  AND NOT EXISTS (
      SELECT 1 FROM sys_role_menus already
      WHERE already.tenant_id = existing.tenant_id
        AND already.role_id = existing.role_id
        AND already.menu_id = 'a1000000-0000-4000-8000-00000000021f'
        AND already.deleted = false
  );

INSERT INTO sys_role_menus (tenant_id, role_id, menu_id)
SELECT existing.tenant_id, existing.role_id, 'a1000000-0000-4000-8000-00000000022f'
FROM sys_role_menus existing
WHERE existing.menu_id = 'a1000000-0000-4000-8000-000000000223'
  AND existing.deleted = false
  AND NOT EXISTS (
      SELECT 1 FROM sys_role_menus already
      WHERE already.tenant_id = existing.tenant_id
        AND already.role_id = existing.role_id
        AND already.menu_id = 'a1000000-0000-4000-8000-00000000022f'
        AND already.deleted = false
  );

INSERT INTO sys_role_menus (tenant_id, role_id, menu_id)
SELECT existing.tenant_id, existing.role_id, 'a1000000-0000-4000-8000-00000000001f'
FROM sys_role_menus existing
WHERE existing.menu_id = 'a1000000-0000-4000-8000-000000000222'
  AND existing.deleted = false
  AND NOT EXISTS (
      SELECT 1 FROM sys_role_menus already
      WHERE already.tenant_id = existing.tenant_id
        AND already.role_id = existing.role_id
        AND already.menu_id = 'a1000000-0000-4000-8000-00000000001f'
        AND already.deleted = false
  );

DELETE FROM sys_role_menus
WHERE menu_id IN (
    'a1000000-0000-4000-8000-000000000222',
    'a1000000-0000-4000-8000-000000000223'
);

DELETE FROM sys_menus
WHERE id IN (
    'a1000000-0000-4000-8000-000000000222',
    'a1000000-0000-4000-8000-000000000223'
);
