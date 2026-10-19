CREATE TABLE sys_knowledge_bases (
    id uuid PRIMARY KEY,
    name varchar(100) NOT NULL,
    description varchar(500) NOT NULL DEFAULT '',
    weknora_kb_id varchar(64) NOT NULL,
    scope_type varchar(16) NOT NULL,
    scope_id uuid NOT NULL,
    wiki_enabled boolean NOT NULL DEFAULT false,
    status integer NOT NULL DEFAULT 0,
    creator_id uuid NOT NULL,
    wiki_wait_started_at timestamptz,
    creator varchar(64) NOT NULL DEFAULT '',
    create_time timestamptz NOT NULL DEFAULT now(),
    updater varchar(64) NOT NULL DEFAULT '',
    update_time timestamptz NOT NULL DEFAULT now(),
    deleted boolean NOT NULL DEFAULT false,
    CONSTRAINT sys_knowledge_bases_scope_check CHECK (scope_type IN ('personal', 'org_node', 'position', 'role')),
    CONSTRAINT sys_knowledge_bases_status_check CHECK (status IN (0, 1))
);

CREATE UNIQUE INDEX sys_knowledge_bases_remote ON sys_knowledge_bases (weknora_kb_id) WHERE deleted = false;
CREATE UNIQUE INDEX sys_knowledge_bases_name ON sys_knowledge_bases (scope_type, scope_id, name) WHERE deleted = false;
CREATE INDEX sys_knowledge_bases_scope ON sys_knowledge_bases (scope_type, scope_id) WHERE deleted = false;

COMMENT ON TABLE sys_knowledge_bases IS '知识库。远端编号不对外暴露。Wiki 开关创建后不可修改。';
COMMENT ON COLUMN sys_knowledge_bases.scope_type IS '归属（personal 个人 org_node 组织 position 岗位 role 角色）';
COMMENT ON COLUMN sys_knowledge_bases.weknora_kb_id IS 'WeKnora 知识库编号';

CREATE TABLE sys_knowledge_base_tenants (
    id uuid PRIMARY KEY,
    knowledge_base_id uuid NOT NULL REFERENCES sys_knowledge_bases (id),
    tenant_id uuid NOT NULL REFERENCES sys_tenants (id),
    deleted boolean NOT NULL DEFAULT false
);

CREATE UNIQUE INDEX sys_knowledge_base_tenants_unique ON sys_knowledge_base_tenants (knowledge_base_id, tenant_id) WHERE deleted = false;

COMMENT ON TABLE sys_knowledge_base_tenants IS '知识库发布到的租户';

CREATE TABLE sys_knowledge_shares (
    id uuid PRIMARY KEY,
    knowledge_base_id uuid NOT NULL REFERENCES sys_knowledge_bases (id),
    user_id uuid NOT NULL REFERENCES sys_users (id),
    creator varchar(64) NOT NULL DEFAULT '',
    create_time timestamptz NOT NULL DEFAULT now(),
    deleted boolean NOT NULL DEFAULT false
);

CREATE UNIQUE INDEX sys_knowledge_shares_unique ON sys_knowledge_shares (knowledge_base_id, user_id) WHERE deleted = false;

COMMENT ON TABLE sys_knowledge_shares IS '个人知识库分享给其他用户';

CREATE TABLE sys_knowledge_files (
    id uuid PRIMARY KEY,
    knowledge_base_id uuid NOT NULL REFERENCES sys_knowledge_bases (id),
    weknora_knowledge_id varchar(64) NOT NULL DEFAULT '',
    file_name varchar(255) NOT NULL,
    folder_path varchar(500) NOT NULL DEFAULT '',
    file_size bigint NOT NULL DEFAULT 0,
    parse_status varchar(32) NOT NULL DEFAULT 'pending',
    error_msg varchar(500) NOT NULL DEFAULT '',
    creator_id uuid NOT NULL,
    creator varchar(64) NOT NULL DEFAULT '',
    create_time timestamptz NOT NULL DEFAULT now(),
    updater varchar(64) NOT NULL DEFAULT '',
    update_time timestamptz NOT NULL DEFAULT now(),
    deleted boolean NOT NULL DEFAULT false
);

CREATE INDEX sys_knowledge_files_base ON sys_knowledge_files (knowledge_base_id) WHERE deleted = false;

COMMENT ON TABLE sys_knowledge_files IS '知识库文件';
COMMENT ON COLUMN sys_knowledge_files.parse_status IS '解析状态（pending processing finalizing completed failed duplicate skipped）';

INSERT INTO sys_menus (
    id, name, permission, type, sort, parent_id, path, icon,
    component, component_name, status, visible, keep_alive, always_show
) VALUES
    (
        'a1000000-0000-4000-8000-00000000001e', '知识管理', 'knowledge:query', 2, 15, NULL, '/knowledge', '#',
        'views/KnowledgeView.vue', 'KnowledgeView', 0, true, true, true
    ),
    (
        'a1000000-0000-4000-8000-00000000021e', '查询', 'knowledge:query', 3, 1,
        'a1000000-0000-4000-8000-00000000001e', '', '#', NULL, NULL, 0, false, true, true
    ),
    (
        'a1000000-0000-4000-8000-00000000022e', '管理', 'knowledge:manage', 3, 2,
        'a1000000-0000-4000-8000-00000000001e', '', '#', NULL, NULL, 0, false, true, true
    );

INSERT INTO sys_role_menus (tenant_id, role_id, menu_id)
SELECT r.tenant_id, r.id, m.id
FROM sys_roles r
JOIN sys_menus m ON m.id IN (
    'a1000000-0000-4000-8000-00000000001e',
    'a1000000-0000-4000-8000-00000000021e'
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
SELECT r.tenant_id, r.id, 'a1000000-0000-4000-8000-00000000022e'
FROM sys_roles r
WHERE r.deleted = false
  AND r.code = 'super_admin'
  AND NOT EXISTS (
      SELECT 1 FROM sys_role_menus existing
      WHERE existing.tenant_id = r.tenant_id
        AND existing.role_id = r.id
        AND existing.menu_id = 'a1000000-0000-4000-8000-00000000022e'
        AND existing.deleted = false
  );
