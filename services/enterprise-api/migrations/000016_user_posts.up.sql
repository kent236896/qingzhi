CREATE TABLE sys_user_posts (
    id uuid PRIMARY KEY,
    tenant_id uuid NOT NULL REFERENCES sys_tenants (id),
    user_id uuid NOT NULL REFERENCES sys_users (id),
    post_id uuid NOT NULL REFERENCES sys_posts (id),
    creator varchar(64) NOT NULL DEFAULT '',
    create_time timestamptz NOT NULL DEFAULT now(),
    updater varchar(64) NOT NULL DEFAULT '',
    update_time timestamptz NOT NULL DEFAULT now(),
    deleted boolean NOT NULL DEFAULT false
);

CREATE UNIQUE INDEX sys_user_posts_unique ON sys_user_posts (tenant_id, user_id, post_id) WHERE deleted = false;
CREATE INDEX sys_user_posts_user ON sys_user_posts (tenant_id, user_id) WHERE deleted = false;

COMMENT ON TABLE sys_user_posts IS '用户岗位任职';
COMMENT ON COLUMN sys_user_posts.user_id IS '用户ID';
COMMENT ON COLUMN sys_user_posts.post_id IS '岗位ID';
COMMENT ON COLUMN sys_user_posts.tenant_id IS '租户编号';
