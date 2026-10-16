ALTER TABLE users RENAME TO sys_users;
ALTER TABLE tenants RENAME TO sys_tenants;
ALTER TABLE organizations RENAME TO sys_organizations;
ALTER TABLE tenant_memberships RENAME TO sys_tenant_memberships;
ALTER TABLE organization_memberships RENAME TO sys_organization_memberships;

ALTER TABLE sys_users RENAME COLUMN display_name TO nickname;
ALTER TABLE sys_users RENAME COLUMN created_at TO create_time;
ALTER TABLE sys_users RENAME COLUMN updated_at TO update_time;
ALTER TABLE sys_users DROP CONSTRAINT users_status_check;
ALTER TABLE sys_users
    ALTER COLUMN status DROP DEFAULT,
    ALTER COLUMN status TYPE smallint USING (
        CASE status WHEN 'active' THEN 0 WHEN 'disabled' THEN 1 ELSE 0 END
    ),
    ALTER COLUMN status SET DEFAULT 0,
    ALTER COLUMN status SET NOT NULL;
ALTER TABLE sys_users ADD CONSTRAINT sys_users_status_check CHECK (status IN (0, 1));
ALTER TABLE sys_users DROP CONSTRAINT users_username_unique;
DROP INDEX IF EXISTS users_email_unique;
DROP INDEX IF EXISTS users_mobile_unique;
UPDATE sys_users SET email = '' WHERE email IS NULL;
UPDATE sys_users SET mobile = '' WHERE mobile IS NULL;
ALTER TABLE sys_users
    ALTER COLUMN username TYPE varchar(30),
    ALTER COLUMN nickname TYPE varchar(30),
    ALTER COLUMN email TYPE varchar(50),
    ALTER COLUMN email SET DEFAULT '',
    ALTER COLUMN email SET NOT NULL,
    ALTER COLUMN mobile TYPE varchar(11),
    ALTER COLUMN mobile SET DEFAULT '',
    ALTER COLUMN mobile SET NOT NULL;
ALTER TABLE sys_users
    ADD COLUMN password varchar(100) NOT NULL DEFAULT '',
    ADD COLUMN remark varchar(500),
    ADD COLUMN sex smallint NOT NULL DEFAULT 0,
    ADD COLUMN avatar varchar(512) NOT NULL DEFAULT '',
    ADD COLUMN login_ip varchar(50) NOT NULL DEFAULT '',
    ADD COLUMN login_date timestamptz,
    ADD COLUMN creator varchar(64) NOT NULL DEFAULT '',
    ADD COLUMN updater varchar(64) NOT NULL DEFAULT '',
    ADD COLUMN deleted boolean NOT NULL DEFAULT false;
ALTER TABLE sys_users ADD CONSTRAINT sys_users_sex_check CHECK (sex IN (0, 1, 2));
CREATE UNIQUE INDEX sys_users_username_unique ON sys_users (username) WHERE deleted = false;
CREATE UNIQUE INDEX sys_users_email_unique ON sys_users (email) WHERE deleted = false AND email <> '';
CREATE UNIQUE INDEX sys_users_mobile_unique ON sys_users (mobile) WHERE deleted = false AND mobile <> '';

COMMENT ON TABLE sys_users IS '用户';
COMMENT ON COLUMN sys_users.username IS '用户账号';
COMMENT ON COLUMN sys_users.password IS '密码';
COMMENT ON COLUMN sys_users.nickname IS '用户昵称';
COMMENT ON COLUMN sys_users.remark IS '备注';
COMMENT ON COLUMN sys_users.email IS '用户邮箱';
COMMENT ON COLUMN sys_users.mobile IS '手机号码';
COMMENT ON COLUMN sys_users.sex IS '用户性别（0未知 1男 2女）';
COMMENT ON COLUMN sys_users.status IS '帐号状态（0正常 1停用）';
COMMENT ON COLUMN sys_users.avatar IS '头像地址';
COMMENT ON COLUMN sys_users.login_ip IS '最后登录IP';
COMMENT ON COLUMN sys_users.login_date IS '最后登录时间';
COMMENT ON COLUMN sys_users.creator IS '创建者';
COMMENT ON COLUMN sys_users.create_time IS '创建时间';
COMMENT ON COLUMN sys_users.updater IS '更新者';
COMMENT ON COLUMN sys_users.update_time IS '更新时间';
COMMENT ON COLUMN sys_users.deleted IS '是否删除';

ALTER TABLE sys_tenants RENAME COLUMN created_at TO create_time;
ALTER TABLE sys_tenants RENAME COLUMN updated_at TO update_time;
ALTER TABLE sys_tenants DROP CONSTRAINT tenants_status_check;
ALTER TABLE sys_tenants
    ALTER COLUMN status DROP DEFAULT,
    ALTER COLUMN status TYPE smallint USING (
        CASE status WHEN 'active' THEN 0 WHEN 'disabled' THEN 1 ELSE 0 END
    ),
    ALTER COLUMN status SET DEFAULT 0,
    ALTER COLUMN status SET NOT NULL,
    ALTER COLUMN name TYPE varchar(30);
ALTER TABLE sys_tenants ADD CONSTRAINT sys_tenants_status_check CHECK (status IN (0, 1));
ALTER TABLE sys_tenants DROP CONSTRAINT tenants_code_unique;
ALTER TABLE sys_tenants
    ADD COLUMN contact_user_id uuid REFERENCES sys_users (id),
    ADD COLUMN contact_name varchar(30) NOT NULL DEFAULT '',
    ADD COLUMN contact_mobile varchar(500),
    ADD COLUMN account_count integer NOT NULL DEFAULT 0,
    ADD COLUMN creator varchar(64) NOT NULL DEFAULT '',
    ADD COLUMN updater varchar(64) NOT NULL DEFAULT '',
    ADD COLUMN deleted boolean NOT NULL DEFAULT false;
