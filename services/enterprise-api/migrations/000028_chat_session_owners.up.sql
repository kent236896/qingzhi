-- Platform-side ownership for WeKnora chat sessions (data isolation).
CREATE TABLE IF NOT EXISTS chat_session_owners (
    weknora_session_id varchar(64) PRIMARY KEY,
    user_id varchar(64) NOT NULL,
    tenant_id varchar(64) NOT NULL DEFAULT '',
    weknora_tenant_id varchar(64) NOT NULL DEFAULT '',
    title varchar(255) NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_chat_session_owners_user_updated
    ON chat_session_owners (user_id, updated_at DESC);

COMMENT ON TABLE chat_session_owners IS 'Maps WeKnora session ids to platform users; list/access must use this table, not admin-wide WeKnora lists';
