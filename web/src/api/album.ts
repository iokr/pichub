import { request } from './client'
import type { Album } from '@/types/api'

export interface AlbumCreatePayload {
  name: string
  description?: string
  is_public?: boolean
}

export interface AlbumUpdatePayload {
  name?: string
  description?: string
  is_public?: boolean
  cover_image_id?: number
  sort_order?: number
}

export const albumApi = {
  list(): Promise<Album[]> {
    const call = request<{ list: Album[] }>({ url: '/albums', method: 'get' })
    return call.then((data) => data?.list ?? [])
  },

  get(id: number): Promise<Album> {
    return request<Album>({ url: `/albums/${id}`, method: 'get' })
  },

  create(payload: AlbumCreatePayload): Promise<Album> {
    return request<Album>({ url: '/albums', method: 'post', data: payload })
  },

  update(id: number, payload: AlbumUpdatePayload): Promise<Album> {
    return request<Album>({ url: `/albums/${id}`, method: 'put', data: payload })
  },

  remove(id: number): Promise<void> {
    return request<void>({ url: `/albums/${id}`, method: 'delete' })
  },
}
