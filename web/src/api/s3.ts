import { request } from './client'
import type { S3Credential, S3CredentialCreateResult } from '@/types/api'

export const s3CredentialApi = {
  list(): Promise<S3Credential[]> {
    return request<{ list: S3Credential[] }>({ url: '/s3/credentials', method: 'get' }).then(
      (data) => data?.list ?? [],
    )
  },

  create(name: string): Promise<S3CredentialCreateResult> {
    return request<S3CredentialCreateResult>({ url: '/s3/credentials', method: 'post', data: { name } })
  },

  remove(id: number): Promise<void> {
    return request<void>({ url: `/s3/credentials/${id}`, method: 'delete' })
  },
}
