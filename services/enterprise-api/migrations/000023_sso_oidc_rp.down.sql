ALTER TABLE sys_sso_settings
    DROP COLUMN IF EXISTS frontend_base_url,
    DROP COLUMN IF EXISTS default_tenant_id;

DROP TABLE IF EXISTS sys_sso_login_tickets;
DROP TABLE IF EXISTS sys_sso_oidc_auth_requests;
