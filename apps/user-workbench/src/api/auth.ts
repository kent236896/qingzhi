import { http } from '@/api/client'
import type { LoginResult, AuthMe, OwnProfile, OwnProfilePayload, PasswordPolicy } from '@/types/auth'

export type { LoginResult, AuthMe, OwnProfile, OwnProfilePayload, PasswordPolicy }

export function login(body: { username: string; password: string; tenant_id?: string }) {
  return http.post<LoginResult>('/api/v1/auth/login', body, { skipErrorToast: true })
}

export function fetchAuthMe() {
  return http.get<AuthMe>('/api/v1/auth/me')
}

export function fetchOwnProfile() {
  return http.get<OwnProfilePayload>('/api/v1/auth/profile')
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
  return http.put<{ changed: boolean }>('/api/v1/auth/password', body, { skipErrorToast: true })
}

export function fetchPasswordPolicy() {
  return http.get<PasswordPolicy>('/api/v1/auth/password-policy')
}
