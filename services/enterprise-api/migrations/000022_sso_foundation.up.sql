-- 统一身份认证与 SSO 基础表、系统安全菜单。协议运行时表见后续迁移。

CREATE TABLE sys_sso_settings (
    id smallint PRIMARY KEY,
    local_login_enabled boolean NOT NULL DEFAULT true,
    issuer varchar(512) NOT NULL DEFAULT '',
    updated_by varchar(64) NOT NULL DEFAULT '',
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT sys_sso_settings_singleton CHECK (id = 1)
);

INSERT INTO sys_sso_settings (id) VALUES (1);

COMMENT ON TABLE sys_sso_settings IS 'SSO 全局策略（本地登录应急开关、受控 Issuer）';
COMMENT ON COLUMN sys_sso_settings.local_login_enabled IS '是否允许本地账号密码登录（应急访问）';
COMMENT ON COLUMN sys_sso_settings.issuer IS '本平台作为 OP/IdP 时的固定 Issuer，禁止按 Host 动态生成';

CREATE TABLE sys_sso_secrets (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    kind varchar(32) NOT NULL,
    ciphertext bytea NOT NULL,
    key_version integer NOT NULL DEFAULT 1,
    hint varchar(16) NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    revoked_at timestamptz,
    CONSTRAINT sys_sso_secrets_kind_check CHECK (
        kind IN ('provider_client_secret', 'oauth_client_secret', 'oidc_signing_key', 'saml_signing_key')
    )
);

CREATE INDEX sys_sso_secrets_kind ON sys_sso_secrets (kind) WHERE revoked_at IS NULL;

COMMENT ON TABLE sys_sso_secrets IS '联邦与授权服务端密文，管理 API 永不返回明文';
COMMENT ON COLUMN sys_sso_secrets.ciphertext IS 'AES-GCM 密文';
COMMENT ON COLUMN sys_sso_secrets.hint IS '脱敏提示，如 Secret 末四位';

CREATE TABLE sys_sso_providers (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name varchar(128) NOT NULL,
    protocol varchar(16) NOT NULL,
    status varchar(16) NOT NULL DEFAULT 'disabled',
    priority integer NOT NULL DEFAULT 100,
    issuer varchar(512) NOT NULL DEFAULT '',
    display_on_login boolean NOT NULL DEFAULT true,
    auto_login boolean NOT NULL DEFAULT false,
    auto_create_user boolean NOT NULL DEFAULT false,
    discovery_url varchar(512) NOT NULL DEFAULT '',
    authorization_url varchar(512) NOT NULL DEFAULT '',
    token_url varchar(512) NOT NULL DEFAULT '',
    userinfo_url varchar(512) NOT NULL DEFAULT '',
    jwks_url varchar(512) NOT NULL DEFAULT '',
    client_id varchar(256) NOT NULL DEFAULT '',
    secret_id uuid REFERENCES sys_sso_secrets (id),
    redirect_uri varchar(512) NOT NULL DEFAULT '',
    scopes varchar(512) NOT NULL DEFAULT '',
    claim_mapping jsonb NOT NULL DEFAULT '{}'::jsonb,
    entity_id varchar(512) NOT NULL DEFAULT '',
    metadata_url varchar(512) NOT NULL DEFAULT '',
    sso_url varchar(512) NOT NULL DEFAULT '',
    slo_url varchar(512) NOT NULL DEFAULT '',
    idp_certificate_pem text NOT NULL DEFAULT '',
    certificate_not_before timestamptz,
    certificate_not_after timestamptz,
    sp_entity_id varchar(512) NOT NULL DEFAULT '',
    acs_url varchar(512) NOT NULL DEFAULT '',
    name_id_format varchar(128) NOT NULL DEFAULT '',
    last_success_at timestamptz,
    last_error varchar(500) NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted boolean NOT NULL DEFAULT false,
    CONSTRAINT sys_sso_providers_protocol_check CHECK (protocol IN ('oidc', 'oauth2', 'saml')),
    CONSTRAINT sys_sso_providers_status_check CHECK (status IN ('enabled', 'disabled'))
);

