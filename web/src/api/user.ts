import { request } from './client'
import type { User, UserStats } from '@/types/api'

export interface UpdateProfilePayload {
  nickname?: string
  avatar_url?: string
  default_visibility?: 'public' | 'private'
  referer_stat_enabled?: boolean
  storage_config_id?: number
}

export interface ChangePasswordPayload {
  old_password: string
  new_password: string
  captcha_id?: string
  captcha_code?: string
}

export const userApi = {
  profile(): Promise<User> {
    return request<User>({ url: '/user/profile', method: 'get' })
  },

  updateProfile(payload: UpdateProfilePayload): Promise<User> {
    return request<User>({ url: '/user/profile', method: 'put', data: payload })
  },

  changePassword(payload: ChangePasswordPayload): Promise<void> {
    return request<void>({ url: '/user/password', method: 'put', data: payload })
  },

  stats(): Promise<UserStats> {
    return request<UserStats>({ url: '/user/stats', method: 'get' })
  },
}
