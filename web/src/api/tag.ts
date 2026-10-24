import { request } from './client'
import type { Tag } from '@/types/api'

export const tagApi = {
  list(): Promise<Tag[]> {
    return request<{ list: Tag[] }>({ url: '/tags', method: 'get' }).then((data) => data?.list ?? [])
  },

  create(payload: { name: string; color?: string }): Promise<Tag> {
    return request<Tag>({ url: '/tags', method: 'post', data: payload })
  },

  update(id: number, payload: { name?: string; color?: string }): Promise<Tag> {
    return request<Tag>({ url: `/tags/${id}`, method: 'put', data: payload })
  },

  remove(id: number): Promise<void> {
    return request<void>({ url: `/tags/${id}`, method: 'delete' })
  },
}
