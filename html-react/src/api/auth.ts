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

export function getLoginCaptcha(params?: { check?: boolean }) {
  return request({
    url: '/login/captcha',
    method: 'get',
    params: params?.check ? { check: 1 } : undefined,
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
        site_theme_color?: string
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
