import { request } from './client'
import type {
  AdminUser,
  Dashboard,
  EmailLog,
  Group,
  Invite,
  OperationLog,
  PageResp,
  SettingItem,
} from '@/types/api'

export interface AdminUsersParams {
  keyword?: string
  status?: number
  role?: number
  group_id?: number
  page?: number
  size?: number
}

export interface AdminLogsParams {
  user_id?: number
  action?: string
  page?: number
  size?: number
}

export interface AdminEmailLogsParams {
  user_id?: number
  status?: number
  page?: number
  size?: number
}

export interface GroupPayload {
  name: string
  description?: string
  quota_bytes?: number
  max_file_size?: number
  daily_limit?: number
  max_albums?: number
  max_tokens?: number
  max_shares?: number
  allow_register?: boolean
  allow_custom_storage?: boolean
  is_default?: boolean
  status?: number
}

export interface InviteCreatePayload {
  max_uses?: number
  group_id?: number
  expires_in_hours?: number
  count?: number
}

export const adminApi = {
  dashboard(): Promise<Dashboard> {
    return request<Dashboard>({ url: '/admin/dashboard', method: 'get' })
  },

  users(params: AdminUsersParams = {}): Promise<PageResp<AdminUser>> {
    return request<PageResp<AdminUser>>({ url: '/admin/users', method: 'get', params })
  },

  updateUserStatus(id: number, status: number, reason?: string): Promise<void> {
    return request<void>({ url: `/admin/users/${id}/status`, method: 'put', data: { status, reason } })
  },

  updateUserGroup(
    id: number,
    payload: { group_id: number; quota_bytes?: number; max_file_size?: number; daily_limit?: number },
  ): Promise<void> {
    return request<void>({ url: `/admin/users/${id}/group`, method: 'put', data: payload })
  },

  resetUserPassword(id: number, newPassword: string): Promise<void> {
    return request<void>({ url: `/admin/users/${id}/password`, method: 'put', data: { new_password: newPassword } })
  },

  groups(): Promise<Group[]> {
    return request<{ list: Group[] }>({ url: '/admin/groups', method: 'get' }).then((data) => data?.list ?? [])
  },

  createGroup(payload: GroupPayload): Promise<Group> {
    return request<Group>({ url: '/admin/groups', method: 'post', data: payload })
  },

  updateGroup(id: number, payload: Partial<GroupPayload>): Promise<Group> {
    return request<Group>({ url: `/admin/groups/${id}`, method: 'put', data: payload })
  },

  deleteGroup(id: number): Promise<void> {
    return request<void>({ url: `/admin/groups/${id}`, method: 'delete' })
  },

  settings(): Promise<SettingItem[]> {
    return request<{ list: SettingItem[] }>({ url: '/admin/settings', method: 'get' }).then(
      (data) => data?.list ?? [],
    )
  },

  updateSettings(settings: Record<string, string>): Promise<void> {
    return request<void>({ url: '/admin/settings', method: 'put', data: { settings } })
  },

  logs(params: AdminLogsParams = {}): Promise<PageResp<OperationLog>> {
    return request<PageResp<OperationLog>>({ url: '/admin/logs', method: 'get', params })
  },

  emailLogs(params: AdminEmailLogsParams = {}): Promise<PageResp<EmailLog>> {
    return request<PageResp<EmailLog>>({ url: '/admin/email-logs', method: 'get', params })
  },

  invites(page = 1, size = 20): Promise<PageResp<Invite>> {
    return request<PageResp<Invite>>({ url: '/admin/invites', method: 'get', params: { page, size } })
  },

  createInvites(payload: InviteCreatePayload): Promise<Invite[]> {
    return request<{ list: Invite[] }>({ url: '/admin/invites', method: 'post', data: payload }).then(
      (data) => data?.list ?? [],
    )
  },

  deleteInvite(id: number): Promise<void> {
    return request<void>({ url: `/admin/invites/${id}`, method: 'delete' })
  },
}
