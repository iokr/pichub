import { request } from './client'
import type { StorageStrategy, StorageTestResult } from '@/types/api'

export interface StoragePayload {
  name: string
  driver: 'local' | 's3' | 'oss' | 'cos' | 'webdav'
  domain?: string
  path_prefix?: string
  is_default?: boolean
  status?: number
  root_path?: string
  path_template?: string
  endpoint?: string
  region?: string
  bucket?: string
  path_style?: boolean
  use_ssl?: boolean
  public_url?: string
  url?: string
  access_key?: string
  secret_key?: string
  username?: string
  password?: string
}

export type StorageUpdatePayload = Partial<Omit<StoragePayload, 'driver'>>

function unwrapList(data: { list: StorageStrategy[] } | StorageStrategy[] | null): StorageStrategy[] {
  if (!data) {
    return []
  }
  return Array.isArray(data) ? data : data.list ?? []
}

export const storageApi = {
  list(): Promise<StorageStrategy[]> {
    return request<{ list: StorageStrategy[] }>({ url: '/storages', method: 'get' }).then(unwrapList)
  },

  create(payload: StoragePayload): Promise<StorageStrategy> {
    return request<StorageStrategy>({ url: '/storages', method: 'post', data: payload })
  },

  update(id: number, payload: StorageUpdatePayload): Promise<StorageStrategy> {
    return request<StorageStrategy>({ url: `/storages/${id}`, method: 'put', data: payload })
  },

  remove(id: number): Promise<void> {
    return request<void>({ url: `/storages/${id}`, method: 'delete' })
  },

  test(id: number): Promise<StorageTestResult> {
    return request<StorageTestResult>({ url: `/storages/${id}/test`, method: 'post', timeout: 60_000 })
  },
}

/** 管理端存储策略（含系统级策略）。 */
export const adminStorageApi = {
  list(): Promise<StorageStrategy[]> {
    return request<{ list: StorageStrategy[] }>({ url: '/admin/storages', method: 'get' }).then(unwrapList)
  },

  create(payload: StoragePayload & { user_id?: number }): Promise<StorageStrategy> {
    return request<StorageStrategy>({ url: '/admin/storages', method: 'post', data: payload })
  },

  update(id: number, payload: StorageUpdatePayload & { user_id?: number }): Promise<StorageStrategy> {
    return request<StorageStrategy>({ url: `/admin/storages/${id}`, method: 'put', data: payload })
  },

  remove(id: number): Promise<void> {
    return request<void>({ url: `/admin/storages/${id}`, method: 'delete' })
  },

  test(id: number): Promise<StorageTestResult> {
    return request<StorageTestResult>({ url: `/admin/storages/${id}/test`, method: 'post', timeout: 60_000 })
  },
}
