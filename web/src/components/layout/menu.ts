import type { Component } from 'vue'
import {
  Coin,
  DataLine,
  Delete,
  Document,
  Folder,
  Key,
  Odometer,
  Picture,
  Setting,
  Share,
  Ticket,
  Tools,
  UploadFilled,
  User,
  UserFilled,
} from '@element-plus/icons-vue'

export interface NavItem {
  label: string
  path: string
  icon: Component
}

/** 用户端主导航。 */
export const mainNav: NavItem[] = [
  { label: '上传工作台', path: '/', icon: UploadFilled },
  { label: '图库', path: '/gallery', icon: Picture },
  { label: '相册', path: '/albums', icon: Folder },
  { label: '回收站', path: '/trash', icon: Delete },
  { label: '数据统计', path: '/stats', icon: DataLine },
  { label: '分享管理', path: '/shares', icon: Share },
  { label: 'API 令牌', path: '/tokens', icon: Key },
  { label: '设置', path: '/settings', icon: Setting },
]

/** 管理端导航。 */
export const adminNav: NavItem[] = [
  { label: '仪表盘', path: '/admin', icon: Odometer },
  { label: '用户管理', path: '/admin/users', icon: User },
  { label: '用户组', path: '/admin/groups', icon: UserFilled },
  { label: '存储策略', path: '/admin/storages', icon: Coin },
  { label: '系统设置', path: '/admin/settings', icon: Tools },
  { label: '操作日志', path: '/admin/logs', icon: Document },
  { label: '邀请码', path: '/admin/invites', icon: Ticket },
]

/** 移动端底部标签栏。 */
export const tabNav: NavItem[] = [
  { label: '工作台', path: '/', icon: UploadFilled },
  { label: '图库', path: '/gallery', icon: Picture },
  { label: '相册', path: '/albums', icon: Folder },
  { label: '我的', path: '/settings', icon: User },
]

export function matchActivePath(path: string): string {
  if (path === '/') {
    return '/'
  }
  const matched = [...mainNav, ...adminNav]
    .filter((item) => path === item.path || path.startsWith(`${item.path}/`))
    .sort((a, b) => b.path.length - a.path.length)
  return matched[0]?.path ?? path
}
