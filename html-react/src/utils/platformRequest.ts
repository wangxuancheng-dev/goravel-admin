import axios, { type AxiosError } from 'axios'
import { message as antdMessage } from 'antd'
import i18n from '@/i18n'
import Storage from './storage'
import { navigateTo, getCurrentPath } from './navigation'

const PLATFORM_TOKEN_KEY = 'platform_token'
const PLATFORM_ADMIN_KEY = 'platform_admin'

export function getPlatformToken(): string {
  return String(Storage.getItem(PLATFORM_TOKEN_KEY, '') || '')
}

export function setPlatformToken(token: string) {
  if (token) Storage.setItem(PLATFORM_TOKEN_KEY, token)
  else Storage.removeItem(PLATFORM_TOKEN_KEY)
}

export function getPlatformAdmin(): {
  id?: number
  username?: string
  name?: string
  role?: string
} | null {
  try {
    const raw = Storage.getItem(PLATFORM_ADMIN_KEY, '')
    return raw ? JSON.parse(String(raw)) : null
  } catch {
    return null
  }
}

export function setPlatformAdmin(admin: unknown) {
  if (admin) Storage.setItem(PLATFORM_ADMIN_KEY, JSON.stringify(admin))
  else Storage.removeItem(PLATFORM_ADMIN_KEY)
}

/** Owner has full platform ops; viewer is read-only (empty role => owner). */
export function isPlatformOwner(): boolean {
  const admin = getPlatformAdmin()
  return !admin || admin.role !== 'viewer'
}

export function clearPlatformSession() {
  setPlatformToken('')
  setPlatformAdmin(null)
}

function getBaseURL() {
  const apiBaseURL = import.meta.env.VITE_API_BASE_URL as string | undefined
  const apiPrefix = (import.meta.env.VITE_PLATFORM_API_PREFIX as string | undefined) || '/api/platform'
  if (apiBaseURL) {
    const base = apiBaseURL.replace(/\/+$/, '')
    const prefix = apiPrefix.startsWith('/') ? apiPrefix : `/${apiPrefix}`
    return `${base}${prefix}`
  }
  return apiPrefix
}

function isAuthEndpointUrl(url = '') {
  const path = url.split('?')[0].replace(/\/+$/, '')
  return (
    path === 'login' ||
    path === 'login/captcha' ||
    path === 'logout' ||
    path.endsWith('/login') ||
    path.endsWith('/login/captcha') ||
    path.endsWith('/logout')
  )
}

let isRedirecting = false

function handle401(msg?: string) {
  if (isRedirecting) return
  isRedirecting = true
  clearPlatformSession()
  if (!getCurrentPath().startsWith('/platform/login')) {
    antdMessage.error(msg || i18n.t('error.unauthorized'))
    navigateTo('/platform/login', { replace: true })
  }
  setTimeout(() => {
    isRedirecting = false
  }, 2000)
}

const platformRequest = axios.create({
  baseURL: getBaseURL(),
  timeout: 60000,
})

platformRequest.interceptors.request.use((config) => {
  const token = getPlatformToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token.trim()}`
  }
  const locale = i18n.language || Storage.getItem('language', 'zh-CN')
  config.headers['Accept-Language'] = locale === 'en-US' ? 'en-US' : 'zh-CN'
  return config
})

platformRequest.interceptors.response.use(
  (response) => {
    if (response.config?.responseType === 'blob' || response.data instanceof Blob) {
      return response.data
    }
    const res = response.data
    const url = response.config?.url || ''
    const isAuth = isAuthEndpointUrl(url)

    if (!isRedirecting) {
      const headerToken = response.headers.authorization || response.headers.Authorization
      if (headerToken) {
        const token = String(headerToken).replace(/^Bearer\s+/i, '').trim()
        if (token) setPlatformToken(token)
      }
      if (res?.data?.token) setPlatformToken(res.data.token)
    }

    if (res.code !== 200) {
      const message = res.message || i18n.t('error.default')
      if (!isAuth) {
        if (Number(res.code) === 401) handle401(message)
        else antdMessage.error(message)
      }
      const err = new Error(message) as Error & {
        code?: number
        errorCode?: string
        translatedMessage?: string
        __handled?: boolean
      }
      err.code = res.code
      err.errorCode = res.error_code
      err.translatedMessage = message
      if (!isAuth) err.__handled = true
      return Promise.reject(err)
    }
    return res
  },
  (error: AxiosError<{ message?: string; error_code?: string; code?: number }> & {
    errorCode?: string
    translatedMessage?: string
    __handled?: boolean
    businessCode?: number
  }) => {
    const status = error.response?.status
    const data = error.response?.data
    const message = data?.message || error.message || i18n.t('error.default')
    const errorCode = data?.error_code || ''
    const url = error.config?.url || ''
    const isAuth = isAuthEndpointUrl(url)

    error.errorCode = errorCode
    error.translatedMessage = message
    if (typeof status === 'number') error.businessCode = status

    if (status === 401 && !isAuth) {
      handle401(message)
      error.__handled = true
    } else if (isAuth && status && status >= 400) {
      // Login page handles captcha / 2FA adaptive UI; do not toast here.
      error.__handled = false
    } else if (status) {
      antdMessage.error(message)
      error.__handled = true
    }
    return Promise.reject(error)
  },
)

export default platformRequest
