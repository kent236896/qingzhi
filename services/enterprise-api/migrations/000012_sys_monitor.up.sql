CREATE TABLE sys_monitor_alerts (
    id uuid PRIMARY KEY,
    kind varchar(32) NOT NULL,
    target varchar(64) NOT NULL,
    level varchar(16) NOT NULL,
    title varchar(100) NOT NULL,
    message varchar(500) NOT NULL DEFAULT '',
    status varchar(16) NOT NULL,
    first_time timestamptz NOT NULL DEFAULT now(),
    last_time timestamptz NOT NULL DEFAULT now(),
    resolved_time timestamptz,
    CONSTRAINT sys_monitor_alerts_status_check CHECK (status IN ('firing', 'resolved')),
    CONSTRAINT sys_monitor_alerts_level_check CHECK (level IN ('warning', 'critical'))
);

CREATE UNIQUE INDEX sys_monitor_alerts_open ON sys_monitor_alerts (kind, target) WHERE status = 'firing';
CREATE INDEX sys_monitor_alerts_time ON sys_monitor_alerts (last_time DESC);

COMMENT ON TABLE sys_monitor_alerts IS '监控告警';
COMMENT ON COLUMN sys_monitor_alerts.kind IS '类别（service 服务不可用 traffic 请求异常 capacity 容量不足 task 任务失败）';
COMMENT ON COLUMN sys_monitor_alerts.status IS '状态（firing 未恢复 resolved 已恢复）';

INSERT INTO sys_menus (
    id, name, permission, type, sort, parent_id, path, icon,
    component, component_name, status, visible, keep_alive, always_show
) VALUES
    (
        'a1000000-0000-4000-8000-000000000004', '系统监控', '', 1, 40, NULL, '', '#',
        NULL, NULL, 0, true, true, true
    ),
    (
        'a1000000-0000-4000-8000-00000000001d', '监控概览', 'system:monitor:query', 2, 10,
        'a1000000-0000-4000-8000-000000000004', '/monitor', '#',
        'views/MonitorView.vue', 'MonitorView', 0, true, true, true
    ),
    (
        'a1000000-0000-4000-8000-00000000021d', '查询', 'system:monitor:query', 3, 1,
        'a1000000-0000-4000-8000-00000000001d', '', '#', NULL, NULL, 0, false, true, true
    );

INSERT INTO sys_role_menus (tenant_id, role_id, menu_id)
SELECT rm.tenant_id, rm.role_id, m.id
FROM sys_role_menus rm
JOIN sys_menus m ON m.id IN (
    'a1000000-0000-4000-8000-000000000004',
    'a1000000-0000-4000-8000-00000000001d',
    'a1000000-0000-4000-8000-00000000021d'
)
WHERE rm.menu_id = 'a1000000-0000-4000-8000-000000000001'
  AND rm.deleted = false
  AND NOT EXISTS (
      SELECT 1 FROM sys_role_menus existing
      WHERE existing.tenant_id = rm.tenant_id
        AND existing.role_id = rm.role_id
        AND existing.menu_id = m.id
        AND existing.deleted = false
  );
