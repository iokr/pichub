import axios, { type AxiosError, type AxiosRequestConfig, type AxiosResponse } from 'axios'
import { ElMessage } from 'element-plus'

import type { ApiResponse, AuthResult, User } from '@/types/api'

export const API_BASE = (import.meta.env.VITE_API_BASE as string) || '/api/v1'

const ACCESS_TOKEN_KEY = 'pichub.access_token'
const REFRESH_TOKEN_KEY = 'pichub.refresh_token'
const USER_INFO_KEY = 'pichub.user'

const storage = {
  get(key: string): string {
    try {
      return window.localStorage.getItem(key) || ''
    } catch {
      return ''
    }
  },
  set(key: string, value: string): void {
    try {
      if (value) {
        window.localStorage.setItem(key, value)
      } else {
        window.localStorage.removeItem(key)
      }
    } catch {
      /* 存储不可用时忽略 */
    }
  },
}

/** 访问令牌与刷新令牌的本地存储。 */
export const tokenStorage = {
  access(): string {
    return storage.get(ACCESS_TOKEN_KEY)
  },
  refresh(): string {
    return storage.get(REFRESH_TOKEN_KEY)
  },
  save(access: string, refresh?: string): void {
    storage.set(ACCESS_TOKEN_KEY, access)
    if (refresh) {
      storage.set(REFRESH_TOKEN_KEY, refresh)
    }
  },
  clear(): void {
    storage.set(ACCESS_TOKEN_KEY, '')
    storage.set(REFRESH_TOKEN_KEY, '')
    storage.set(USER_INFO_KEY, '')
  },
}

/** 当前用户的本地缓存，用于刷新页面后立即渲染。 */
export const userStorage = {
  get(): User | null {
    const raw = storage.get(USER_INFO_KEY)
    if (!raw) {
      return null
    }
    try {
      return JSON.parse(raw) as User
    } catch {
      return null
    }
  },
  set(user: User | null | undefined): void {
    storage.set(USER_INFO_KEY, user ? JSON.stringify(user) : '')
  },
  clear(): void {
    storage.set(USER_INFO_KEY, '')
  },
}

/** 业务异常，同时携带业务码与 HTTP 状态。 */
export class ApiError extends Error {
  readonly code: number
  readonly status: number

  constructor(message: string, code = -1, status = 0) {
    super(message)
    this.name = 'ApiError'
    this.code = code
    this.status = status
  }
}

const HTTP_MESSAGES: Record<number, string> = {
  400: '请求参数有误',
  401: '登录状态已失效，请重新登录',
  403: '没有权限执行该操作',
  404: '请求的资源不存在',
  409: '数据冲突，请刷新后重试',
  413: '文件体积超出限制',
  429: '操作过于频繁，请稍后再试',
  500: '服务器开小差了，请稍后再试',
  502: '网关异常，请稍后再试',
  503: '服务暂不可用',
  504: '网关超时，请稍后再试',
}

type UnauthorizedHandler = () => void

let unauthorizedHandler: UnauthorizedHandler | null = null

/** 注册登录失效后的跳转逻辑，避免 api 层直接依赖路由。 */
export function setUnauthorizedHandler(handler: UnauthorizedHandler | null): void {
  unauthorizedHandler = handler
}

function clearSession(): void {
  tokenStorage.clear()
  if (unauthorizedHandler) {
    unauthorizedHandler()
  }
}

const http = axios.create({
  baseURL: API_BASE,
  timeout: 30_000,
})

http.interceptors.request.use((config) => {
  const token = tokenStorage.access()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

let refreshing: Promise<string> | null = null

/** 使用刷新令牌换取新的一对令牌，并发调用共享同一个请求。 */
function refreshAccessToken(): Promise<string> {
  if (refreshing) {
    return refreshing
  }
  const refreshToken = tokenStorage.refresh()
  if (!refreshToken) {
    return Promise.reject(new ApiError('登录状态已失效，请重新登录', 40100, 401))
  }
  refreshing = axios
    .post<ApiResponse<AuthResult>>(`${API_BASE}/auth/refresh`, { refresh_token: refreshToken })
    .then((res) => {
      const body = res.data
      if (!body || body.code !== 0 || !body.data) {
        throw new ApiError(body?.message || '刷新登录状态失败', body?.code ?? -1, res.status)
      }
      tokenStorage.save(body.data.access_token, body.data.refresh_token)
      userStorage.set(body.data.user)
      return body.data.access_token
    })
    .finally(() => {
      refreshing = null
    })
  return refreshing
}

function toApiError(error: unknown): ApiError {
  if (error instanceof ApiError) {
    return error
  }
  if (axios.isAxiosError(error)) {
    const axiosError = error as AxiosError<ApiResponse<unknown>>
    const status = axiosError.response?.status ?? 0
    const body = axiosError.response?.data
    if (body && typeof body === 'object' && 'message' in body) {
      return new ApiError(String(body.message || '') || '请求失败', Number(body.code ?? -1), status)
    }
    if (axiosError.code === 'ECONNABORTED') {
      return new ApiError('请求超时，请稍后重试', -1, status)
    }
    if (status === 0) {
      return new ApiError(axiosError.message?.includes('abort') ? '请求已取消' : '网络异常，请检查网络连接', -1, 0)
    }
    return new ApiError(HTTP_MESSAGES[status] || `请求失败（${status}）`, -1, status)
  }
  return new ApiError(error instanceof Error ? error.message : '请求失败')
}

function unpack<T>(res: AxiosResponse<ApiResponse<T>>): T {
  const body = res.data
  if (!body || typeof body.code !== 'number') {
    return body as unknown as T
  }
  if (body.code !== 0) {
    throw new ApiError(body.message || '请求失败', body.code, res.status)
  }
  return body.data as T
}

export interface RequestConfig extends AxiosRequestConfig {
  /** 为 true 时不弹出全局错误提示。 */
  silent?: boolean
  /** 为 true 时 401 不触发刷新令牌流程。 */
  skipAuthRefresh?: boolean
}

/** 发起请求并解包统一响应结构。 */
export async function request<T>(options: RequestConfig = {}): Promise<T> {
  const { silent = false, skipAuthRefresh = false, ...config } = options
  try {
    return unpack<T>(await http.request<ApiResponse<T>>(config))
  } catch (error) {
    let apiError = toApiError(error)
    if (apiError.status === 401 && !skipAuthRefresh && tokenStorage.refresh()) {
      try {
        await refreshAccessToken()
        return unpack<T>(await http.request<ApiResponse<T>>(config))
      } catch (retryError) {
        apiError = toApiError(retryError)
        clearSession()
      }
    } else if (apiError.status === 401) {
      clearSession()
    }
    if (!silent) {
      ElMessage.error(apiError.message)
    }
    throw apiError
  }
}

/** 以原始 AxiosResponse 返回，用于下载等需要读取响应头的场景。 */
export async function requestRaw(
  options: RequestConfig = {},
): Promise<AxiosResponse<Blob>> {
  const { silent = false, ...config } = options
  try {
    return await http.request<Blob>({ ...config, responseType: 'blob' })
  } catch (error) {
    const apiError = toApiError(error)
    if (!silent) {
      ElMessage.error(apiError.message)
    }
    throw apiError
  }
}

export { http }
