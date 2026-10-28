import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { fetchAuthMe, login as loginRequest } from '@/api/auth'
import type { AuthMe, AuthMethod, LoginResult } from '@/types/auth'

/** Separate keys from admin-web so both apps can run in one browser. */
const TOKEN_KEY = 'qz_wb_access_token'
const TENANT_KEY = 'qz_wb_tenant_id'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem(TOKEN_KEY) || '')
  const tenantId = ref(localStorage.getItem(TENANT_KEY) || '')
  const user = ref<AuthMe['user'] | null>(null)
  const version = ref('')
  const permissions = ref<string[]>([])
  const dataScope = ref(1)
  const authMethod = ref<AuthMethod>('local')
  const localPasswordEnabled = ref(false)
  const usernameEditable = ref(false)
  const accountConsoleUrl = ref('')
  const bootstrapped = ref(false)

  const isAuthenticated = computed(() => Boolean(token.value && user.value))
  const displayName = computed(() => user.value?.nickname || user.value?.username || '未登录')

  function persist(nextToken: string, nextTenant: string) {
    token.value = nextToken
    tenantId.value = nextTenant
    if (nextToken) {
      localStorage.setItem(TOKEN_KEY, nextToken)
    } else {
      localStorage.removeItem(TOKEN_KEY)
    }
    if (nextTenant) {
      localStorage.setItem(TENANT_KEY, nextTenant)
    } else {
      localStorage.removeItem(TENANT_KEY)
    }
  }

  function applyCredentialFlags(source: Partial<AuthMe>) {
    if (source.auth_method === 'sso' || source.auth_method === 'local') {
      authMethod.value = source.auth_method
    }
    if (typeof source.local_password_enabled === 'boolean') {
      localPasswordEnabled.value = source.local_password_enabled
    }
    if (typeof source.username_editable === 'boolean') {
      usernameEditable.value = source.username_editable
    }
    if (typeof source.account_console_url === 'string') {
      accountConsoleUrl.value = source.account_console_url
    }
  }

  function applySession(result: LoginResult | AuthMe) {
    if ('access_token' in result) {
      persist(result.access_token, result.tenant_id)
      // Login responses do not yet include credential flags; restore() fills them.
      authMethod.value = 'local'
      localPasswordEnabled.value = true
      usernameEditable.value = false
      accountConsoleUrl.value = ''
    } else {
      persist(token.value, result.tenant_id)
      applyCredentialFlags(result)
    }
    user.value = result.user
    if ('version' in result && result.version) {
      version.value = result.version
    }
    permissions.value = result.permissions ?? []
    dataScope.value = result.data_scope
  }

  function applyProfile(profile: {
    nickname: string
    avatar: string
    local_password_enabled?: boolean
  }) {
    if (!user.value) {
      return
    }
    user.value = { ...user.value, nickname: profile.nickname, avatar: profile.avatar }
    if (typeof profile.local_password_enabled === 'boolean') {
      localPasswordEnabled.value = profile.local_password_enabled
    }
  }

  function applyAuthMeta(meta: {
    auth_method?: AuthMethod
    local_password_enabled?: boolean
    username_editable?: boolean
    account_console_url?: string
  }) {
    applyCredentialFlags(meta)
  }

  async function login(username: string, password: string, nextTenantId?: string) {
    const response = await loginRequest({
      username,
      password,
      tenant_id: nextTenantId || undefined,
    })
    applySession(response.data)
    // Refresh me to load auth_method / local_password_enabled from backend.
    await restoreSessionFlags()
    bootstrapped.value = true
    return response.data
  }

  async function applyExternalLogin(result: LoginResult) {
    applySession(result)
    authMethod.value = 'sso'
    await restoreSessionFlags()
    bootstrapped.value = true
  }

  async function restoreSessionFlags() {
    if (!token.value) {
      return
    }
    try {
      const response = await fetchAuthMe()
      applyCredentialFlags(response.data)
      user.value = response.data.user
      permissions.value = response.data.permissions ?? []
      dataScope.value = response.data.data_scope
      version.value = response.data.version
      tenantId.value = response.data.tenant_id
    } catch {
      /* keep login result if me fails transiently */
    }
  }

  async function restore() {
    if (!token.value) {
      clear()
      bootstrapped.value = true
      return false
    }
    try {
      const response = await fetchAuthMe()
      applySession(response.data)
      bootstrapped.value = true
      return true
    } catch {
      clear()
      bootstrapped.value = true
      return false
    }
  }

  function clear() {
    persist('', '')
    user.value = null
    version.value = ''
    permissions.value = []
    dataScope.value = 1
    authMethod.value = 'local'
    localPasswordEnabled.value = false
    usernameEditable.value = false
    accountConsoleUrl.value = ''
  }

  /**
   * Platform JWT is self-contained; there is no server logout endpoint.
   * Clearing local storage invalidates this browser session only.
   */
  function logout() {
    clear()
  }

  return {
    token,
    tenantId,
    user,
    version,
    permissions,
    dataScope,
    authMethod,
    localPasswordEnabled,
    usernameEditable,
    accountConsoleUrl,
    bootstrapped,
    isAuthenticated,
    displayName,
    login,
    applyExternalLogin,
    restore,
    logout,
    applyProfile,
    applyAuthMeta,
    clear,
  }
})
