import { http } from '@/api/client'

export interface ServiceState {
  api: string
  database: string
  redis: string
}

export interface Traffic {
  requests: number
  server_errors: number
  client_errors: number
  timeouts: number
  rejected: number
  avg_ms: number
  p95_ms: number
}

export interface ErrorStat {
  type: string
  count: number
  last_time: string
}

export interface RecentError {
  time: string
  method: string
  path: string
  status: number
  duration_ms: number
  type: string
  request_id: string
}

export interface Resources {
  available: boolean
  cpu_percent: number
  memory_used: number
  memory_total: number
  memory_percent: number
  disk_used: number
  disk_total: number
  disk_percent: number
  disk_mount: string
}

export interface Protection {
  rate_limit_rps: number
  rate_limit_burst: number
  max_body_bytes: number
  request_timeout_ms: number
}

export interface MonitorAlert {
  id: string
  kind: string
  target: string
  level: string
  title: string
  message: string
  status: string
  first_time: string
  last_time: string
  resolved_time?: string | null
}

export interface MonitorOverview {
  collected_at: string
  window_seconds: number
  service: ServiceState
  traffic: Traffic
  errors: ErrorStat[]
  recent_errors: RecentError[]
  resources: Resources
  protection: Protection
  alerts: MonitorAlert[]
}

export interface AlertPage {
  items: MonitorAlert[]
  page: number
  page_size: number
  total: number
}

export function getMonitorOverview() {
  return http.get<MonitorOverview>('/api/v1/monitor/overview')
}

export function listMonitorAlerts(params: { page?: number; page_size?: number; status?: string } = {}) {
  return http.get<AlertPage>('/api/v1/monitor/alerts', { params })
}

export function formatBytes(value: number) {
  if (!value) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let size = value
  let index = 0
  while (size >= 1024 && index < units.length - 1) {
    size /= 1024
    index += 1
  }
  const digits = index === 0 ? 0 : 1
  return `${size.toFixed(digits)} ${units[index]}`
}

export function errorTypeLabel(type: string) {
  if (type === 'timeout') return '超时'
  return `HTTP ${type}`
}

export function alertStatusLabel(status: string) {
  if (status === 'firing') return '未恢复'
  if (status === 'resolved') return '已恢复'
  return status || '未知'
}
