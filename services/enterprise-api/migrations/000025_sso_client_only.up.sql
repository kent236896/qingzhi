-- 收敛为仅外部 IdP 客户端登录（OIDC RP + SAML SP）；移除本平台作为 AS/OP/IdP 的登记能力。

UPDATE sys_menus
SET deleted = true, status = 1, visible = false, update_time = NOW()
WHERE id IN (
    'a1000000-0000-4000-8000-000000000032',
    'a1000000-0000-4000-8000-000000000033',
    'a1000000-0000-4000-8000-000000000233',
    'a1000000-0000-4000-8000-000000000234',
    'a1000000-0000-4000-8000-000000000235',
    'a1000000-0000-4000-8000-000000000236'
);

DELETE FROM sys_role_menus
WHERE menu_id IN (
    'a1000000-0000-4000-8000-000000000032',
    'a1000000-0000-4000-8000-000000000033',
    'a1000000-0000-4000-8000-000000000233',
    'a1000000-0000-4000-8000-000000000234',
    'a1000000-0000-4000-8000-000000000235',
    'a1000000-0000-4000-8000-000000000236'
);

-- CHECK 约束作用于全部行（含已软删）；须先去掉 oauth2，否则 ADD CONSTRAINT 会失败并留下 dirty。
DELETE FROM sys_sso_providers WHERE protocol = 'oauth2';

ALTER TABLE sys_sso_providers DROP CONSTRAINT IF EXISTS sys_sso_providers_protocol_check;
ALTER TABLE sys_sso_providers
    ADD CONSTRAINT sys_sso_providers_protocol_check CHECK (protocol IN ('oidc', 'saml'));

DROP TABLE IF EXISTS sys_sso_oauth_clients;
DROP TABLE IF EXISTS sys_sso_saml_sps;

ALTER TABLE sys_sso_settings
    ADD COLUMN sp_certificate_pem text NOT NULL DEFAULT '';
ALTER TABLE sys_sso_settings
    ADD COLUMN sp_key_secret_id uuid REFERENCES sys_sso_secrets (id);

COMMENT ON COLUMN sys_sso_settings.issuer IS '已废弃：本平台不再作为 OP/IdP；保留列兼容旧数据';
COMMENT ON COLUMN sys_sso_settings.sp_certificate_pem IS '本平台 SAML SP 证书 PEM（客户端登录）';
COMMENT ON COLUMN sys_sso_settings.sp_key_secret_id IS '本平台 SAML SP 私钥密文引用';
COMMENT ON TABLE sys_sso_secrets IS '联邦客户端密文（外部 IdP Client Secret、SAML SP 私钥等）';
