import { request } from './client'
import type { ChunkInitResult, UploadCheckResult, UploadOptions, UploadResult } from '@/types/api'

export interface ChunkInitPayload {
  name: string
  size: number
  hash: string
  chunks: number
  album_id?: number
  tags?: string
  is_public?: boolean
  storage_id?: number
}

function buildForm(file: Blob, fields: Record<string, string | number | boolean | undefined>): FormData {
  const form = new FormData()
  for (const [key, value] of Object.entries(fields)) {
    if (value === undefined || value === '') {
      continue
    }
    form.append(key, typeof value === 'boolean' ? String(value) : String(value))
  }
  form.append('file', file)
  return form
}

export const uploadApi = {
  /** 单文件上传，onProgress 返回 0-100 的百分比。 */
  upload(
    file: File,
    options: UploadOptions = {},
    onProgress?: (percent: number) => void,
    signal?: AbortSignal,
  ): Promise<UploadResult> {
    const form = buildForm(file, {
      album_id: options.album_id,
      tags: options.tags,
      is_public: options.is_public,
      storage_id: options.storage_id,
    })
    return request<UploadResult>({
      url: '/upload',
      method: 'post',
      data: form,
      timeout: 0,
      signal,
      silent: true,
      onUploadProgress: (event) => {
        if (onProgress && event.total) {
          onProgress(Math.min(100, Math.round((event.loaded / event.total) * 100)))
        }
      },
    })
  },

  /** 秒传检测，命中时直接返回已有图片。 */
  check(payload: { hash: string; size: number; name?: string; ext?: string }): Promise<UploadCheckResult> {
    return request<UploadCheckResult>({ url: '/upload/check', method: 'post', data: payload, silent: true })
  },

  /** 通过远程 URL 转存图片。 */
  uploadURL(payload: {
    url: string
    album_id?: number
    tags?: string
    is_public?: boolean
    storage_id?: number
  }): Promise<UploadResult> {
    return request<UploadResult>({ url: '/upload/url', method: 'post', data: payload, timeout: 0, silent: true })
  },

  chunkInit(payload: ChunkInitPayload): Promise<ChunkInitResult> {
    return request<ChunkInitResult>({ url: '/upload/chunk/init', method: 'post', data: payload, silent: true })
  },

  chunkPart(
    uploadId: string,
    index: number,
    blob: Blob,
    onProgress?: (percent: number) => void,
    signal?: AbortSignal,
  ): Promise<{ index: number; size: number }> {
    const form = buildForm(blob, { index })
    return request<{ index: number; size: number }>({
      url: `/upload/chunk/${encodeURIComponent(uploadId)}`,
      method: 'post',
      data: form,
      timeout: 0,
      signal,
      silent: true,
      onUploadProgress: (event) => {
        if (onProgress && event.total) {
          onProgress(Math.min(100, Math.round((event.loaded / event.total) * 100)))
        }
      },
    })
  },

  chunkComplete(uploadId: string, hash?: string): Promise<UploadResult> {
    return request<UploadResult>({
      url: `/upload/chunk/${encodeURIComponent(uploadId)}/complete`,
      method: 'post',
      data: { hash },
      timeout: 0,
      silent: true,
    })
  },
}
