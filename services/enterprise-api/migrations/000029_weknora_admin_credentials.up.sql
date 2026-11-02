-- WeKnora 租户管理员账号：用于登录换 JWT，再配置 api-principal-config（signed_token）。
ALTER TABLE sys_weknora_settings
    ADD COLUMN IF NOT EXISTS admin_email varchar(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS admin_password text NOT NULL DEFAULT '';

COMMENT ON COLUMN sys_weknora_settings.admin_email IS 'WeKnora 租户管理员邮箱；接口不返回密码明文';
COMMENT ON COLUMN sys_weknora_settings.admin_password IS 'WeKnora 租户管理员密码（仅服务端保存）；接口不返回明文';
