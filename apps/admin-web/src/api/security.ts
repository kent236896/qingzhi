import { http } from '@/api/client'
import type { ListParams, PageResult } from '@/api/identity'

export interface LoginLog {
  id: string
  username: string
  user_id: string
  tenant_id: string
  result: number
  message: string
  ip: string
  user_agent: string
  create_time: string
}

export interface OperateLog {
  id: string
  user_id: string
  username: string
  tenant_id: string
  module: string
  action: string
  request_method: string
  request_url: string
  ip: string
  user_agent: string
  status_code: number
  duration_ms: number
  request_id: string
  create_time: string
}

export interface PermissionLog {
  id: string
  tenant_id: string
  operator_id: string
  operator_name: string
  role_id: string
  role_name: string
  change_type: string
  summary: string
  before_text: string
  after_text: string
  create_time: string
}

export interface SecurityPolicy {
  password_min_length: number
  password_max_length: number
  password_require_letter: boolean
  password_require_number: boolean
  password_require_symbol: boolean
  login_fail_limit: number
  login_lock_minutes: number
  updater: string
  update_time: string
}

export function formatTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value || '—'
  }
  return date.toLocaleString('zh-CN', { hour12: false })
}

export function loginResultLabel(result: number) {
  if (result === 0) return '成功'
  if (result === 1) return '失败'
  if (result === 2) return '锁定'
  return '未知'
}

export function changeTypeLabel(value: string) {
  if (value === 'menus') return '菜单权限'
  if (value === 'users') return '角色用户'
  if (value === 'depts') return '数据部门'
  if (value === 'data_scope') return '数据范围'
  return value || '未知'
}

export function listLoginLogs(params: ListParams & { result?: number } = {}) {
  return http.get<PageResult<LoginLog>>('/api/v1/security/login-logs', { params })
}

export function listOperateLogs(params: ListParams = {}) {
  return http.get<PageResult<OperateLog>>('/api/v1/security/operate-logs', { params })
}

export function listPermissionLogs(params: ListParams = {}) {
  return http.get<PageResult<PermissionLog>>('/api/v1/security/permission-logs', { params })
}

export function getSecurityPolicy() {
  return http.get<SecurityPolicy>('/api/v1/security/policy')
}

export function updateSecurityPolicy(body: Omit<SecurityPolicy, 'updater' | 'update_time'>) {
  return http.patch<SecurityPolicy>('/api/v1/security/policy', body)
}
