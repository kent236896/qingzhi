export type AuthMethod = 'local' | 'sso'

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
  auth_method: AuthMethod
  local_password_enabled: boolean
  username_editable: boolean
  account_console_url: string
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
  local_password_enabled: boolean
}

export interface OwnProfilePayload {
  version: string
  user: OwnProfile
  auth_method: AuthMethod
  local_password_enabled: boolean
  username_editable: boolean
  account_console_url: string
}

export interface PasswordPolicy {
  password_min_length: number
  password_max_length: number
  password_require_letter: boolean
  password_require_number: boolean
  password_require_symbol: boolean
}
