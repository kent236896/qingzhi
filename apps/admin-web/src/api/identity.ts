import { http } from '@/api/client'

export type RecordStatus = 0 | 1

export interface PageResult<T> {
  items: T[]
  page: number
  page_size: number
  total: number
}

export interface Tenant {
  id: string
  name: string
  code: string
  contact_user_id: string
  contact_name: string
  contact_mobile: string
  status: RecordStatus
  account_count: number
  creator: string
  create_time: string
  updater: string
  update_time: string
  deleted: boolean
}

export interface OrganizationNode {
  id: string
  tenant_id: string
  parent_id: string
  is_tenant_root: boolean
  name: string
  code: string
  sort: number
  leader_user_id: string
  phone: string
  email: string
  status: RecordStatus
  creator: string
  create_time: string
  updater: string
  update_time: string
  deleted: boolean
  children: OrganizationNode[]
}

export interface UserAccount {
  id: string
  username: string
  nickname: string
  remark: string
  email: string
  mobile: string
  sex: number
  avatar: string
  status: RecordStatus
  login_ip: string
  login_date: string | null
  creator: string
  create_time: string
  updater: string
  update_time: string
  deleted: boolean
}

export interface Assignment {
  id: string
  name: string
  is_primary: boolean
}

export interface UserMembership {
  tenant_id: string
  tenant_name: string
  tenant_code: string
  status: RecordStatus
  root_organization_id: string
  root_organization_name: string
  organizations: Assignment[]
}

export function sexLabel(sex: number): string {
  if (sex === 1) {
    return '男'
  }
  if (sex === 2) {
    return '女'
  }
  return '未知'
}

export interface UserDetail extends UserAccount {
  memberships: UserMembership[]
}

export interface Member {
  user_id: string
  username: string
  nickname: string
  status: RecordStatus
  primary_organization_id: string
  organizations: Assignment[]
}

export interface ListParams {
  page?: number
  page_size?: number
  keyword?: string
}

export function statusLabel(status: number): string {
  return status === 0 ? '正常' : '停用'
}

function query(params: ListParams = {}) {
  return { params }
}

export function listTenants(params: ListParams = {}) {
  return http.get<PageResult<Tenant>>('/api/v1/tenants', query(params))
}

export function createTenant(body: {
  name: string
  code: string
  contact_name: string
  contact_mobile?: string
  contact_user_id?: string
  account_count?: number
}) {
  return http.post<Tenant>('/api/v1/tenants', body)
}

export function updateTenant(
  id: string,
  body: {
    name?: string
    status?: RecordStatus
    contact_name?: string
    contact_mobile?: string
    contact_user_id?: string
    account_count?: number
  },
) {
  return http.patch<Tenant>(`/api/v1/tenants/${id}`, body)
}

export function getTenant(id: string) {
  return http.get<Tenant>(`/api/v1/tenants/${id}`)
}

export function listOrganizations(tenantId: string) {
  return http.get<OrganizationNode[]>(`/api/v1/tenants/${tenantId}/organizations`)
}

export function createOrganization(
  tenantId: string,
  body: {
    name: string
    code: string
    parent_id?: string
    sort?: number
    leader_user_id?: string
    phone?: string
    email?: string
  },
) {
  return http.post<OrganizationNode>(`/api/v1/tenants/${tenantId}/organizations`, body)
}

export function updateOrganization(
  tenantId: string,
  organizationId: string,
  body: {
    name?: string
    code?: string
    parent_id?: string | null
    sort?: number
    leader_user_id?: string
    phone?: string
    email?: string
    status?: RecordStatus
  },
) {
  return http.patch<OrganizationNode>(`/api/v1/tenants/${tenantId}/organizations/${organizationId}`, body)
}

export function deleteOrganization(tenantId: string, organizationId: string) {
  return http.delete<null>(`/api/v1/tenants/${tenantId}/organizations/${organizationId}`)
}

export interface Post {
  id: string
  tenant_id: string
  code: string
  name: string
  sort: number
  status: RecordStatus
  remark: string
  creator: string
  create_time: string
  updater: string
  update_time: string
  deleted: boolean
}

