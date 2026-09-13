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
  return path === 'login' || path === 'logout' || path.endsWith('/login') || path.endsWith('/logout')
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
  (error: AxiosError<{ message?: string }>) => {
    const status = error.response?.status
    const message = error.response?.data?.message || error.message || i18n.t('error.default')
    const url = error.config?.url || ''
    if (status === 401 && !isAuthEndpointUrl(url)) {
      handle401(message)
      ;(error as AxiosError & { __handled?: boolean; translatedMessage?: string }).__handled = true
    } else if (status) {
      antdMessage.error(message)
      ;(error as AxiosError & { __handled?: boolean }).__handled = true
    }
    ;(error as AxiosError & { translatedMessage?: string }).translatedMessage = message
    return Promise.reject(error)
  },
)

export default platformRequest
