CREATE TABLE sys_posts (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL REFERENCES sys_tenants (id),
    code varchar(64) NOT NULL,
    name varchar(50) NOT NULL,
    sort integer NOT NULL DEFAULT 0,
    status smallint NOT NULL DEFAULT 0,
    remark varchar(500),
    creator varchar(64) NOT NULL DEFAULT '',
    create_time timestamptz NOT NULL DEFAULT now(),
    updater varchar(64) NOT NULL DEFAULT '',
    update_time timestamptz NOT NULL DEFAULT now(),
    deleted boolean NOT NULL DEFAULT false,
    CONSTRAINT sys_posts_status_check CHECK (status IN (0, 1))
);

CREATE UNIQUE INDEX sys_posts_code_unique ON sys_posts (tenant_id, code) WHERE deleted = false;
CREATE INDEX sys_posts_tenant ON sys_posts (tenant_id, sort, name);

COMMENT ON TABLE sys_posts IS '岗位';
COMMENT ON COLUMN sys_posts.code IS '岗位编码';
COMMENT ON COLUMN sys_posts.name IS '岗位名称';
COMMENT ON COLUMN sys_posts.sort IS '显示顺序';
COMMENT ON COLUMN sys_posts.status IS '状态（0正常 1停用）';
COMMENT ON COLUMN sys_posts.remark IS '备注';
COMMENT ON COLUMN sys_posts.creator IS '创建者';
COMMENT ON COLUMN sys_posts.create_time IS '创建时间';
COMMENT ON COLUMN sys_posts.updater IS '更新者';
COMMENT ON COLUMN sys_posts.update_time IS '更新时间';
COMMENT ON COLUMN sys_posts.deleted IS '是否删除';
COMMENT ON COLUMN sys_posts.tenant_id IS '租户编号';
