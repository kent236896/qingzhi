-- 将知识管理 / 智能体 / 工具箱 / 产物收拢到同一目录「知识智能」下。
INSERT INTO sys_menus (
    id, name, permission, type, sort, parent_id, path, icon,
    component, component_name, status, visible, keep_alive, always_show
) VALUES (
    'a1000000-0000-4000-8000-000000000046', '知识智能', '', 1, 15, NULL, '', '#',
    NULL, NULL, 0, true, true, true
)
ON CONFLICT (id) DO NOTHING;

UPDATE sys_menus
SET parent_id = 'a1000000-0000-4000-8000-000000000046',
    sort = 10,
    update_time = now()
WHERE id = 'a1000000-0000-4000-8000-00000000001e'
  AND deleted = false;

UPDATE sys_menus
SET parent_id = 'a1000000-0000-4000-8000-000000000046',
    sort = 20,
    update_time = now()
WHERE id = 'a1000000-0000-4000-8000-000000000040'
  AND deleted = false;

UPDATE sys_menus
SET parent_id = 'a1000000-0000-4000-8000-000000000046',
    sort = 30,
    update_time = now()
WHERE id = 'a1000000-0000-4000-8000-000000000043'
  AND deleted = false;

UPDATE sys_menus
SET parent_id = 'a1000000-0000-4000-8000-000000000046',
    sort = 40,
    update_time = now()
WHERE id = 'a1000000-0000-4000-8000-000000000036'
  AND deleted = false;

INSERT INTO sys_role_menus (tenant_id, role_id, menu_id)
SELECT r.tenant_id, r.id, 'a1000000-0000-4000-8000-000000000046'
FROM sys_roles r
WHERE r.deleted = false
  AND NOT EXISTS (
      SELECT 1 FROM sys_role_menus existing
      WHERE existing.tenant_id = r.tenant_id
        AND existing.role_id = r.id
        AND existing.menu_id = 'a1000000-0000-4000-8000-000000000046'
        AND existing.deleted = false
  );
