import { request } from './client'
import type { AuthResult, Captcha, CheckResult } from '@/types/api'

export interface LoginPayload {
  account: string
  password: string
  captcha_id?: string
  captcha_code?: string
}

export interface RegisterPayload {
  username: string
  email: string
  password: string
  captcha_id?: string
  captcha_code?: string
  invite_code?: string
}

export interface CaptchaPayload {
  captcha_id?: string
  captcha_code?: string
}

export const authApi = {
  captcha(): Promise<Captcha> {
    return request<Captcha>({ url: '/auth/captcha', method: 'get', silent: true })
  },

  login(payload: LoginPayload): Promise<AuthResult> {
    return request<AuthResult>({ url: '/auth/login', method: 'post', data: payload })
  },

  register(payload: RegisterPayload): Promise<AuthResult> {
    return request<AuthResult>({ url: '/auth/register', method: 'post', data: payload })
  },

  refresh(refreshToken: string): Promise<AuthResult> {
    return request<AuthResult>({
      url: '/auth/refresh',
      method: 'post',
      data: { refresh_token: refreshToken },
      skipAuthRefresh: true,
      silent: true,
    })
  },

  logout(refreshToken: string): Promise<void> {
    return request<void>({
      url: '/auth/logout',
      method: 'post',
      data: { refresh_token: refreshToken },
      silent: true,
    })
  },

  checkUsername(username: string): Promise<CheckResult> {
    return request<CheckResult>({
      url: '/auth/check-username',
      method: 'get',
      params: { u: username },
      silent: true,
    })
  },

  checkEmail(email: string): Promise<CheckResult> {
    return request<CheckResult>({
      url: '/auth/check-email',
      method: 'get',
      params: { e: email },
      silent: true,
    })
  },

  verifyEmail(token: string): Promise<void> {
    return request<void>({ url: '/auth/verify-email', method: 'post', data: { token } })
  },

  resendVerify(email: string, captcha?: CaptchaPayload): Promise<void> {
    return request<void>({
      url: '/auth/resend-verify',
      method: 'post',
      data: { email, ...captcha },
    })
  },

  forgotPassword(email: string, captcha?: CaptchaPayload): Promise<void> {
    return request<void>({
      url: '/auth/forgot-password',
      method: 'post',
      data: { email, ...captcha },
    })
  },

  resetPassword(token: string, password: string): Promise<void> {
    return request<void>({
      url: '/auth/reset-password',
      method: 'post',
      data: { token, password },
    })
  },
}
