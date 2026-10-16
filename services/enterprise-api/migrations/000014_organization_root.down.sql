DROP INDEX IF EXISTS sys_organizations_one_root;

COMMENT ON TABLE sys_organizations IS '部门';
COMMENT ON COLUMN sys_organizations.parent_id IS '父部门id，空表示根部门';
