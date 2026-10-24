import { computed, ref } from 'vue'
import { defineStore } from 'pinia'

import { authApi, type LoginPayload, type RegisterPayload } from '@/api/auth'
import { tokenStorage, userStorage } from '@/api/client'
import { userApi, type UpdateProfilePayload } from '@/api/user'
import type { AuthResult, User } from '@/types/api'

export const useUserStore = defineStore('user', () => {
  const user = ref<User | null>(userStorage.get())
  const authenticated = ref(!!tokenStorage.access())
  const loading = ref(false)

  const isAdmin = computed(() => user.value?.role === 1)
  const displayName = computed(() => user.value?.nickname || user.value?.username || '')
  const quotaPercent = computed(() => {
    const current = user.value
    if (!current || !current.quota_bytes) {
      return 0
    }
    return Math.min(100, Math.round((current.used_bytes / current.quota_bytes) * 100))
  })

  function applyAuth(result: AuthResult): void {
    tokenStorage.save(result.access_token, result.refresh_token)
    user.value = result.user
    userStorage.set(result.user)
    authenticated.value = true
  }

  async function login(payload: LoginPayload): Promise<AuthResult> {
    loading.value = true
    try {
      const result = await authApi.login(payload)
      applyAuth(result)
      return result
    } finally {
      loading.value = false
    }
  }

  async function register(payload: RegisterPayload): Promise<AuthResult> {
    loading.value = true
    try {
      const result = await authApi.register(payload)
      applyAuth(result)
      return result
    } finally {
      loading.value = false
    }
  }

  async function fetchProfile(): Promise<User> {
    const data = await userApi.profile()
    user.value = data
    userStorage.set(data)
    authenticated.value = true
    return data
  }

  async function updateProfile(payload: UpdateProfilePayload): Promise<User> {
    const data = await userApi.updateProfile(payload)
    user.value = data
    userStorage.set(data)
    return data
  }

  function reset(): void {
    tokenStorage.clear()
    user.value = null
    authenticated.value = false
  }

  async function logout(): Promise<void> {
    const refresh = tokenStorage.refresh()
    if (refresh) {
      try {
        await authApi.logout(refresh)
      } catch {
        /* 登出失败不影响本地清理 */
      }
    }
    reset()
  }

  return {
    user,
    authenticated,
    loading,
    isAdmin,
    displayName,
    quotaPercent,
    login,
    register,
    fetchProfile,
    updateProfile,
    logout,
    reset,
  }
})
