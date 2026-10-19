CREATE TABLE sys_weknora_model_bindings (
    role varchar(32) PRIMARY KEY,
    model_id varchar(64) NOT NULL,
    updated_by varchar(64) NOT NULL DEFAULT '',
    updated_at timestamptz NOT NULL DEFAULT now()
);

COMMENT ON TABLE sys_weknora_model_bindings IS '场景默认模型绑定，只保存模型 ID';
COMMENT ON COLUMN sys_weknora_model_bindings.role IS 'embedding summary wiki_synthesis vlm asr rerank';

INSERT INTO sys_menus (
    id, name, permission, type, sort, parent_id, path, icon,
    component, component_name, status, visible, keep_alive, always_show
) VALUES
    (
        'a1000000-0000-4000-8000-00000000001f', '模型配置', 'model:query', 2, 16, NULL, '/models', '#',
        'views/ModelsView.vue', 'ModelsView', 0, true, true, true
    ),
    (
        'a1000000-0000-4000-8000-00000000021f', '查看', 'model:query', 3, 1,
        'a1000000-0000-4000-8000-00000000001f', '', '#', NULL, NULL, 0, false, true, true
    ),
    (
        'a1000000-0000-4000-8000-00000000022f', '管理', 'model:manage', 3, 2,
        'a1000000-0000-4000-8000-00000000001f', '', '#', NULL, NULL, 0, false, true, true
    );

INSERT INTO sys_role_menus (tenant_id, role_id, menu_id)
SELECT r.tenant_id, r.id, m.id
FROM sys_roles r
JOIN sys_menus m ON m.id IN (
    'a1000000-0000-4000-8000-00000000001f',
    'a1000000-0000-4000-8000-00000000021f',
    'a1000000-0000-4000-8000-00000000022f'
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
