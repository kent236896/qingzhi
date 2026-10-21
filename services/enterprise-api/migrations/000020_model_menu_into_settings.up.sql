INSERT INTO sys_menus (
    id, name, permission, type, sort, parent_id, path, icon,
    component, component_name, status, visible, keep_alive, always_show
) VALUES
    (
        'a1000000-0000-4000-8000-000000000222', '查看模型', 'model:query', 3, 3,
        'a1000000-0000-4000-8000-000000000020', '', '#', NULL, NULL, 0, false, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000223', '管理模型', 'model:manage', 3, 4,
        'a1000000-0000-4000-8000-000000000020', '', '#', NULL, NULL, 0, false, true, true
    );

INSERT INTO sys_role_menus (tenant_id, role_id, menu_id)
SELECT existing.tenant_id, existing.role_id, 'a1000000-0000-4000-8000-000000000222'
FROM sys_role_menus existing
WHERE existing.menu_id = 'a1000000-0000-4000-8000-00000000021f'
  AND existing.deleted = false
  AND NOT EXISTS (
      SELECT 1 FROM sys_role_menus already
      WHERE already.tenant_id = existing.tenant_id
        AND already.role_id = existing.role_id
        AND already.menu_id = 'a1000000-0000-4000-8000-000000000222'
        AND already.deleted = false
  );

INSERT INTO sys_role_menus (tenant_id, role_id, menu_id)
SELECT existing.tenant_id, existing.role_id, 'a1000000-0000-4000-8000-000000000223'
FROM sys_role_menus existing
WHERE existing.menu_id = 'a1000000-0000-4000-8000-00000000022f'
  AND existing.deleted = false
  AND NOT EXISTS (
      SELECT 1 FROM sys_role_menus already
      WHERE already.tenant_id = existing.tenant_id
        AND already.role_id = existing.role_id
        AND already.menu_id = 'a1000000-0000-4000-8000-000000000223'
        AND already.deleted = false
  );

DELETE FROM sys_role_menus
WHERE menu_id IN (
    'a1000000-0000-4000-8000-00000000001f',
    'a1000000-0000-4000-8000-00000000021f',
    'a1000000-0000-4000-8000-00000000022f'
);

DELETE FROM sys_menus
WHERE id IN (
    'a1000000-0000-4000-8000-00000000001f',
    'a1000000-0000-4000-8000-00000000021f',
    'a1000000-0000-4000-8000-00000000022f'
);
