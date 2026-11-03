-- Bind platform catalog agent to a chat session (ACL still via chat_session_owners + scoped agents).
ALTER TABLE chat_session_owners
    ADD COLUMN IF NOT EXISTS agent_id uuid NULL;

CREATE INDEX IF NOT EXISTS idx_chat_session_owners_agent
    ON chat_session_owners (agent_id)
    WHERE agent_id IS NOT NULL;

COMMENT ON COLUMN chat_session_owners.agent_id IS '平台 sys_agents.id；空则走 knowledge-chat 且 agent_enabled=false';
