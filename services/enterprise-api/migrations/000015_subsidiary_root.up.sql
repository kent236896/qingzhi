-- 主体组织用标记区分，子公司主体可以挂在集团主体下面，不再依赖 parent_id 为空。
ALTER TABLE sys_organizations
    ADD COLUMN is_tenant_root boolean NOT NULL DEFAULT false;

UPDATE sys_organizations
SET is_tenant_root = true
WHERE parent_id IS NULL AND deleted = false;

CREATE UNIQUE INDEX sys_organizations_one_tenant_root
    ON sys_organizations (tenant_id)
    WHERE is_tenant_root = true AND deleted = false;

COMMENT ON COLUMN sys_organizations.is_tenant_root IS '是否为该租户的唯一主体组织。子公司主体可以挂在其他租户的主体组织下面';
COMMENT ON COLUMN sys_organizations.parent_id IS '上级组织。主体组织可为空或指向其他租户的主体组织；部门必须属于本租户';