CREATE INDEX sys_sso_providers_list ON sys_sso_providers (status, priority, updated_at DESC) WHERE deleted = false;
CREATE INDEX sys_sso_providers_protocol ON sys_sso_providers (protocol) WHERE deleted = false;

COMMENT ON TABLE sys_sso_providers IS '外部身份提供方（本平台作为客户端/SP）';
COMMENT ON COLUMN sys_sso_providers.auto_create_user IS '自动开户，默认关闭；开启后仅最小权限';
COMMENT ON COLUMN sys_sso_providers.claim_mapping IS '属性映射 JSON，不含密钥';
COMMENT ON COLUMN sys_sso_providers.secret_id IS '指向加密后的 Client Secret';

CREATE TABLE sys_sso_oauth_clients (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id varchar(64) NOT NULL,
    name varchar(128) NOT NULL,
    description varchar(500) NOT NULL DEFAULT '',
    client_type varchar(16) NOT NULL,
    status varchar(16) NOT NULL DEFAULT 'disabled',
    redirect_uris jsonb NOT NULL DEFAULT '[]'::jsonb,
    grant_types jsonb NOT NULL DEFAULT '["authorization_code"]'::jsonb,
    scopes jsonb NOT NULL DEFAULT '["openid"]'::jsonb,
    pkce_policy varchar(16) NOT NULL DEFAULT 'required',
    access_token_ttl_sec integer NOT NULL DEFAULT 3600,
    refresh_token_ttl_sec integer NOT NULL DEFAULT 2592000,
    id_token_ttl_sec integer NOT NULL DEFAULT 3600,
    secret_id uuid REFERENCES sys_sso_secrets (id),
    last_used_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted boolean NOT NULL DEFAULT false,
    CONSTRAINT sys_sso_oauth_clients_type_check CHECK (client_type IN ('public', 'confidential')),
    CONSTRAINT sys_sso_oauth_clients_status_check CHECK (status IN ('enabled', 'disabled', 'revoked')),
    CONSTRAINT sys_sso_oauth_clients_pkce_check CHECK (pkce_policy IN ('required', 'optional', 'forbidden')),
    CONSTRAINT sys_sso_oauth_clients_ttl_check CHECK (
        access_token_ttl_sec BETWEEN 60 AND 86400
        AND refresh_token_ttl_sec BETWEEN 0 AND 31536000
        AND id_token_ttl_sec BETWEEN 60 AND 86400
    )
);

CREATE UNIQUE INDEX sys_sso_oauth_clients_client_id ON sys_sso_oauth_clients (client_id) WHERE deleted = false;
CREATE INDEX sys_sso_oauth_clients_list ON sys_sso_oauth_clients (status, updated_at DESC) WHERE deleted = false;

COMMENT ON TABLE sys_sso_oauth_clients IS '本平台作为 AS/OP 时注册的 OAuth2/OIDC 客户端';
COMMENT ON COLUMN sys_sso_oauth_clients.redirect_uris IS '精确匹配白名单，禁止通配';
COMMENT ON COLUMN sys_sso_oauth_clients.secret_id IS '机密客户端 Secret 密文引用';

CREATE TABLE sys_sso_saml_sps (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name varchar(128) NOT NULL,
    entity_id varchar(512) NOT NULL,
    status varchar(16) NOT NULL DEFAULT 'disabled',
    acs_urls jsonb NOT NULL DEFAULT '[]'::jsonb,
    slo_url varchar(512) NOT NULL DEFAULT '',
    name_id_format varchar(128) NOT NULL DEFAULT 'urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress',
    attribute_mapping jsonb NOT NULL DEFAULT '{}'::jsonb,
    allow_idp_initiated boolean NOT NULL DEFAULT false,
    certificate_pem text NOT NULL DEFAULT '',
    certificate_not_before timestamptz,
    certificate_not_after timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted boolean NOT NULL DEFAULT false,
    CONSTRAINT sys_sso_saml_sps_status_check CHECK (status IN ('enabled', 'disabled'))
);

