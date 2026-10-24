import { request } from './client'
import type { SiteConfig } from '@/types/api'

export const siteApi = {
  config(): Promise<SiteConfig> {
    return request<SiteConfig>({ url: '/site', method: 'get', silent: true })
  },
}
