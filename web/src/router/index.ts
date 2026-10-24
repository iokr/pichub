import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'

/** 路由元信息约定：layout 控制外层布局，public/auth 控制访问策略。 */
declare module 'vue-router' {
  interface RouteMeta {
    layout?: 'default' | 'blank'
    public?: boolean
    auth?: boolean
    admin?: boolean
    title?: string
  }
}

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'login',
    component: () => import('@/views/auth/LoginView.vue'),
    meta: { layout: 'blank', public: true, title: '登录' },
  },
  {
    path: '/register',
    name: 'register',
    component: () => import('@/views/auth/RegisterView.vue'),
    meta: { layout: 'blank', public: true, title: '注册' },
  },
  {
    path: '/verify-email',
    name: 'verify-email',
    component: () => import('@/views/auth/VerifyEmailView.vue'),
    meta: { layout: 'blank', public: true, title: '邮箱验证' },
  },
  {
    path: '/forgot-password',
    name: 'forgot-password',
    component: () => import('@/views/auth/ForgotPasswordView.vue'),
    meta: { layout: 'blank', public: true, title: '找回密码' },
  },
  {
    path: '/reset-password',
    name: 'reset-password',
    component: () => import('@/views/auth/ResetPasswordView.vue'),
    meta: { layout: 'blank', public: true, title: '重置密码' },
  },
  {
    path: '/s/:slug',
    name: 'share-access',
    component: () => import('@/views/share/ShareView.vue'),
    meta: { layout: 'blank', public: true, title: '分享' },
  },
  {
    path: '/',
    name: 'workspace',
    component: () => import('@/views/workspace/WorkspaceView.vue'),
    meta: { layout: 'default', auth: true, title: '上传工作台' },
  },
  {
    path: '/gallery',
    name: 'gallery',
    component: () => import('@/views/gallery/GalleryView.vue'),
    meta: { layout: 'default', auth: true, title: '图库' },
  },
  {
    path: '/albums',
    name: 'albums',
    component: () => import('@/views/albums/AlbumListView.vue'),
    meta: { layout: 'default', auth: true, title: '相册' },
  },
  {
    path: '/albums/:id',
    name: 'album-detail',
    component: () => import('@/views/albums/AlbumDetailView.vue'),
    meta: { layout: 'default', auth: true, title: '相册详情' },
  },
  {
    path: '/trash',
    name: 'trash',
    component: () => import('@/views/trash/TrashView.vue'),
    meta: { layout: 'default', auth: true, title: '回收站' },
  },
  {
    path: '/tokens',
    name: 'tokens',
    component: () => import('@/views/tokens/TokenView.vue'),
    meta: { layout: 'default', auth: true, title: 'API 令牌' },
  },
  {
    path: '/shares',
    name: 'shares',
    component: () => import('@/views/share/ShareManageView.vue'),
    meta: { layout: 'default', auth: true, title: '分享管理' },
  },
  {
    path: '/settings',
    name: 'settings',
    component: () => import('@/views/settings/SettingsView.vue'),
    meta: { layout: 'default', auth: true, title: '设置' },
  },
  {
    path: '/stats',
    name: 'stats',
    component: () => import('@/views/stats/StatsView.vue'),
    meta: { layout: 'default', auth: true, title: '数据统计' },
  },
  {
    path: '/admin',
    name: 'admin-dashboard',
    component: () => import('@/views/admin/DashboardView.vue'),
    meta: { layout: 'default', auth: true, admin: true, title: '管理仪表盘' },
  },
  {
    path: '/admin/users',
    name: 'admin-users',
    component: () => import('@/views/admin/UsersView.vue'),
    meta: { layout: 'default', auth: true, admin: true, title: '用户管理' },
  },
  {
    path: '/admin/groups',
    name: 'admin-groups',
    component: () => import('@/views/admin/GroupsView.vue'),
    meta: { layout: 'default', auth: true, admin: true, title: '用户组' },
  },
  {
    path: '/admin/storages',
    name: 'admin-storages',
    component: () => import('@/views/admin/StoragesView.vue'),
    meta: { layout: 'default', auth: true, admin: true, title: '存储策略' },
  },
  {
    path: '/admin/settings',
    name: 'admin-settings',
    component: () => import('@/views/admin/SettingsView.vue'),
    meta: { layout: 'default', auth: true, admin: true, title: '系统设置' },
  },
  {
    path: '/admin/logs',
    name: 'admin-logs',
    component: () => import('@/views/admin/LogsView.vue'),
    meta: { layout: 'default', auth: true, admin: true, title: '操作日志' },
  },
  {
    path: '/admin/invites',
    name: 'admin-invites',
    component: () => import('@/views/admin/InvitesView.vue'),
    meta: { layout: 'default', auth: true, admin: true, title: '邀请码' },
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'not-found',
    component: () => import('@/views/NotFoundView.vue'),
    meta: { layout: 'blank', public: true, title: '页面不存在' },
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior(to, from, savedPosition) {
    if (savedPosition) {
      return savedPosition
    }
    if (to.hash) {
      return { el: to.hash, behavior: 'smooth' }
    }
    return { top: 0 }
  },
})

export default router
export { routes }