CREATE UNIQUE INDEX sys_sso_saml_sps_entity ON sys_sso_saml_sps (entity_id) WHERE deleted = false;
CREATE INDEX sys_sso_saml_sps_list ON sys_sso_saml_sps (status, updated_at DESC) WHERE deleted = false;

COMMENT ON TABLE sys_sso_saml_sps IS '本平台作为 SAML IdP 时登记的服务提供方';
COMMENT ON COLUMN sys_sso_saml_sps.allow_idp_initiated IS 'IdP-initiated SSO，默认关闭';
COMMENT ON COLUMN sys_sso_saml_sps.acs_urls IS 'ACS URL 精确白名单';

CREATE TABLE sys_sso_identity_links (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id uuid NOT NULL REFERENCES sys_sso_providers (id),
    external_subject varchar(256) NOT NULL,
    user_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted boolean NOT NULL DEFAULT false,
    CONSTRAINT sys_sso_identity_links_subject_check CHECK (char_length(trim(external_subject)) > 0)
);

CREATE UNIQUE INDEX sys_sso_identity_links_unique
    ON sys_sso_identity_links (provider_id, external_subject) WHERE deleted = false;
CREATE INDEX sys_sso_identity_links_user ON sys_sso_identity_links (user_id) WHERE deleted = false;

COMMENT ON TABLE sys_sso_identity_links IS '外部 Subject/NameID 与本平台用户映射；禁止仅凭邮箱静默合并';

CREATE TABLE sys_sso_auth_events (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type varchar(64) NOT NULL,
    result varchar(16) NOT NULL,
    protocol varchar(16) NOT NULL DEFAULT '',
    provider_id varchar(64) NOT NULL DEFAULT '',
    client_id varchar(64) NOT NULL DEFAULT '',
    user_id varchar(64) NOT NULL DEFAULT '',
    username varchar(64) NOT NULL DEFAULT '',
    message varchar(500) NOT NULL DEFAULT '',
    request_id varchar(64) NOT NULL DEFAULT '',
    ip varchar(64) NOT NULL DEFAULT '',
    create_time timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT sys_sso_auth_events_result_check CHECK (result IN ('success', 'failure'))
);

CREATE INDEX sys_sso_auth_events_time ON sys_sso_auth_events (create_time DESC);
CREATE INDEX sys_sso_auth_events_type ON sys_sso_auth_events (event_type, create_time DESC);
CREATE INDEX sys_sso_auth_events_protocol ON sys_sso_auth_events (protocol, create_time DESC);

COMMENT ON TABLE sys_sso_auth_events IS '认证与联邦安全事件；禁止记录密码、完整令牌、Secret、私钥或完整断言';

