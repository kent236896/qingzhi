CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    username text NOT NULL,
    display_name text NOT NULL,
    email text,
    mobile text,
    status text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT users_username_unique UNIQUE (username),
    CONSTRAINT users_status_check CHECK (status IN ('active', 'disabled'))
);

CREATE UNIQUE INDEX users_email_unique ON users (email) WHERE email IS NOT NULL;
CREATE UNIQUE INDEX users_mobile_unique ON users (mobile) WHERE mobile IS NOT NULL;

CREATE TABLE tenants (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL,
    code text NOT NULL,
    status text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT tenants_code_unique UNIQUE (code),
    CONSTRAINT tenants_status_check CHECK (status IN ('active', 'disabled'))
);

CREATE TABLE organizations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants (id),
    parent_id uuid,
    name text NOT NULL,
    code text NOT NULL,
    sort_order integer NOT NULL DEFAULT 0,
    status text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT organizations_parent_fk FOREIGN KEY (parent_id) REFERENCES organizations (id),
    CONSTRAINT organizations_id_tenant_unique UNIQUE (id, tenant_id),
    CONSTRAINT organizations_code_unique UNIQUE (tenant_id, code),
    CONSTRAINT organizations_status_check CHECK (status IN ('active', 'disabled'))
);

CREATE UNIQUE INDEX organizations_root_name
    ON organizations (tenant_id, name)
    WHERE parent_id IS NULL;

CREATE UNIQUE INDEX organizations_sibling_name
    ON organizations (tenant_id, parent_id, name)
    WHERE parent_id IS NOT NULL;

CREATE INDEX organizations_tenant_parent ON organizations (tenant_id, parent_id);

CREATE TABLE tenant_memberships (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES tenants (id),
    user_id uuid NOT NULL REFERENCES users (id),
    status text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT tenant_memberships_unique UNIQUE (tenant_id, user_id),
    CONSTRAINT tenant_memberships_status_check CHECK (status IN ('active', 'disabled'))
);

CREATE INDEX tenant_memberships_user ON tenant_memberships (user_id);

CREATE TABLE organization_memberships (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL,
    organization_id uuid NOT NULL,
    user_id uuid NOT NULL,
    is_primary boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT organization_memberships_unique UNIQUE (tenant_id, user_id, organization_id),
    CONSTRAINT organization_memberships_member_fk
        FOREIGN KEY (tenant_id, user_id) REFERENCES tenant_memberships (tenant_id, user_id),
    CONSTRAINT organization_memberships_org_tenant_fk
        FOREIGN KEY (organization_id, tenant_id) REFERENCES organizations (id, tenant_id)
);

CREATE UNIQUE INDEX organization_memberships_one_primary
    ON organization_memberships (tenant_id, user_id)
    WHERE is_primary;

CREATE INDEX organization_memberships_user ON organization_memberships (tenant_id, user_id);
