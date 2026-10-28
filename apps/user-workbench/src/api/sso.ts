import { http } from '@/api/client'
import type { LoginResult } from '@/types/auth'

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
  return http.post<LoginResult>('/api/v1/sso/oidc/exchange', { ticket }, { skipErrorToast: true })
}

export function browserFrontendBase() {
  if (typeof window === 'undefined') {
    return ''
  }
  return window.location.origin
}

export function oidcStartURL(providerId: string, redirect = '/chat', tenantId?: string) {
  return externalStartURL('oidc', providerId, redirect, tenantId)
}

export function samlStartURL(providerId: string, redirect = '/chat', tenantId?: string) {
  return externalStartURL('saml', providerId, redirect, tenantId)
}

function externalStartURL(kind: 'oidc' | 'saml', providerId: string, redirect = '/chat', tenantId?: string) {
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
