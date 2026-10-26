-- OIDC / OAuth2 客户端登录运行时：授权请求与一次性登录票据。

CREATE TABLE sys_sso_oidc_auth_requests (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    state varchar(128) NOT NULL,
    nonce varchar(128) NOT NULL,
    code_verifier varchar(128) NOT NULL DEFAULT '',
    provider_id uuid NOT NULL REFERENCES sys_sso_providers (id),
    protocol varchar(16) NOT NULL DEFAULT 'oidc',
    redirect_path varchar(512) NOT NULL DEFAULT '/',
    tenant_id varchar(64) NOT NULL DEFAULT '',
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT sys_sso_oidc_auth_requests_protocol_check CHECK (protocol IN ('oidc', 'oauth2'))
);

CREATE UNIQUE INDEX sys_sso_oidc_auth_requests_state ON sys_sso_oidc_auth_requests (state);
CREATE INDEX sys_sso_oidc_auth_requests_expires ON sys_sso_oidc_auth_requests (expires_at);

COMMENT ON TABLE sys_sso_oidc_auth_requests IS '外部登录授权请求（state/nonce/PKCE），一次性消费';
COMMENT ON COLUMN sys_sso_oidc_auth_requests.redirect_path IS '登录成功后前端相对路径，禁止开放重定向';

CREATE TABLE sys_sso_login_tickets (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket varchar(64) NOT NULL,
    user_id uuid NOT NULL,
    tenant_id varchar(64) NOT NULL,
    provider_id uuid NOT NULL,
    expires_at timestamptz NOT NULL,
    consumed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX sys_sso_login_tickets_ticket ON sys_sso_login_tickets (ticket);
CREATE INDEX sys_sso_login_tickets_expires ON sys_sso_login_tickets (expires_at);

COMMENT ON TABLE sys_sso_login_tickets IS 'OIDC/OAuth2 回调后的一次性登录票据，前端兑换本平台 JWT';

ALTER TABLE sys_sso_settings
    ADD COLUMN IF NOT EXISTS default_tenant_id varchar(64) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS frontend_base_url varchar(512) NOT NULL DEFAULT '';

COMMENT ON COLUMN sys_sso_settings.default_tenant_id IS '外部自动开户时加入的默认租户';
COMMENT ON COLUMN sys_sso_settings.frontend_base_url IS '登录回调跳转的前端基址，须与 CORS 白名单匹配';
