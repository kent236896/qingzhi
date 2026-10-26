CREATE TABLE sys_knowledge_share_scopes (
    id uuid PRIMARY KEY,
    knowledge_base_id uuid NOT NULL REFERENCES sys_knowledge_bases (id),
    target_type varchar(16) NOT NULL,
    target_id uuid NOT NULL,
    creator varchar(64) NOT NULL DEFAULT '',
    create_time timestamptz NOT NULL DEFAULT now(),
    deleted boolean NOT NULL DEFAULT false,
    CONSTRAINT sys_knowledge_share_scopes_type_check CHECK (target_type IN ('org_node', 'role'))
);

CREATE UNIQUE INDEX sys_knowledge_share_scopes_unique
    ON sys_knowledge_share_scopes (knowledge_base_id, target_type, target_id)
    WHERE deleted = false;

CREATE INDEX sys_knowledge_share_scopes_base
    ON sys_knowledge_share_scopes (knowledge_base_id)
    WHERE deleted = false;

COMMENT ON TABLE sys_knowledge_share_scopes IS '组织/角色知识库分享给其他组织或角色';
COMMENT ON COLUMN sys_knowledge_share_scopes.target_type IS '分享目标类型（org_node 组织 role 角色）';
