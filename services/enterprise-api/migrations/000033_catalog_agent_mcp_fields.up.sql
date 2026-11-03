-- Agent/MCP 目录字段演进（不建平行表）：配置快照、MCP URL、auth_mode。
ALTER TABLE sys_agents
    ADD COLUMN IF NOT EXISTS config_json jsonb NOT NULL DEFAULT '{}'::jsonb;

COMMENT ON COLUMN sys_agents.config_json IS '智能体配置快照（不下发远端密钥；远端 id 仍存 weknora_agent_id）';

ALTER TABLE sys_mcp_services
    ADD COLUMN IF NOT EXISTS url varchar(1000) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS auth_mode varchar(16) NOT NULL DEFAULT 'shared';

ALTER TABLE sys_mcp_services
    DROP CONSTRAINT IF EXISTS sys_mcp_services_auth_mode_check;

ALTER TABLE sys_mcp_services
    ADD CONSTRAINT sys_mcp_services_auth_mode_check
    CHECK (auth_mode IN ('shared', 'user_oauth'));

COMMENT ON COLUMN sys_mcp_services.url IS 'MCP 服务地址（与 WeKnora 同步的平台侧副本）';
COMMENT ON COLUMN sys_mcp_services.auth_mode IS 'shared=空间静态凭证；user_oauth=每用户 OAuth；不做个人静态 API Key Gateway';
