-- 回滚客户端收敛：恢复 AS/IdP 登记表与菜单（不含业务数据）。

ALTER TABLE sys_sso_settings DROP COLUMN IF EXISTS sp_key_secret_id;
ALTER TABLE sys_sso_settings DROP COLUMN IF EXISTS sp_certificate_pem;

CREATE TABLE IF NOT EXISTS sys_sso_oauth_clients (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id varchar(128) NOT NULL,
    name varchar(128) NOT NULL,
    description varchar(500) NOT NULL DEFAULT '',
    client_type varchar(32) NOT NULL,
    status varchar(32) NOT NULL DEFAULT 'disabled',
    redirect_uris jsonb NOT NULL DEFAULT '[]'::jsonb,
    grant_types jsonb NOT NULL DEFAULT '["authorization_code"]'::jsonb,
    scopes jsonb NOT NULL DEFAULT '["openid"]'::jsonb,
    pkce_policy varchar(32) NOT NULL DEFAULT 'optional',
    access_token_ttl_sec int NOT NULL DEFAULT 3600,
    refresh_token_ttl_sec int NOT NULL DEFAULT 2592000,
    id_token_ttl_sec int NOT NULL DEFAULT 3600,
    secret_id uuid REFERENCES sys_sso_secrets (id),
    last_used_at timestamptz,
    deleted boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    CONSTRAINT sys_sso_oauth_clients_type_check CHECK (client_type IN ('public', 'confidential')),
    CONSTRAINT sys_sso_oauth_clients_status_check CHECK (status IN ('enabled', 'disabled', 'revoked')),
    CONSTRAINT sys_sso_oauth_clients_pkce_check CHECK (pkce_policy IN ('required', 'optional', 'forbidden')),
    CONSTRAINT sys_sso_oauth_clients_ttl_check CHECK (
        access_token_ttl_sec >= 60 AND access_token_ttl_sec <= 86400
        AND id_token_ttl_sec >= 60 AND id_token_ttl_sec <= 86400
        AND refresh_token_ttl_sec >= 0 AND refresh_token_ttl_sec <= 31536000
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS sys_sso_oauth_clients_client_id
    ON sys_sso_oauth_clients (client_id) WHERE deleted = false;
CREATE INDEX IF NOT EXISTS sys_sso_oauth_clients_list
    ON sys_sso_oauth_clients (status, updated_at DESC) WHERE deleted = false;

CREATE TABLE IF NOT EXISTS sys_sso_saml_sps (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name varchar(128) NOT NULL,
    entity_id varchar(512) NOT NULL,
    status varchar(32) NOT NULL DEFAULT 'disabled',
    acs_urls jsonb NOT NULL DEFAULT '[]'::jsonb,
    slo_url varchar(512) NOT NULL DEFAULT '',
    name_id_format varchar(128) NOT NULL DEFAULT 'urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress',
    attribute_mapping jsonb NOT NULL DEFAULT '{}'::jsonb,
    allow_idp_initiated boolean NOT NULL DEFAULT false,
    certificate_pem text NOT NULL DEFAULT '',
    certificate_not_before timestamptz,
    certificate_not_after timestamptz,
    deleted boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT NOW(),
    updated_at timestamptz NOT NULL DEFAULT NOW(),
    CONSTRAINT sys_sso_saml_sps_status_check CHECK (status IN ('enabled', 'disabled'))
);

CREATE UNIQUE INDEX IF NOT EXISTS sys_sso_saml_sps_entity
    ON sys_sso_saml_sps (entity_id) WHERE deleted = false;
CREATE INDEX IF NOT EXISTS sys_sso_saml_sps_list
    ON sys_sso_saml_sps (status, updated_at DESC) WHERE deleted = false;

ALTER TABLE sys_sso_providers DROP CONSTRAINT IF EXISTS sys_sso_providers_protocol_check;
ALTER TABLE sys_sso_providers
    ADD CONSTRAINT sys_sso_providers_protocol_check CHECK (protocol IN ('oidc', 'oauth2', 'saml'));

UPDATE sys_menus
SET deleted = false, status = 0, visible = true, update_time = NOW()
WHERE id IN (
    'a1000000-0000-4000-8000-000000000032',
    'a1000000-0000-4000-8000-000000000033',
    'a1000000-0000-4000-8000-000000000233',
    'a1000000-0000-4000-8000-000000000234',
    'a1000000-0000-4000-8000-000000000235',
    'a1000000-0000-4000-8000-000000000236'
);

INSERT INTO sys_role_menus (tenant_id, role_id, menu_id)
SELECT r.tenant_id, r.id, m.id
FROM sys_roles r
JOIN sys_menus m ON m.id IN (
    'a1000000-0000-4000-8000-000000000032',
    'a1000000-0000-4000-8000-000000000033',
    'a1000000-0000-4000-8000-000000000233',
    'a1000000-0000-4000-8000-000000000234',
    'a1000000-0000-4000-8000-000000000235',
    'a1000000-0000-4000-8000-000000000236'
)
WHERE r.code = 'super_admin' AND r.deleted = false
ON CONFLICT DO NOTHING;