export function listUserPosts(tenantId: string, userId: string) {
  return http.get<Post[]>(`/api/v1/tenants/${tenantId}/members/${userId}/posts`)
}

export function replaceUserPosts(tenantId: string, userId: string, postIds: string[]) {
  return http.put<Post[]>(`/api/v1/tenants/${tenantId}/members/${userId}/posts`, { post_ids: postIds })
}

export function listPosts(tenantId: string, params: ListParams = {}) {
  return http.get<PageResult<Post>>(`/api/v1/tenants/${tenantId}/posts`, query(params))
}

export function createPost(tenantId: string, body: { code: string; name: string; sort?: number; remark?: string }) {
  return http.post<Post>(`/api/v1/tenants/${tenantId}/posts`, body)
}

export function updatePost(
  tenantId: string,
  postId: string,
  body: { code?: string; name?: string; sort?: number; status?: RecordStatus; remark?: string },
) {
  return http.patch<Post>(`/api/v1/tenants/${tenantId}/posts/${postId}`, body)
}

export function deletePost(tenantId: string, postId: string) {
  return http.delete<null>(`/api/v1/tenants/${tenantId}/posts/${postId}`)
}

export type RoleType = 1 | 2

export type DataScope = 1 | 2 | 3 | 4 | 5

export interface Role {
  id: string
  tenant_id: string
  name: string
  code: string
  sort: number
  status: RecordStatus
  type: RoleType
  data_scope: DataScope
  remark: string
  creator: string
  create_time: string
  updater: string
  update_time: string
  deleted: boolean
}

export interface RoleUser {
  user_id: string
  username: string
  nickname: string
}

export function roleTypeLabel(type: number): string {
  if (type === 1) {
    return '系统'
  }
  if (type === 2) {
    return '自定义'
  }
  return '未知'
}

export function dataScopeLabel(scope: number): string {
  switch (scope) {
    case 1:
      return '全部数据'
    case 2:
      return '自定义部门'
    case 3:
      return '本部门'
    case 4:
      return '本部门及以下'
    case 5:
      return '仅本人'
    default:
      return '未知'
  }
}

export function listRoles(tenantId: string, params: ListParams = {}) {
  return http.get<PageResult<Role>>(`/api/v1/tenants/${tenantId}/roles`, query(params))
}

export function createRole(
  tenantId: string,
  body: { name: string; code: string; sort?: number; type?: RoleType; data_scope?: DataScope; remark?: string },
) {
  return http.post<Role>(`/api/v1/tenants/${tenantId}/roles`, body)
}

export function updateRole(
  tenantId: string,
  roleId: string,
  body: {
    name?: string
    code?: string
    sort?: number
    status?: RecordStatus
    type?: RoleType
    data_scope?: DataScope
    remark?: string
  },
) {
  return http.patch<Role>(`/api/v1/tenants/${tenantId}/roles/${roleId}`, body)
}

export function deleteRole(tenantId: string, roleId: string) {
  return http.delete<null>(`/api/v1/tenants/${tenantId}/roles/${roleId}`)
}

export function listRoleMenus(tenantId: string, roleId: string) {
  return http.get<string[]>(`/api/v1/tenants/${tenantId}/roles/${roleId}/menus`)
}

export function replaceRoleMenus(tenantId: string, roleId: string, menuIds: string[]) {
  return http.put<string[]>(`/api/v1/tenants/${tenantId}/roles/${roleId}/menus`, { menu_ids: menuIds })
}

export function listRoleDepts(tenantId: string, roleId: string) {
  return http.get<string[]>(`/api/v1/tenants/${tenantId}/roles/${roleId}/depts`)
}

export function replaceRoleDepts(tenantId: string, roleId: string, deptIds: string[]) {
  return http.put<string[]>(`/api/v1/tenants/${tenantId}/roles/${roleId}/depts`, { dept_ids: deptIds })
}

export function listRoleUsers(tenantId: string, roleId: string) {
  return http.get<RoleUser[]>(`/api/v1/tenants/${tenantId}/roles/${roleId}/users`)
}

export function replaceRoleUsers(tenantId: string, roleId: string, userIds: string[]) {
  return http.put<RoleUser[]>(`/api/v1/tenants/${tenantId}/roles/${roleId}/users`, { user_ids: userIds })
}

