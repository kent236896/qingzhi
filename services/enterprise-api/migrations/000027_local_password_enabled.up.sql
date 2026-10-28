ALTER TABLE sys_users
  ADD COLUMN IF NOT EXISTS local_password_enabled boolean NOT NULL DEFAULT true;

-- Federated auto-provisioned accounts keep a random hash for schema compatibility
-- but must not be treated as locally managed password credentials.
UPDATE sys_users
SET local_password_enabled = false
WHERE deleted = false
  AND remark = 'federated';
