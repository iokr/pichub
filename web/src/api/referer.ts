import { request } from './client'
import type { HostStat } from '@/types/api'

export interface RefererTopParams {
  limit?: number
  days?: number
}

export const refererApi = {
  top(params: RefererTopParams = {}): Promise<HostStat[]> {
    return request<{ list: HostStat[] }>({ url: '/stats/referers/top', method: 'get', params }).then(
      (data) => data?.list ?? [],
    )
  },

  trend(days = 30): Promise<{ date: string; hits: number }[]> {
    return request<{ list: { date: string; hits: number }[] }>({
      url: '/stats/referers/trend',
      method: 'get',
      params: { days },
    }).then((data) => data?.list ?? [])
  },
}