export function listMembers(tenantId: string, params: ListParams = {}) {
  return http.get<PageResult<Member>>(`/api/v1/tenants/${tenantId}/members`, query(params))
}

export type MenuType = 1 | 2 | 3

export interface MenuNode {
  id: string
  name: string
  permission: string
  type: MenuType
  sort: number
  parent_id: string
  path: string
  icon: string
  component: string
  component_name: string
  status: RecordStatus
  visible: boolean
  keep_alive: boolean
  always_show: boolean
  creator: string
  create_time: string
  updater: string
  update_time: string
  deleted: boolean
  children: MenuNode[]
}

export function menuTypeLabel(type: number): string {
  if (type === 1) {
    return '目录'
  }
  if (type === 2) {
    return '菜单'
  }
  if (type === 3) {
    return '按钮'
  }
  return '未知'
}

export function listMenus() {
  return http.get<MenuNode[]>('/api/v1/menus')
}

export function createMenu(body: {
  name: string
  type: MenuType
  permission?: string
  sort?: number
  parent_id?: string
  path?: string
  icon?: string
  component?: string
  component_name?: string
  visible?: boolean
  keep_alive?: boolean
  always_show?: boolean
}) {
  return http.post<MenuNode>('/api/v1/menus', body)
}

export function updateMenu(
  id: string,
  body: {
    name?: string
    type?: MenuType
    permission?: string
    sort?: number
    parent_id?: string | null
    path?: string
    icon?: string
    component?: string
    component_name?: string
    status?: RecordStatus
    visible?: boolean
    keep_alive?: boolean
    always_show?: boolean
  },
) {
  return http.patch<MenuNode>(`/api/v1/menus/${id}`, body)
}

export function deleteMenu(id: string) {
  return http.delete<null>(`/api/v1/menus/${id}`)
}

export function flattenMenus(
  nodes: MenuNode[],
  prefix = '',
): Array<{ id: string; label: string; type: MenuType }> {
  const result: Array<{ id: string; label: string; type: MenuType }> = []
  for (const node of nodes) {
    const label = prefix ? `${prefix} / ${node.name}` : node.name
    result.push({ id: node.id, label, type: node.type })
    result.push(...flattenMenus(node.children ?? [], label))
  }
  return result
}

export function listUsers(params: ListParams = {}) {
  return http.get<PageResult<UserAccount>>('/api/v1/users', query(params))
}

export function createUser(body: {
  username: string
  password: string
  nickname: string
  remark?: string
  email?: string
  mobile?: string
  sex?: number
  avatar?: string
}) {
  return http.post<UserAccount>('/api/v1/users', body)
}

export function getUser(id: string) {
  return http.get<UserDetail>(`/api/v1/users/${id}`)
}

export function updateUser(
  id: string,
  body: {
    nickname?: string
    password?: string
    remark?: string
    email?: string
    mobile?: string
    sex?: number
    avatar?: string
    status?: RecordStatus
  },
) {
  return http.patch<UserAccount>(`/api/v1/users/${id}`, body)
}

export function createMember(
  tenantId: string,
  body: {
    user_id: string
    status?: RecordStatus
    organization_ids: string[]
    primary_organization_id?: string
  },
) {
  return http.post<Member>(`/api/v1/tenants/${tenantId}/members`, body)
}

export function replaceMemberAssignments(
  tenantId: string,
  userId: string,
  body: { organization_ids: string[]; primary_organization_id?: string },
) {
  return http.put<Member>(`/api/v1/tenants/${tenantId}/members/${userId}`, body)
}

export function removeMember(tenantId: string, userId: string) {
  return http.delete<null>(`/api/v1/tenants/${tenantId}/members/${userId}`)
}

export interface FlatOrganization {
  id: string
  label: string
}

export function flattenOrganizations(nodes: OrganizationNode[], prefix = ''): FlatOrganization[] {
  const result: FlatOrganization[] = []
  for (const node of nodes) {
    const label = prefix ? `${prefix} / ${node.name}` : node.name
    result.push({ id: node.id, label })
    result.push(...flattenOrganizations(node.children ?? [], label))
  }
  return result
}
