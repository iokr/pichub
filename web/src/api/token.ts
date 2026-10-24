import { request } from './client'
import type { Token, TokenCreateResult } from '@/types/api'

export interface TokenCreatePayload {
  name: string
  scopes: string[]
  expires_in_days?: number
}

export const tokenApi = {
  list(): Promise<Token[]> {
    return request<{ list: Token[] }>({ url: '/tokens', method: 'get' }).then((data) => data?.list ?? [])
  },

  create(payload: TokenCreatePayload): Promise<TokenCreateResult> {
    return request<TokenCreateResult>({ url: '/tokens', method: 'post', data: payload })
  },

  remove(id: number): Promise<void> {
    return request<void>({ url: `/tokens/${id}`, method: 'delete' })
  },

  setStatus(id: number, status: number): Promise<void> {
    return request<void>({ url: `/tokens/${id}/status`, method: 'put', data: { status } })
  },
}
