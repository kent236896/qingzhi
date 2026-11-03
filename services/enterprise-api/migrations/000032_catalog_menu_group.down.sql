UPDATE sys_menus SET parent_id = NULL, sort = 15, update_time = now()
WHERE id = 'a1000000-0000-4000-8000-00000000001e' AND deleted = false;
UPDATE sys_menus SET parent_id = NULL, sort = 16, update_time = now()
WHERE id = 'a1000000-0000-4000-8000-000000000040' AND deleted = false;
UPDATE sys_menus SET parent_id = NULL, sort = 17, update_time = now()
WHERE id = 'a1000000-0000-4000-8000-000000000043' AND deleted = false;
UPDATE sys_menus SET parent_id = NULL, sort = 18, update_time = now()
WHERE id = 'a1000000-0000-4000-8000-000000000036' AND deleted = false;

DELETE FROM sys_role_menus WHERE menu_id = 'a1000000-0000-4000-8000-000000000046';
DELETE FROM sys_menus WHERE id = 'a1000000-0000-4000-8000-000000000046';