INSERT INTO sys_menus (
    id, name, permission, type, sort, parent_id, path, icon,
    component, component_name, status, visible, keep_alive, always_show
) VALUES
    (
        'a1000000-0000-4000-8000-000000000005', '系统安全', '', 1, 35, NULL, '', '#',
        NULL, NULL, 0, true, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000030', '身份认证与单点登录', 'system:sso:overview', 2, 10,
        'a1000000-0000-4000-8000-000000000005', '/security/sso', '#',
        'views/sso/OverviewView.vue', 'SsoOverviewView', 0, true, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000031', '外部身份提供方', 'system:sso:provider:query', 2, 20,
        'a1000000-0000-4000-8000-000000000005', '/security/sso/providers', '#',
        'views/sso/ProvidersView.vue', 'SsoProvidersView', 0, true, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000032', 'OAuth 客户端', 'system:sso:oauth-client:query', 2, 30,
        'a1000000-0000-4000-8000-000000000005', '/security/sso/oauth-clients', '#',
        'views/sso/OAuthClientsView.vue', 'SsoOAuthClientsView', 0, true, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000033', 'SAML 服务提供方', 'system:sso:saml-sp:query', 2, 40,
        'a1000000-0000-4000-8000-000000000005', '/security/sso/saml-sps', '#',
        'views/sso/SamlSpsView.vue', 'SsoSamlSpsView', 0, true, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000034', '身份映射', 'system:sso:identity-link:query', 2, 50,
        'a1000000-0000-4000-8000-000000000005', '/security/sso/identity-links', '#',
        'views/sso/IdentityLinksView.vue', 'SsoIdentityLinksView', 0, true, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000035', '认证审计', 'system:sso:auth-event:query', 2, 60,
        'a1000000-0000-4000-8000-000000000005', '/security/sso/auth-events', '#',
        'views/sso/AuthEventsView.vue', 'SsoAuthEventsView', 0, true, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000230', '查询', 'system:sso:overview', 3, 1,
        'a1000000-0000-4000-8000-000000000030', '', '#', NULL, NULL, 0, false, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000231', '查询', 'system:sso:provider:query', 3, 1,
        'a1000000-0000-4000-8000-000000000031', '', '#', NULL, NULL, 0, false, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000232', '管理', 'system:sso:provider:manage', 3, 2,
        'a1000000-0000-4000-8000-000000000031', '', '#', NULL, NULL, 0, false, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000233', '查询', 'system:sso:oauth-client:query', 3, 1,
        'a1000000-0000-4000-8000-000000000032', '', '#', NULL, NULL, 0, false, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000234', '管理', 'system:sso:oauth-client:manage', 3, 2,
        'a1000000-0000-4000-8000-000000000032', '', '#', NULL, NULL, 0, false, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000235', '查询', 'system:sso:saml-sp:query', 3, 1,
        'a1000000-0000-4000-8000-000000000033', '', '#', NULL, NULL, 0, false, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000236', '管理', 'system:sso:saml-sp:manage', 3, 2,
        'a1000000-0000-4000-8000-000000000033', '', '#', NULL, NULL, 0, false, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000237', '查询', 'system:sso:identity-link:query', 3, 1,
        'a1000000-0000-4000-8000-000000000034', '', '#', NULL, NULL, 0, false, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000238', '管理', 'system:sso:identity-link:manage', 3, 2,
        'a1000000-0000-4000-8000-000000000034', '', '#', NULL, NULL, 0, false, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000239', '查询', 'system:sso:auth-event:query', 3, 1,
        'a1000000-0000-4000-8000-000000000035', '', '#', NULL, NULL, 0, false, true, true
    );

INSERT INTO sys_role_menus (tenant_id, role_id, menu_id)
SELECT r.tenant_id, r.id, m.id
FROM sys_roles r
JOIN sys_menus m ON m.id IN (
    'a1000000-0000-4000-8000-000000000005',
    'a1000000-0000-4000-8000-000000000030',
    'a1000000-0000-4000-8000-000000000031',
    'a1000000-0000-4000-8000-000000000032',
    'a1000000-0000-4000-8000-000000000033',
    'a1000000-0000-4000-8000-000000000034',
    'a1000000-0000-4000-8000-000000000035',
    'a1000000-0000-4000-8000-000000000230',
    'a1000000-0000-4000-8000-000000000231',
    'a1000000-0000-4000-8000-000000000232',
    'a1000000-0000-4000-8000-000000000233',
    'a1000000-0000-4000-8000-000000000234',
    'a1000000-0000-4000-8000-000000000235',
    'a1000000-0000-4000-8000-000000000236',
    'a1000000-0000-4000-8000-000000000237',
    'a1000000-0000-4000-8000-000000000238',
    'a1000000-0000-4000-8000-000000000239'
)
WHERE r.deleted = false
  AND r.code = 'super_admin'
  AND NOT EXISTS (
      SELECT 1 FROM sys_role_menus existing
      WHERE existing.tenant_id = r.tenant_id
        AND existing.role_id = r.id
        AND existing.menu_id = m.id
        AND existing.deleted = false
  );
