-- 每个未删除租户只保留一个主体组织（parent_id 为空）。
-- 已有多个主体组织时中止，避免自动改挂上下级。
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM sys_organizations
        WHERE parent_id IS NULL AND deleted = false
        GROUP BY tenant_id
        HAVING count(*) > 1
    ) THEN
        RAISE EXCEPTION 'a tenant has more than one root organization';
    END IF;
END $$;

INSERT INTO sys_organizations (
    tenant_id, name, code, sort, status, creator, updater, create_time, update_time, deleted
)
SELECT
    t.id,
    t.name,
    CASE
        WHEN EXISTS (
            SELECT 1 FROM sys_organizations existing
            WHERE existing.tenant_id = t.id AND existing.deleted = false AND existing.code = t.code
        ) THEN left(t.code, 59) || '-root'
        ELSE t.code
    END,
    0,
    0,
    'system',
    'system',
    now(),
    now(),
    false
FROM sys_tenants t
WHERE t.deleted = false
  AND NOT EXISTS (
      SELECT 1 FROM sys_organizations o
      WHERE o.tenant_id = t.id AND o.parent_id IS NULL AND o.deleted = false
  );

CREATE UNIQUE INDEX sys_organizations_one_root
    ON sys_organizations (tenant_id)
    WHERE parent_id IS NULL AND deleted = false;

-- 已有租户成员但没有组织任职时，补挂到该租户唯一的主体组织。不删除账号或成员。
INSERT INTO sys_organization_memberships (
    tenant_id, organization_id, user_id, is_primary, create_time, update_time
)
SELECT m.tenant_id, o.id, m.user_id, true, now(), now()
FROM sys_tenant_memberships m
JOIN sys_organizations o
    ON o.tenant_id = m.tenant_id AND o.parent_id IS NULL AND o.deleted = false
WHERE NOT EXISTS (
    SELECT 1 FROM sys_organization_memberships a
    WHERE a.tenant_id = m.tenant_id AND a.user_id = m.user_id
);

COMMENT ON TABLE sys_organizations IS '组织节点。每个租户恰好一个主体组织，其余节点是该主体下的部门';
COMMENT ON COLUMN sys_organizations.parent_id IS '上级组织。空表示该租户的主体组织';
