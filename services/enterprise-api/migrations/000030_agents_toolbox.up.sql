-- 智能体 / MCP 工具箱：平台侧可见性（个人、租户、组织、角色），远端实体在 WeKnora。
CREATE TABLE sys_agents (
    id uuid PRIMARY KEY,
    name varchar(100) NOT NULL,
    description varchar(500) NOT NULL DEFAULT '',
    weknora_agent_id varchar(64) NOT NULL,
    scope_type varchar(16) NOT NULL,
    scope_id uuid NOT NULL,
    status integer NOT NULL DEFAULT 0,
    creator_id uuid NOT NULL,
    creator varchar(64) NOT NULL DEFAULT '',
    create_time timestamptz NOT NULL DEFAULT now(),
    updater varchar(64) NOT NULL DEFAULT '',
    update_time timestamptz NOT NULL DEFAULT now(),
    deleted boolean NOT NULL DEFAULT false,
    CONSTRAINT sys_agents_scope_check CHECK (scope_type IN ('personal', 'tenant', 'org_node', 'role')),
    CONSTRAINT sys_agents_status_check CHECK (status IN (0, 1))
);

CREATE UNIQUE INDEX sys_agents_remote ON sys_agents (weknora_agent_id) WHERE deleted = false;
CREATE UNIQUE INDEX sys_agents_name ON sys_agents (scope_type, scope_id, name) WHERE deleted = false;
CREATE INDEX sys_agents_scope ON sys_agents (scope_type, scope_id) WHERE deleted = false;

COMMENT ON TABLE sys_agents IS '智能体目录；远端编号不对外暴露';
COMMENT ON COLUMN sys_agents.scope_type IS '归属（personal 个人 tenant 租户 org_node 组织 role 角色）';

CREATE TABLE sys_agent_tenants (
    id uuid PRIMARY KEY,
    agent_id uuid NOT NULL REFERENCES sys_agents (id),
    tenant_id uuid NOT NULL REFERENCES sys_tenants (id),
    deleted boolean NOT NULL DEFAULT false
);
CREATE UNIQUE INDEX sys_agent_tenants_unique ON sys_agent_tenants (agent_id, tenant_id) WHERE deleted = false;

CREATE TABLE sys_agent_shares (
    id uuid PRIMARY KEY,
    agent_id uuid NOT NULL REFERENCES sys_agents (id),
    user_id uuid NOT NULL REFERENCES sys_users (id),
    creator varchar(64) NOT NULL DEFAULT '',
    create_time timestamptz NOT NULL DEFAULT now(),
    deleted boolean NOT NULL DEFAULT false
);
CREATE UNIQUE INDEX sys_agent_shares_unique ON sys_agent_shares (agent_id, user_id) WHERE deleted = false;

CREATE TABLE sys_mcp_services (
    id uuid PRIMARY KEY,
    name varchar(100) NOT NULL,
    description varchar(500) NOT NULL DEFAULT '',
    weknora_mcp_id varchar(64) NOT NULL,
    transport_type varchar(32) NOT NULL DEFAULT 'sse',
    scope_type varchar(16) NOT NULL,
    scope_id uuid NOT NULL,
    status integer NOT NULL DEFAULT 0,
    creator_id uuid NOT NULL,
    creator varchar(64) NOT NULL DEFAULT '',
    create_time timestamptz NOT NULL DEFAULT now(),
    updater varchar(64) NOT NULL DEFAULT '',
    update_time timestamptz NOT NULL DEFAULT now(),
    deleted boolean NOT NULL DEFAULT false,
    CONSTRAINT sys_mcp_services_scope_check CHECK (scope_type IN ('personal', 'tenant', 'org_node', 'role')),
    CONSTRAINT sys_mcp_services_status_check CHECK (status IN (0, 1))
);

CREATE UNIQUE INDEX sys_mcp_services_remote ON sys_mcp_services (weknora_mcp_id) WHERE deleted = false;
CREATE UNIQUE INDEX sys_mcp_services_name ON sys_mcp_services (scope_type, scope_id, name) WHERE deleted = false;
CREATE INDEX sys_mcp_services_scope ON sys_mcp_services (scope_type, scope_id) WHERE deleted = false;

