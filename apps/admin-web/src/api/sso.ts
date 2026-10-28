import { http } from '@/api/client'
import type { ListParams, PageResult } from '@/api/identity'

export interface ProviderCount {
  total: number
  enabled: number
}


export interface AuthEvent {
  id: string
  event_type: string
  result: string
  protocol: string
  provider_id: string
  client_id: string
  user_id: string
  username: string
  message: string
  request_id: string
  ip: string
  create_time: string
}

export interface CertWarning {
  kind: string
  ref_id: string
  name: string
  not_after?: string
  days_left?: number
  expired: boolean
}

export interface SsoOverview {
  local_login_enabled: boolean
  oidc_providers: ProviderCount
  saml_providers: ProviderCount
  identity_links: number
  recent_login_success: number
  recent_login_failure: number
  recent_protocol_failures: AuthEvent[]
  certificate_warnings: CertWarning[]
  collected_at: string
}

export interface SsoSettings {
  local_login_enabled: boolean
  default_tenant_id: string
  frontend_base_url: string
  updater: string
  updated_at: string
}

export interface SsoProvider {
  id: string
  name: string
  protocol: string
  status: string
  priority: number
  issuer: string
  display_on_login: boolean
  auto_login: boolean
  auto_create_user: boolean
  discovery_url?: string
  client_id?: string
  client_secret_set: boolean
  client_secret_hint?: string
  redirect_uri?: string
  scopes?: string
  entity_id?: string
  metadata_url?: string
  sso_url?: string
  slo_url?: string
  idp_certificate_pem?: string
  sp_entity_id?: string
  acs_url?: string
  name_id_format?: string
  last_success_at?: string
  last_error?: string
  created_at: string
  updated_at: string
}

export interface IdentityLink {
  id: string
  provider_id: string
  provider_name: string
  protocol: string
  external_subject: string
  user_id: string
  created_at: string
  updated_at: string
}

export function getSsoOverview() {
  return http.get<SsoOverview>('/api/v1/sso/overview')
}

export function getSsoSettings() {
  return http.get<SsoSettings>('/api/v1/sso/settings')
}

export function updateSsoSettings(body: Partial<SsoSettings>) {
  return http.put<SsoSettings>('/api/v1/sso/settings', body)
}

export function listSsoProviders(params: ListParams & { protocol?: string; status?: string } = {}) {
  return http.get<PageResult<SsoProvider>>('/api/v1/sso/providers', { params })
}

export function getSsoProvider(id: string) {
  return http.get<SsoProvider>(`/api/v1/sso/providers/${id}`)
}

export function createSsoProvider(body: Record<string, unknown>) {
  return http.post<SsoProvider>('/api/v1/sso/providers', body)
}

export function updateSsoProvider(id: string, body: Record<string, unknown>) {
  return http.put<SsoProvider>(`/api/v1/sso/providers/${id}`, body)
}

export function setSsoProviderStatus(id: string, status: string) {
  return http.patch(`/api/v1/sso/providers/${id}/status`, { status })
}

export function listIdentityLinks(params: ListParams = {}) {
  return http.get<PageResult<IdentityLink>>('/api/v1/sso/identity-links', { params })
}

export function deleteIdentityLink(id: string) {
  return http.delete(`/api/v1/sso/identity-links/${id}`)
}

export function listAuthEvents(params: ListParams & { protocol?: string; result?: string } = {}) {
  return http.get<PageResult<AuthEvent>>('/api/v1/sso/auth-events', { params })
}

export interface LoginOption {
  id: string
  name: string
  protocol: string
  display_on_login: boolean
  auto_login: boolean
}

export interface LoginOptionsResult {
  local_login_enabled: boolean
  providers: LoginOption[]
}

export function getLoginOptions() {
  return http.get<LoginOptionsResult>('/api/v1/sso/login-options', { skipErrorToast: true })
}

export function exchangeSsoTicket(ticket: string) {
  return http.post<import('@/api/auth').LoginResult>('/api/v1/sso/oidc/exchange', { ticket }, { skipErrorToast: true })
}

export function browserFrontendBase() {
  if (typeof window === 'undefined') {
    return ''
  }
  return window.location.origin
}

export function oidcStartURL(providerId: string, redirect = '/', tenantId?: string) {
  return externalStartURL('oidc', providerId, redirect, tenantId)
}

export function samlStartURL(providerId: string, redirect = '/', tenantId?: string) {
  return externalStartURL('saml', providerId, redirect, tenantId)
}

function externalStartURL(kind: 'oidc' | 'saml', providerId: string, redirect = '/', tenantId?: string) {
  const base = (import.meta.env.VITE_API_BASE_URL || '').replace(/\/$/, '')
  const query = new URLSearchParams({
    provider_id: providerId,
    redirect,
    frontend_base: browserFrontendBase(),
  })
  if (tenantId) {
    query.set('tenant_id', tenantId)
  }
  const path = kind === 'saml' ? '/api/v1/sso/saml/start' : '/api/v1/sso/oidc/start'
  return `${base}${path}?${query.toString()}`
}

export function protocolLabel(value: string) {
  if (value === 'oidc') return 'OIDC'
  if (value === 'saml') return 'SAML 2.0'
  return value || '—'
}

export function statusLabel(value: string) {
  if (value === 'enabled') return '启用'
  if (value === 'disabled') return '停用'
  if (value === 'revoked') return '已吊销'
  return value || '—'
}

export function resultLabel(value: string) {
  if (value === 'success') return '成功'
  if (value === 'failure') return '失败'
  return value || '—'
}
