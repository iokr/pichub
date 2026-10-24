import { ElMessage } from 'element-plus'
import type { Router } from 'vue-router'

import { setUnauthorizedHandler } from '@/api/client'
import { useSettingStore } from '@/stores/setting'
import { useUserStore } from '@/stores/user'

/** 安装全局路由守卫与登录失效处理。 */
export function setupRouterGuards(router: Router): void {
  router.beforeEach(async (to) => {
    const setting = useSettingStore()
    const user = useUserStore()

    if (!setting.loaded) {
      await setting.load()
    }
    setting.setTitle(to.meta.title)

    const requiresAuth = to.meta.auth === true

    if (requiresAuth && !user.authenticated) {
      return { path: '/login', query: { redirect: to.fullPath } }
    }

    if (user.authenticated && !user.user) {
      try {
        await user.fetchProfile()
      } catch {
        user.reset()
        if (requiresAuth) {
          return { path: '/login', query: { redirect: to.fullPath } }
        }
      }
    }

    if (to.meta.admin === true && !user.isAdmin) {
      ElMessage.warning('需要管理员权限')
      return { path: '/' }
    }

    if (to.meta.public === true && user.authenticated && (to.path === '/login' || to.path === '/register')) {
      return { path: '/' }
    }

    return true
  })

  setUnauthorizedHandler(() => {
    const current = router.currentRoute.value
    if (current.meta.auth === true) {
      void router.replace({ path: '/login', query: { redirect: current.fullPath } })
    }
  })
}
