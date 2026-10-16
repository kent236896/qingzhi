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
  version: string
  user: {
    id: string
    username: string
    nickname: string
    avatar?: string
  }
  tenant_id: string
  permissions: string[]
  data_scope: number
}

export interface OwnProfile {
  id: string
  username: string
  nickname: string
  remark: string
  email: string
  mobile: string
  sex: number
  avatar: string
  status: number
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

export function fetchOwnProfile() {
  return http.get<{ version: string; user: OwnProfile }>('/api/v1/auth/profile')
}

export function updateOwnProfile(body: {
  nickname?: string
  remark?: string
  email?: string
  mobile?: string
  sex?: number
  avatar?: string
}) {
  return http.patch<OwnProfile>('/api/v1/auth/profile', body)
}

export function changeOwnPassword(body: { current_password: string; new_password: string }) {
  return http.put<{ changed: boolean }>('/api/v1/auth/password', body)
}
