import type { PasswordPolicy } from '@/types/auth'

export function describePasswordPolicy(policy: PasswordPolicy | null): string {
  if (!policy) {
    return '新密码须符合平台安全策略（以后端校验为准）。'
  }
  const parts = [`长度 ${policy.password_min_length}-${policy.password_max_length} 个字符`]
  if (policy.password_require_letter) {
    parts.push('需包含字母')
  }
  if (policy.password_require_number) {
    parts.push('需包含数字')
  }
  if (policy.password_require_symbol) {
    parts.push('需包含符号')
  }
  return parts.join('；')
}

export function validateNewPassword(
  password: string,
  policy: PasswordPolicy | null,
): string | null {
  if (!password) {
    return '请输入新密码'
  }
  if (!policy) {
    return null
  }
  if (password.length < policy.password_min_length || password.length > policy.password_max_length) {
    return `密码长度必须在 ${policy.password_min_length} 到 ${policy.password_max_length} 个字符之间`
  }
  if (policy.password_require_letter && !/[A-Za-z]/.test(password)) {
    return '密码必须包含字母'
  }
  if (policy.password_require_number && !/\d/.test(password)) {
    return '密码必须包含数字'
  }
  if (policy.password_require_symbol && !/[^A-Za-z0-9]/.test(password)) {
    return '密码必须包含符号'
  }
  return null
}

export const LOCAL_PASSWORD_NOTICE =
  '注意：此处修改的密码仅用于本平台本地账号登录，不会修改统一认证中心（Keycloak）的密码。如果你使用统一认证中心登录，请前往统一认证中心修改密码。'

export const SSO_PASSWORD_NOTICE =
  '当前账号使用统一认证中心登录。本页面不会修改统一认证中心密码，请前往统一认证中心修改密码。'
