DROP INDEX IF EXISTS sys_organizations_one_tenant_root;
ALTER TABLE sys_organizations DROP COLUMN IF EXISTS is_tenant_root;

COMMENT ON COLUMN sys_organizations.parent_id IS '上级组织。空表示该租户的主体组织';
