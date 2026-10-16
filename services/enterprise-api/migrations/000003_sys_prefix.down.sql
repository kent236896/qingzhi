ALTER TABLE sys_organization_memberships RENAME COLUMN create_time TO created_at;
ALTER TABLE sys_organization_memberships RENAME COLUMN update_time TO updated_at;

ALTER TABLE sys_tenant_memberships DROP CONSTRAINT sys_tenant_memberships_status_check;
ALTER TABLE sys_tenant_memberships
    ALTER COLUMN status DROP DEFAULT,
    ALTER COLUMN status TYPE text USING (
        CASE status WHEN 0 THEN 'active' WHEN 1 THEN 'disabled' ELSE 'active' END
    ),
    ALTER COLUMN status SET NOT NULL;
ALTER TABLE sys_tenant_memberships ADD CONSTRAINT tenant_memberships_status_check CHECK (status IN ('active', 'disabled'));
ALTER TABLE sys_tenant_memberships RENAME COLUMN create_time TO created_at;
ALTER TABLE sys_tenant_memberships RENAME COLUMN update_time TO updated_at;

DROP INDEX IF EXISTS sys_organizations_code_unique;
DROP INDEX IF EXISTS sys_organizations_root_name;
DROP INDEX IF EXISTS sys_organizations_sibling_name;
ALTER TABLE sys_organizations
    DROP COLUMN leader_user_id,
    DROP COLUMN phone,
    DROP COLUMN email,
    DROP COLUMN creator,
    DROP COLUMN updater,
    DROP COLUMN deleted;
ALTER TABLE sys_organizations DROP CONSTRAINT sys_organizations_status_check;
ALTER TABLE sys_organizations
    ALTER COLUMN status DROP DEFAULT,
    ALTER COLUMN status TYPE text USING (
        CASE status WHEN 0 THEN 'active' WHEN 1 THEN 'disabled' ELSE 'active' END
    ),
    ALTER COLUMN status SET NOT NULL,
    ALTER COLUMN name TYPE text,
    ALTER COLUMN name DROP DEFAULT;
ALTER TABLE sys_organizations ADD CONSTRAINT organizations_status_check CHECK (status IN ('active', 'disabled'));
ALTER TABLE sys_organizations ADD CONSTRAINT organizations_code_unique UNIQUE (tenant_id, code);
CREATE UNIQUE INDEX organizations_root_name ON sys_organizations (tenant_id, name) WHERE parent_id IS NULL;
CREATE UNIQUE INDEX organizations_sibling_name ON sys_organizations (tenant_id, parent_id, name) WHERE parent_id IS NOT NULL;
ALTER TABLE sys_organizations RENAME COLUMN sort TO sort_order;
ALTER TABLE sys_organizations RENAME COLUMN create_time TO created_at;
ALTER TABLE sys_organizations RENAME COLUMN update_time TO updated_at;

DROP INDEX IF EXISTS sys_tenants_code_unique;
ALTER TABLE sys_tenants
    DROP COLUMN contact_user_id,
    DROP COLUMN contact_name,
    DROP COLUMN contact_mobile,
    DROP COLUMN account_count,
    DROP COLUMN creator,
    DROP COLUMN updater,
    DROP COLUMN deleted;
ALTER TABLE sys_tenants DROP CONSTRAINT sys_tenants_status_check;
ALTER TABLE sys_tenants
    ALTER COLUMN status DROP DEFAULT,
    ALTER COLUMN status TYPE text USING (
        CASE status WHEN 0 THEN 'active' WHEN 1 THEN 'disabled' ELSE 'active' END
    ),
    ALTER COLUMN status SET NOT NULL,
    ALTER COLUMN name TYPE text;
ALTER TABLE sys_tenants ADD CONSTRAINT tenants_status_check CHECK (status IN ('active', 'disabled'));
ALTER TABLE sys_tenants ADD CONSTRAINT tenants_code_unique UNIQUE (code);
ALTER TABLE sys_tenants RENAME COLUMN create_time TO created_at;
ALTER TABLE sys_tenants RENAME COLUMN update_time TO updated_at;

DROP INDEX IF EXISTS sys_users_username_unique;
DROP INDEX IF EXISTS sys_users_email_unique;
DROP INDEX IF EXISTS sys_users_mobile_unique;
ALTER TABLE sys_users DROP CONSTRAINT sys_users_sex_check;
ALTER TABLE sys_users
    DROP COLUMN password,
    DROP COLUMN remark,
    DROP COLUMN sex,
    DROP COLUMN avatar,
    DROP COLUMN login_ip,
    DROP COLUMN login_date,
    DROP COLUMN creator,
    DROP COLUMN updater,
    DROP COLUMN deleted;
ALTER TABLE sys_users DROP CONSTRAINT sys_users_status_check;
ALTER TABLE sys_users
    ALTER COLUMN status DROP DEFAULT,
    ALTER COLUMN status TYPE text USING (
        CASE status WHEN 0 THEN 'active' WHEN 1 THEN 'disabled' ELSE 'active' END
    ),
    ALTER COLUMN status SET NOT NULL,
    ALTER COLUMN username TYPE text,
    ALTER COLUMN nickname TYPE text,
    ALTER COLUMN email DROP NOT NULL,
    ALTER COLUMN email DROP DEFAULT,
    ALTER COLUMN email TYPE text,
    ALTER COLUMN mobile DROP NOT NULL,
    ALTER COLUMN mobile DROP DEFAULT,
    ALTER COLUMN mobile TYPE text;
UPDATE sys_users SET email = NULL WHERE email = '';
UPDATE sys_users SET mobile = NULL WHERE mobile = '';
ALTER TABLE sys_users ADD CONSTRAINT users_status_check CHECK (status IN ('active', 'disabled'));
ALTER TABLE sys_users ADD CONSTRAINT users_username_unique UNIQUE (username);
CREATE UNIQUE INDEX users_email_unique ON sys_users (email) WHERE email IS NOT NULL;
CREATE UNIQUE INDEX users_mobile_unique ON sys_users (mobile) WHERE mobile IS NOT NULL;
ALTER TABLE sys_users RENAME COLUMN nickname TO display_name;
ALTER TABLE sys_users RENAME COLUMN create_time TO created_at;
ALTER TABLE sys_users RENAME COLUMN update_time TO updated_at;

ALTER TABLE sys_organization_memberships RENAME TO organization_memberships;
ALTER TABLE sys_tenant_memberships RENAME TO tenant_memberships;
ALTER TABLE sys_organizations RENAME TO organizations;
ALTER TABLE sys_tenants RENAME TO tenants;
ALTER TABLE sys_users RENAME TO users;
