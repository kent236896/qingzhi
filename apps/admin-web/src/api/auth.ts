import { http } from '@/api/client'
import type { MenuNode } from '@/api/identity'

export interface LoginResult {
  access_token: string
  token_type: string
  expires_at: string
  user: {
    id: string
    username: string
    nickname: string
    avatar: string
  }
  tenant_id: string
  permissions: string[]
  data_scope: number
}

export interface AuthMe {
  user: {
    id: string
    username: string
    nickname: string
  }
  tenant_id: string
  permissions: string[]
  data_scope: number
}

export function login(body: { username: string; password: string; tenant_id?: string }) {
  return http.post<LoginResult>('/api/v1/auth/login', body, { skipErrorToast: true })
}

export function fetchAuthMe() {
  return http.get<AuthMe>('/api/v1/auth/me')
}

export function fetchAuthMenus() {
  return http.get<MenuNode[]>('/api/v1/auth/menus')
}
