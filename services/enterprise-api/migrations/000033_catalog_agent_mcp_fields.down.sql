ALTER TABLE sys_mcp_services DROP CONSTRAINT IF EXISTS sys_mcp_services_auth_mode_check;
ALTER TABLE sys_mcp_services DROP COLUMN IF EXISTS auth_mode;
ALTER TABLE sys_mcp_services DROP COLUMN IF EXISTS url;
ALTER TABLE sys_agents DROP COLUMN IF EXISTS config_json;
