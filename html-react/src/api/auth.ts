import request from '@/utils/request'
import type { ApiResponse, CaptchaInfo, LoginPayload, UserInfoPayload } from '@/types'

export function login(data: LoginPayload) {
  return request({
    url: '/login',
    method: 'post',
    data,
  }) as Promise<ApiResponse<{ token?: string; admin?: unknown }>>
}

export function getInfo() {
  return request({
    url: '/info',
    method: 'get',
  }) as Promise<ApiResponse<UserInfoPayload>>
}

export function logout() {
  return request({
    url: '/logout',
    method: 'post',
  }) as Promise<ApiResponse<unknown>>
}

export function getLoginCaptcha(params?: { check?: boolean; username?: string }) {
  const query: Record<string, string | number> = {}
  if (params?.check) {
    query.check = 1
  }
  const username = params?.username?.trim()
  if (username) {
    query.username = username
  }
  return request({
    url: '/login/captcha',
    method: 'get',
    params: Object.keys(query).length ? query : undefined,
  }) as Promise<ApiResponse<CaptchaInfo>>
}

export function getLoginBranding() {
  return request({
    url: '/login/branding',
    method: 'get',
    skipErrorMessage: true,
  }) as Promise<
    ApiResponse<{
      branding?: {
        site_enabled?: string
        site_name?: string
        site_logo?: string
        site_copyright?: string
      }
    }>
  >
}

export function getGoogleAuthenticatorStatus() {
  return request({
    url: '/google-authenticator/status',
    method: 'get',
  })
}

export function getGoogleAuthenticatorQRCode() {
  return request({
    url: '/google-authenticator/qrcode',
    method: 'get',
  })
}

export function bindGoogleAuthenticator(data: { secret: string; code: string }) {
  return request({
    url: '/google-authenticator/bind',
    method: 'post',
    data,
  })
}

export function unbindGoogleAuthenticator(data: { code: string }) {
  return request({
    url: '/google-authenticator/unbind',
    method: 'post',
    data,
  })
}

export function getAuthTokens() {
  return request({
    url: '/auth/tokens',
    method: 'get',
  }) as Promise<ApiResponse<{ tokens?: Array<Record<string, unknown>> }>>
}

export function revokeAuthToken(id: number | string) {
  return request({
    url: `/auth/tokens/${id}`,
    method: 'delete',
  })
}

export function revokeOtherAuthTokens() {
  return request({
    url: '/auth/tokens',
    method: 'delete',
    params: { except_current: 1 },
  })
}
