import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { fetchAuthMe, login as loginRequest, type AuthMe, type LoginResult } from '@/api/auth'

const TOKEN_KEY = 'qz_access_token'
const TENANT_KEY = 'qz_tenant_id'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem(TOKEN_KEY) || '')
  const tenantId = ref(localStorage.getItem(TENANT_KEY) || '')
  const user = ref<AuthMe['user'] | null>(null)
  const permissions = ref<string[]>([])
  const dataScope = ref(1)
  const bootstrapped = ref(false)

  const isAuthenticated = computed(() => Boolean(token.value))
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

  function applySession(result: LoginResult | AuthMe) {
    if ('access_token' in result) {
      persist(result.access_token, result.tenant_id)
    } else {
      persist(token.value, result.tenant_id)
    }
    user.value = result.user
    permissions.value = result.permissions ?? []
    dataScope.value = result.data_scope
  }

  async function login(username: string, password: string, nextTenantId?: string) {
    const response = await loginRequest({
      username,
      password,
      tenant_id: nextTenantId || undefined,
    })
    applySession(response.data)
    bootstrapped.value = true
    return response.data
  }

  async function restore() {
    if (!token.value) {
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
    permissions.value = []
    dataScope.value = 1
  }

  function logout() {
    clear()
  }

  function hasPermission(permission: string) {
    return permissions.value.includes(permission)
  }

  return {
    token,
    tenantId,
    user,
    permissions,
    dataScope,
    bootstrapped,
    isAuthenticated,
    displayName,
    login,
    restore,
    logout,
    clear,
    hasPermission,
  }
})
