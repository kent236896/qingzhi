DELETE FROM sys_role_menus
WHERE role_id = 'b1000000-0000-4000-8000-000000000002';

DELETE FROM sys_user_roles
WHERE role_id = 'b1000000-0000-4000-8000-000000000002'
   OR user_id = 'b1000000-0000-4000-8000-000000000001';

DELETE FROM sys_tenant_memberships
WHERE user_id = 'b1000000-0000-4000-8000-000000000001';

DELETE FROM sys_roles
WHERE id = 'b1000000-0000-4000-8000-000000000002'
   OR (code = 'super_admin' AND type = 1);

DELETE FROM sys_users
WHERE id = 'b1000000-0000-4000-8000-000000000001'
   OR username = 'admin';

DROP TABLE IF EXISTS sys_role_depts;

ALTER TABLE sys_roles
    DROP CONSTRAINT IF EXISTS sys_roles_data_scope_check;

ALTER TABLE sys_roles
    DROP COLUMN IF EXISTS data_scope;
