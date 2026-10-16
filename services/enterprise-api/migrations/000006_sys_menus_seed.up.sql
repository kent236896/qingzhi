-- 默认菜单。「首页」是一级菜单，其余业务页面挂在「系统管理」下。
INSERT INTO sys_menus (
    id, name, permission, type, sort, parent_id, path, icon,
    component, component_name, status, visible, keep_alive, always_show
) VALUES
    (
        'a1000000-0000-4000-8000-000000000001', '系统管理', '', 1, 10, NULL, '', '#',
        NULL, NULL, 0, true, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000002', '基础设施', '', 1, 20, NULL, '', '#',
        NULL, NULL, 0, true, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000011', '首页', 'system:home', 2, 0,
        NULL, '/', '#',
        'views/HomeView.vue', 'HomeView', 0, true, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000012', '租户管理', 'system:tenant:query', 2, 20,
        'a1000000-0000-4000-8000-000000000001', '/tenants', '#',
        'views/TenantsView.vue', 'TenantsView', 0, true, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000013', '组织管理', 'system:organization:query', 2, 30,
        'a1000000-0000-4000-8000-000000000001', '/organizations', '#',
        'views/OrganizationsView.vue', 'OrganizationsView', 0, true, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000014', '岗位管理', 'system:post:query', 2, 40,
        'a1000000-0000-4000-8000-000000000001', '/posts', '#',
        'views/PostsView.vue', 'PostsView', 0, true, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000015', '菜单管理', 'system:menu:query', 2, 50,
        'a1000000-0000-4000-8000-000000000001', '/menus', '#',
        'views/MenusView.vue', 'MenusView', 0, true, true, true
    ),
    (
        'a1000000-0000-4000-8000-000000000016', '用户管理', 'system:user:query', 2, 60,
        'a1000000-0000-4000-8000-000000000001', '/users', '#',
        'views/UsersView.vue', 'UsersView', 0, true, true, true
    );