COMMENT ON TABLE sys_mcp_services IS '工具箱（仅 MCP）；远端编号不对外暴露';

CREATE TABLE sys_mcp_service_tenants (
    id uuid PRIMARY KEY,
    mcp_service_id uuid NOT NULL REFERENCES sys_mcp_services (id),
    tenant_id uuid NOT NULL REFERENCES sys_tenants (id),
    deleted boolean NOT NULL DEFAULT false
);
CREATE UNIQUE INDEX sys_mcp_service_tenants_unique ON sys_mcp_service_tenants (mcp_service_id, tenant_id) WHERE deleted = false;

CREATE TABLE sys_mcp_service_shares (
    id uuid PRIMARY KEY,
    mcp_service_id uuid NOT NULL REFERENCES sys_mcp_services (id),
    user_id uuid NOT NULL REFERENCES sys_users (id),
    creator varchar(64) NOT NULL DEFAULT '',
    create_time timestamptz NOT NULL DEFAULT now(),
    deleted boolean NOT NULL DEFAULT false
);
CREATE UNIQUE INDEX sys_mcp_service_shares_unique ON sys_mcp_service_shares (mcp_service_id, user_id) WHERE deleted = false;

-- 管理端菜单
INSERT INTO sys_menus (
    id, name, permission, type, sort, parent_id, path, icon,
    component, component_name, status, visible, keep_alive, always_show
) VALUES
    (
        'a1000000-0000-4000-8000-000000000030', '智能体', 'agent:query', 2, 16, NULL, '/agents', '#',
        'views/AgentsView.vue', 'AgentsView', 0, true, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000031', '查询', 'agent:query', 3, 1,
        'a1000000-0000-4000-8000-000000000030', '', '#', NULL, NULL, 0, false, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000032', '管理', 'agent:manage', 3, 2,
        'a1000000-0000-4000-8000-000000000030', '', '#', NULL, NULL, 0, false, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000033', '工具箱', 'toolbox:query', 2, 17, NULL, '/toolbox', '#',
        'views/ToolboxView.vue', 'ToolboxView', 0, true, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000034', '查询', 'toolbox:query', 3, 1,
        'a1000000-0000-4000-8000-000000000033', '', '#', NULL, NULL, 0, false, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000035', '管理', 'toolbox:manage', 3, 2,
        'a1000000-0000-4000-8000-000000000033', '', '#', NULL, NULL, 0, false, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000036', '产物', 'artifact:query', 2, 18, NULL, '/artifacts', '#',
        'views/ArtifactsView.vue', 'ArtifactsView', 0, true, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000037', '查询', 'artifact:query', 3, 1,
        'a1000000-0000-4000-8000-000000000036', '', '#', NULL, NULL, 0, false, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000038', '管理', 'artifact:manage', 3, 2,
        'a1000000-0000-4000-8000-000000000036', '', '#', NULL, NULL, 0, false, true, true
    )
ON CONFLICT (id) DO NOTHING;

INSERT INTO sys_role_menus (tenant_id, role_id, menu_id)
SELECT r.tenant_id, r.id, m.id
FROM sys_roles r
JOIN sys_menus m ON m.id IN (
    'a1000000-0000-4000-8000-000000000030',
    'a1000000-0000-4000-8000-000000000031',
    'a1000000-0000-4000-8000-000000000033',
    'a1000000-0000-4000-8000-000000000034',
    'a1000000-0000-4000-8000-000000000036',
    'a1000000-0000-4000-8000-000000000037'
)
WHERE r.deleted = false
  AND NOT EXISTS (
      SELECT 1 FROM sys_role_menus existing
      WHERE existing.tenant_id = r.tenant_id
        AND existing.role_id = r.id
        AND existing.menu_id = m.id
        AND existing.deleted = false
  );

INSERT INTO sys_role_menus (tenant_id, role_id, menu_id)
SELECT r.tenant_id, r.id, m.id
FROM sys_roles r
JOIN sys_menus m ON m.id IN (
    'a1000000-0000-4000-8000-000000000032',
    'a1000000-0000-4000-8000-000000000035',
    'a1000000-0000-4000-8000-000000000038'
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
