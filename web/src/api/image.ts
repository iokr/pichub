import { request } from './client'
import type {
  BatchResult,
  Image,
  ImageListQuery,
  ImageUpdatePayload,
  PageResp,
  Referer,
  RefererSummary,
} from '@/types/api'

export type ImageBatchAction = 'delete' | 'restore' | 'move' | 'public' | 'private'

export const imageApi = {
  list(params: ImageListQuery = {}): Promise<PageResp<Image>> {
    return request<PageResp<Image>>({ url: '/images', method: 'get', params })
  },

  trash(params: ImageListQuery = {}): Promise<PageResp<Image>> {
    return request<PageResp<Image>>({ url: '/images/trash', method: 'get', params })
  },

  get(id: number): Promise<Image> {
    return request<Image>({ url: `/images/${id}`, method: 'get' })
  },

  update(id: number, payload: ImageUpdatePayload): Promise<Image> {
    return request<Image>({ url: `/images/${id}`, method: 'patch', data: payload })
  },

  remove(id: number): Promise<void> {
    return request<void>({ url: `/images/${id}`, method: 'delete' })
  },

  hardDelete(id: number): Promise<void> {
    return request<void>({ url: `/images/${id}/permanent`, method: 'delete' })
  },

  restore(id: number): Promise<void> {
    return request<void>({ url: `/images/${id}/restore`, method: 'post' })
  },

  clearTrash(): Promise<BatchResult> {
    return request<BatchResult>({ url: '/images/trash', method: 'delete' })
  },

  batch(ids: number[], action: ImageBatchAction, albumId?: number): Promise<BatchResult> {
    return request<BatchResult>({
      url: '/images/batch',
      method: 'post',
      data: { ids, action, album_id: albumId },
    })
  },

  referers(
    id: number,
    params: { page?: number; size?: number; order?: 'hits' | 'recent' } = {},
  ): Promise<PageResp<Referer>> {
    return request<PageResp<Referer>>({ url: `/images/${id}/referers`, method: 'get', params })
  },

  refererSummary(id: number): Promise<RefererSummary> {
    return request<RefererSummary>({ url: `/images/${id}/referers/summary`, method: 'get' })
  },

  deleteReferer(id: number, refererId: number): Promise<void> {
    return request<void>({ url: `/images/${id}/referers/${refererId}`, method: 'delete' })
  },
}
