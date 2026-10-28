-- Persist the browser origin chosen at SSO start so the IdP callback can redirect safely.

ALTER TABLE sys_sso_oidc_auth_requests
    ADD COLUMN IF NOT EXISTS frontend_base_url varchar(512) NOT NULL DEFAULT '';

COMMENT ON COLUMN sys_sso_oidc_auth_requests.frontend_base_url IS '登录发起时的前端基址（须匹配 CORS 或已保存的 SSO 前端基址）';
