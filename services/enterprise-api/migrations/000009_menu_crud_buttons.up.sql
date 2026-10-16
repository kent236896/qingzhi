-- 业务菜单下增加增删改查按钮权限（三级：按钮）。查询按钮与菜单自身的 query 标识一致。
INSERT INTO sys_menus (
    id, name, permission, type, sort, parent_id, path, icon,
    component, component_name, status, visible, keep_alive, always_show
) VALUES
    -- 租户
    ('a1000000-0000-4000-8000-000000000112', '查询', 'system:tenant:query', 3, 1,
     'a1000000-0000-4000-8000-000000000012', '', '#', NULL, NULL, 0, false, true, true),
    ('a1000000-0000-4000-8000-000000000122', '新增', 'system:tenant:create', 3, 2,
     'a1000000-0000-4000-8000-000000000012', '', '#', NULL, NULL, 0, false, true, true),
    ('a1000000-0000-4000-8000-000000000132', '修改', 'system:tenant:update', 3, 3,
     'a1000000-0000-4000-8000-000000000012', '', '#', NULL, NULL, 0, false, true, true),
    ('a1000000-0000-4000-8000-000000000142', '删除', 'system:tenant:delete', 3, 4,
     'a1000000-0000-4000-8000-000000000012', '', '#', NULL, NULL, 0, false, true, true),
    -- 组织
    ('a1000000-0000-4000-8000-000000000113', '查询', 'system:organization:query', 3, 1,
     'a1000000-0000-4000-8000-000000000013', '', '#', NULL, NULL, 0, false, true, true),
    ('a1000000-0000-4000-8000-000000000123', '新增', 'system:organization:create', 3, 2,
     'a1000000-0000-4000-8000-000000000013', '', '#', NULL, NULL, 0, false, true, true),
    ('a1000000-0000-4000-8000-000000000133', '修改', 'system:organization:update', 3, 3,
     'a1000000-0000-4000-8000-000000000013', '', '#', NULL, NULL, 0, false, true, true),
    ('a1000000-0000-4000-8000-000000000143', '删除', 'system:organization:delete', 3, 4,
     'a1000000-0000-4000-8000-000000000013', '', '#', NULL, NULL, 0, false, true, true),
    -- 岗位
    ('a1000000-0000-4000-8000-000000000114', '查询', 'system:post:query', 3, 1,
     'a1000000-0000-4000-8000-000000000014', '', '#', NULL, NULL, 0, false, true, true),
    ('a1000000-0000-4000-8000-000000000124', '新增', 'system:post:create', 3, 2,
     'a1000000-0000-4000-8000-000000000014', '', '#', NULL, NULL, 0, false, true, true),
    ('a1000000-0000-4000-8000-000000000134', '修改', 'system:post:update', 3, 3,
     'a1000000-0000-4000-8000-000000000014', '', '#', NULL, NULL, 0, false, true, true),
    ('a1000000-0000-4000-8000-000000000144', '删除', 'system:post:delete', 3, 4,
     'a1000000-0000-4000-8000-000000000014', '', '#', NULL, NULL, 0, false, true, true),
    -- 角色
    ('a1000000-0000-4000-8000-000000000117', '查询', 'system:role:query', 3, 1,
     'a1000000-0000-4000-8000-000000000017', '', '#', NULL, NULL, 0, false, true, true),
    ('a1000000-0000-4000-8000-000000000127', '新增', 'system:role:create', 3, 2,
     'a1000000-0000-4000-8000-000000000017', '', '#', NULL, NULL, 0, false, true, true),
    ('a1000000-0000-4000-8000-000000000137', '修改', 'system:role:update', 3, 3,
     'a1000000-0000-4000-8000-000000000017', '', '#', NULL, NULL, 0, false, true, true),
    ('a1000000-0000-4000-8000-000000000147', '删除', 'system:role:delete', 3, 4,
     'a1000000-0000-4000-8000-000000000017', '', '#', NULL, NULL, 0, false, true, true),
    -- 菜单
    ('a1000000-0000-4000-8000-000000000115', '查询', 'system:menu:query', 3, 1,
     'a1000000-0000-4000-8000-000000000015', '', '#', NULL, NULL, 0, false, true, true),
    ('a1000000-0000-4000-8000-000000000125', '新增', 'system:menu:create', 3, 2,
     'a1000000-0000-4000-8000-000000000015', '', '#', NULL, NULL, 0, false, true, true),
    ('a1000000-0000-4000-8000-000000000135', '修改', 'system:menu:update', 3, 3,
     'a1000000-0000-4000-8000-000000000015', '', '#', NULL, NULL, 0, false, true, true),
    ('a1000000-0000-4000-8000-000000000145', '删除', 'system:menu:delete', 3, 4,
     'a1000000-0000-4000-8000-000000000015', '', '#', NULL, NULL, 0, false, true, true),
    -- 用户
    ('a1000000-0000-4000-8000-000000000116', '查询', 'system:user:query', 3, 1,
     'a1000000-0000-4000-8000-000000000016', '', '#', NULL, NULL, 0, false, true, true),
    ('a1000000-0000-4000-8000-000000000126', '新增', 'system:user:create', 3, 2,
     'a1000000-0000-4000-8000-000000000016', '', '#', NULL, NULL, 0, false, true, true),
    ('a1000000-0000-4000-8000-000000000136', '修改', 'system:user:update', 3, 3,
     'a1000000-0000-4000-8000-000000000016', '', '#', NULL, NULL, 0, false, true, true),
    ('a1000000-0000-4000-8000-000000000146', '删除', 'system:user:delete', 3, 4,
     'a1000000-0000-4000-8000-000000000016', '', '#', NULL, NULL, 0, false, true, true);

-- 已拥有父菜单的角色自动获得其下按钮权限（如超级管理员）。
INSERT INTO sys_role_menus (tenant_id, role_id, menu_id)
SELECT rm.tenant_id, rm.role_id, btn.id
FROM sys_role_menus rm
JOIN sys_menus btn ON btn.parent_id = rm.menu_id AND btn.type = 3 AND btn.deleted = false
WHERE rm.deleted = false
  AND rm.menu_id IN (
      'a1000000-0000-4000-8000-000000000012',
      'a1000000-0000-4000-8000-000000000013',
      'a1000000-0000-4000-8000-000000000014',
      'a1000000-0000-4000-8000-000000000015',
      'a1000000-0000-4000-8000-000000000016',
      'a1000000-0000-4000-8000-000000000017'
  )
  AND NOT EXISTS (
      SELECT 1 FROM sys_role_menus existing
      WHERE existing.tenant_id = rm.tenant_id
        AND existing.role_id = rm.role_id
        AND existing.menu_id = btn.id
        AND existing.deleted = false
  );
