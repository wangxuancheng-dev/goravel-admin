import axios from 'axios'
import { ElMessage } from 'element-plus'
import router from '../router'
import i18n from '../i18n'
import Storage from './storage'

const { t } = i18n.global

const PLATFORM_TOKEN_KEY = 'platform_token'

export function getPlatformToken() {
  return Storage.getItem(PLATFORM_TOKEN_KEY, '') || ''
}

export function setPlatformToken(token) {
  if (token) {
    Storage.setItem(PLATFORM_TOKEN_KEY, token)
  } else {
    Storage.removeItem(PLATFORM_TOKEN_KEY)
  }
}

export function clearPlatformSession() {
  setPlatformToken('')
  Storage.removeItem('platform_admin')
}

export function getPlatformAdmin() {
  try {
    const raw = Storage.getItem('platform_admin', '')
    return raw ? JSON.parse(raw) : null
  } catch {
    return null
  }
}

export function setPlatformAdmin(admin) {
  if (admin) {
    Storage.setItem('platform_admin', JSON.stringify(admin))
  } else {
    Storage.removeItem('platform_admin')
  }
}

/** Owner has full platform ops; viewer is read-only (empty role => owner). */
export function isPlatformOwner() {
  const admin = getPlatformAdmin()
  return !admin || admin.role !== 'viewer'
}

const getBaseURL = () => {
  const apiBaseURL = import.meta.env.VITE_API_BASE_URL
  const apiPrefix = import.meta.env.VITE_PLATFORM_API_PREFIX || '/api/platform'
  if (apiBaseURL) {
    const base = apiBaseURL.replace(/\/+$/, '')
    const prefix = apiPrefix.startsWith('/') ? apiPrefix : `/${apiPrefix}`
    return `${base}${prefix}`
  }
  return apiPrefix
}

const platformRequest = axios.create({
  baseURL: getBaseURL(),
  timeout: 60000
})

function isAuthEndpointUrl(url = '') {
  const path = String(url).split('?')[0].replace(/\/+$/, '')
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

const handle401 = (message) => {
  if (isRedirecting) return
  isRedirecting = true
  clearPlatformSession()
  const currentPath = router.currentRoute.value.path
  if (!currentPath.startsWith('/platform/login')) {
    ElMessage.error(message || t('error.unauthorized'))
    router.replace('/platform/login').catch(() => {
      window.location.href = '/platform/login'
    })
  }
  setTimeout(() => {
    isRedirecting = false
  }, 2000)
}

platformRequest.interceptors.request.use((config) => {
  const token = getPlatformToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token.trim()}`
  }
  const currentLocale = i18n.global.locale.value || Storage.getItem('language', 'zh-CN')
  config.headers['Accept-Language'] =
    currentLocale === 'en-US' ? 'en-US' : 'zh-CN'
  return config
})

platformRequest.interceptors.response.use(
  (response) => {
    if (response.config?.responseType === 'blob' || response.data instanceof Blob) {
      return response.data
    }
    const res = response.data
    const url = response.config?.url || ''
    const isAuthEndpoint = isAuthEndpointUrl(url)

    if (!isRedirecting) {
      const headerToken = response.headers.authorization || response.headers.Authorization
      if (headerToken) {
        const token = String(headerToken).replace(/^Bearer\s+/i, '').trim()
        if (token) setPlatformToken(token)
      }
      if (res?.data?.token) {
        setPlatformToken(res.data.token)
      }
    }

    if (res.code !== 200) {
      const message = res.message || t('error.default')
      const businessCode = Number(res.code)
      if (!isAuthEndpoint) {
        if (businessCode === 401) {
          handle401(message)
        } else {
          ElMessage.error(message)
        }
      }
      const err = new Error(message)
      err.code = res.code
      err.errorCode = res.error_code
      err.data = res.data
      err.translatedMessage = message
      if (!isAuthEndpoint) err.__handled = true
      return Promise.reject(err)
    }
    return res
  },
  (error) => {
    if (error.__handled) return Promise.reject(error)
    const status = error.response?.status
    const data = error.response?.data
    const message = data?.message || error.message || t('error.default')
    const errorCode = data?.error_code || ''
    const url = error.config?.url || ''
    const isAuthEndpoint = isAuthEndpointUrl(url)

    error.errorCode = errorCode
    error.translatedMessage = message
    if (status) error.code = status

    if (status === 401 && !isAuthEndpoint) {
      handle401(message)
      error.__handled = true
    } else if (isAuthEndpoint && status && status >= 400) {
      // Login page handles captcha / 2FA adaptive UI; do not toast here.
      error.__handled = false
    } else if (status && status !== 401) {
      ElMessage.error(message)
      error.__handled = true
    }
    return Promise.reject(error)
  }
)

export default platformRequest