CREATE UNIQUE INDEX sys_tenants_code_unique ON sys_tenants (code) WHERE deleted = false;

COMMENT ON TABLE sys_tenants IS '租户';
COMMENT ON COLUMN sys_tenants.name IS '租户名';
COMMENT ON COLUMN sys_tenants.contact_user_id IS '联系人的用户编号';
COMMENT ON COLUMN sys_tenants.contact_name IS '联系人';
COMMENT ON COLUMN sys_tenants.contact_mobile IS '联系手机';
COMMENT ON COLUMN sys_tenants.status IS '租户状态（0正常 1停用）';
COMMENT ON COLUMN sys_tenants.account_count IS '账号数量';
COMMENT ON COLUMN sys_tenants.creator IS '创建者';
COMMENT ON COLUMN sys_tenants.create_time IS '创建时间';
COMMENT ON COLUMN sys_tenants.updater IS '更新者';
COMMENT ON COLUMN sys_tenants.update_time IS '更新时间';
COMMENT ON COLUMN sys_tenants.deleted IS '是否删除';

ALTER TABLE sys_organizations RENAME COLUMN sort_order TO sort;
ALTER TABLE sys_organizations RENAME COLUMN created_at TO create_time;
ALTER TABLE sys_organizations RENAME COLUMN updated_at TO update_time;
ALTER TABLE sys_organizations DROP CONSTRAINT organizations_status_check;
ALTER TABLE sys_organizations
    ALTER COLUMN status DROP DEFAULT,
    ALTER COLUMN status TYPE smallint USING (
        CASE status WHEN 'active' THEN 0 WHEN 'disabled' THEN 1 ELSE 0 END
    ),
    ALTER COLUMN status SET NOT NULL,
    ALTER COLUMN name TYPE varchar(30),
    ALTER COLUMN name SET DEFAULT '';
ALTER TABLE sys_organizations ADD CONSTRAINT sys_organizations_status_check CHECK (status IN (0, 1));
ALTER TABLE sys_organizations DROP CONSTRAINT organizations_code_unique;
DROP INDEX IF EXISTS organizations_root_name;
DROP INDEX IF EXISTS organizations_sibling_name;
ALTER TABLE sys_organizations
    ADD COLUMN leader_user_id uuid REFERENCES sys_users (id),
    ADD COLUMN phone varchar(11),
    ADD COLUMN email varchar(50),
    ADD COLUMN creator varchar(64) NOT NULL DEFAULT '',
    ADD COLUMN updater varchar(64) NOT NULL DEFAULT '',
    ADD COLUMN deleted boolean NOT NULL DEFAULT false;
CREATE UNIQUE INDEX sys_organizations_code_unique ON sys_organizations (tenant_id, code) WHERE deleted = false;
CREATE UNIQUE INDEX sys_organizations_root_name
    ON sys_organizations (tenant_id, name)
    WHERE parent_id IS NULL AND deleted = false;
CREATE UNIQUE INDEX sys_organizations_sibling_name
    ON sys_organizations (tenant_id, parent_id, name)
    WHERE parent_id IS NOT NULL AND deleted = false;

COMMENT ON TABLE sys_organizations IS '部门';
COMMENT ON COLUMN sys_organizations.name IS '部门名称';
COMMENT ON COLUMN sys_organizations.parent_id IS '父部门id，空表示根部门';
COMMENT ON COLUMN sys_organizations.sort IS '显示顺序';
COMMENT ON COLUMN sys_organizations.leader_user_id IS '负责人';
COMMENT ON COLUMN sys_organizations.phone IS '联系电话';
COMMENT ON COLUMN sys_organizations.email IS '邮箱';
COMMENT ON COLUMN sys_organizations.status IS '部门状态（0正常 1停用）';
COMMENT ON COLUMN sys_organizations.creator IS '创建者';
COMMENT ON COLUMN sys_organizations.create_time IS '创建时间';
COMMENT ON COLUMN sys_organizations.updater IS '更新者';
COMMENT ON COLUMN sys_organizations.update_time IS '更新时间';
COMMENT ON COLUMN sys_organizations.deleted IS '是否删除';

ALTER TABLE sys_tenant_memberships RENAME COLUMN created_at TO create_time;
ALTER TABLE sys_tenant_memberships RENAME COLUMN updated_at TO update_time;
ALTER TABLE sys_tenant_memberships DROP CONSTRAINT tenant_memberships_status_check;
ALTER TABLE sys_tenant_memberships
    ALTER COLUMN status DROP DEFAULT,
    ALTER COLUMN status TYPE smallint USING (
        CASE status WHEN 'active' THEN 0 WHEN 'disabled' THEN 1 ELSE 0 END
    ),
    ALTER COLUMN status SET DEFAULT 0,
    ALTER COLUMN status SET NOT NULL;
ALTER TABLE sys_tenant_memberships ADD CONSTRAINT sys_tenant_memberships_status_check CHECK (status IN (0, 1));

ALTER TABLE sys_organization_memberships RENAME COLUMN created_at TO create_time;
ALTER TABLE sys_organization_memberships RENAME COLUMN updated_at TO update_time;
