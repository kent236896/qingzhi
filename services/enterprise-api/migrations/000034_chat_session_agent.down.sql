DROP INDEX IF EXISTS idx_chat_session_owners_agent;
ALTER TABLE chat_session_owners DROP COLUMN IF EXISTS agent_id;
