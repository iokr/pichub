import { request } from './client'
import type { Share, ShareAccess } from '@/types/api'

export interface ShareCreatePayload {
  target_type: 'image' | 'album'
  target_id: number
  password?: string
  expires_in_hours?: number
  max_views?: number
}

export const shareApi = {
  list(): Promise<Share[]> {
    return request<{ list: Share[] }>({ url: '/shares', method: 'get' }).then((data) => data?.list ?? [])
  },

  create(payload: ShareCreatePayload): Promise<Share> {
    return request<Share>({ url: '/shares', method: 'post', data: payload })
  },

  remove(id: number): Promise<void> {
    return request<void>({ url: `/shares/${id}`, method: 'delete' })
  },

  /** 匿名访问分享内容。 */
  access(slug: string, password?: string): Promise<ShareAccess> {
    return request<ShareAccess>({
      url: `/s/${encodeURIComponent(slug)}`,
      method: 'get',
      params: password ? { password } : undefined,
      silent: true,
      skipAuthRefresh: true,
    })
  },
}
