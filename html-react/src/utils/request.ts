import type { AxiosError, InternalAxiosRequestConfig } from 'axios'
import { message as antdMessage } from 'antd'
import i18n from '@/i18n'
import { useUserStore } from '@/stores/user'
import { useTabsStore } from '@/stores/tabs'
import { useAppStore } from '@/stores/app'
import Storage from './storage'
import { navigateTo, getCurrentPath } from './navigation'
import { ERROR_CODES } from '@/types'
import type { ApiError, ApiResponse } from '@/types'
import { http } from './http'
import type { RequestConfig } from './http'
import { applyTenantHeader } from './tenant'
import { isAuthEndpointUrl } from './authEndpoint'

let isRedirecting = false
let last403ErrorTime = 0
let isShowing403Error = false
const FORBIDDEN_ERROR_COOLDOWN = 3000

function translateErrorCode(errorCode: string, fallbackMessage: string): string {
  if (!errorCode) return fallbackMessage

  const looksLikeCode =
    fallbackMessage === errorCode ||
    (fallbackMessage.includes('_') && !/[\u4e00-\u9fa5]/.test(fallbackMessage))

  if (!looksLikeCode) return fallbackMessage || errorCode

  const commonKey = `common.${errorCode}`
  const commonTranslated = i18n.t(commonKey)
  if (commonTranslated && commonTranslated !== commonKey) return commonTranslated

  const messagesKey = `messages.${errorCode}`
  const messagesTranslated = i18n.t(messagesKey)
  if (messagesTranslated && messagesTranslated !== messagesKey) return messagesTranslated

  return fallbackMessage || errorCode
}

function extractErrorInfo(data: ApiResponse | undefined) {
  let msg = data?.message || (data?.data as { message?: string } | undefined)?.message || ''
  const errorCode = data?.error_code || (data?.data as { error_code?: string } | undefined)?.error_code || ''
  const code = data?.code || 0
  msg = translateErrorCode(errorCode, msg)
  return { message: msg, errorCode, code }
}

function handle401Error(msg?: string) {
  if (isRedirecting) return
  isRedirecting = true

  useUserStore.getState().logout(true)
  useTabsStore.getState().removeAllTabs()

  if (getCurrentPath() !== '/login') {
    antdMessage.error(msg || i18n.t('error.unauthorized'))
    navigateTo('/login', { replace: true })
    // Hard fallback: AppRouter rebuilds when menus clear; soft navigate can no-op.
    setTimeout(() => {
      if (getCurrentPath() !== '/login') {
        const base = import.meta.env.BASE_URL || '/'
        window.location.replace(new URL('login', `${window.location.origin}${base}`).href)
      }
    }, 50)
  }

  setTimeout(() => {
    isRedirecting = false
  }, 2000)
}

function handle403Error(msg?: string) {
  const now = Date.now()
  if (isShowing403Error || now - last403ErrorTime < FORBIDDEN_ERROR_COOLDOWN) return

  isShowing403Error = true
  last403ErrorTime = now
  antdMessage.error(msg || i18n.t('error.forbidden'))

  setTimeout(() => {
    isShowing403Error = false
  }, FORBIDDEN_ERROR_COOLDOWN)
}

function resolveAcceptLanguage(): string {
  const currentLocale = i18n.language || Storage.getItem<string>('language', 'zh-CN') || 'zh-CN'
  if (currentLocale === 'en-US' || currentLocale.startsWith('en')) return 'en-US'
  return 'zh-CN'
}

function resolveTimezone(): string {
  let browserTimezone = 'UTC'
  try {
    browserTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'
  } catch {
    browserTimezone = 'UTC'
  }
  return useAppStore.getState().timezone || Storage.getItem<string>('timezone', browserTimezone) || browserTimezone
}

http.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const token = Storage.getItem<string>('token', '') || ''
  if (token) {
    config.headers.Authorization = `Bearer ${String(token).trim()}`
  }
  config.headers['Accept-Language'] = resolveAcceptLanguage()
  const timezone = resolveTimezone()
  if (timezone) {
    config.headers['X-Timezone'] = timezone
  }
  applyTenantHeader(config.headers as Record<string, unknown>)
  return config
})

http.interceptors.response.use(
  (response) => {
    const res = response.data as ApiResponse
    const url = response.config?.url || ''
    const isAuthEndpoint = isAuthEndpointUrl(url)

    // Never restore token after logout/401 redirect has started.
    if (!isRedirecting) {
      const headerToken = response.headers.authorization || response.headers.Authorization
      if (headerToken) {
        const token = String(headerToken).replace(/^Bearer\s+/i, '').trim()
        if (token) {
          Storage.setItem('token', token)
          useUserStore.getState().setToken(token)
        }
      }

      const payloadToken = (res.data as { token?: string } | undefined)?.token
      if (payloadToken) {
        Storage.setItem('token', payloadToken)
        useUserStore.getState().setToken(payloadToken)
      }
    }

    if (res.code !== 200) {
      const { message: msg, errorCode } = extractErrorInfo(res)
      const businessCode = Number(res.code)

      if (!isAuthEndpoint) {
        if (businessCode === 401) {
          handle401Error(msg || i18n.t('error.unauthorized'))
        } else if (businessCode === 403) {
          if (errorCode === 'must_change_password') {
            antdMessage.warning(msg || i18n.t('profile.must_change_password_title'))
            if (!window.location.pathname.includes('/profile')) {
              window.location.replace('/profile?tab=password')
            }
          } else {
            handle403Error(msg || i18n.t('error.forbidden'))
          }
        } else {
          antdMessage.error(msg || i18n.t('error.default'))
        }
      }

      const err = new Error(msg || i18n.t('error.default')) as ApiError
      err.code = res.code
      err.errorCode = errorCode
      err.data = res.data
      err.response = response
      err.message = msg
      err.translatedMessage = msg
      if (!isAuthEndpoint) err.__handled = true
      return Promise.reject(err)
    }

    // Business layer consumes envelope `{ code, message, data }`, not AxiosResponse.
    return res as unknown as typeof response
  },
  (error: AxiosError<ApiResponse> & ApiError) => {
    if (error.__handled) return Promise.reject(error)

    if (error.response) {
      const { status, data, config } = error.response
      const url = config?.url || ''
      const isAuthEndpoint = isAuthEndpointUrl(url)
      const { message: msg, errorCode } = extractErrorInfo(data)
      const skipErrorMessage = !!(config as RequestConfig)?.skipErrorMessage

      if (skipErrorMessage) {
        error.errorCode = errorCode
        error.message = msg || error.message
        error.translatedMessage = msg || error.message
        ;(error as ApiError).code = status
        error.__handled = false
        return Promise.reject(error)
      }

      if (status === 429) {
        const isExportEndpoint = url.includes('/export')
        if (!isExportEndpoint) {
          antdMessage.error(msg || i18n.t('error.tooManyRequests'))
          error.__handled = true
        }
      } else if (status === 401) {
        if (!isAuthEndpoint) {
          handle401Error(msg || i18n.t('error.unauthorized'))
        } else {
          error.errorCode = errorCode
          error.message = msg
          error.translatedMessage = msg
          error.__handled = false
        }
      } else if (status === 403) {
        if (!isAuthEndpoint) {
          if (errorCode === 'must_change_password') {
            antdMessage.warning(msg || i18n.t('profile.must_change_password_title'))
            if (!window.location.pathname.includes('/profile')) {
              window.location.replace('/profile?tab=password')
            }
            error.__handled = true
          } else {
            handle403Error(msg || i18n.t('error.forbidden'))
            error.__handled = true
          }
        } else {
          error.errorCode = errorCode
          error.message = msg
          error.translatedMessage = msg
          error.__handled = false
        }
      } else if (isAuthEndpoint && status >= 400) {
        error.errorCode = errorCode
        error.message = msg
        error.translatedMessage = msg
        ;(error as ApiError).code = status
        error.__handled = false
      } else if (!isAuthEndpoint) {
        antdMessage.error(msg || i18n.t('error.default'))
        error.__handled = true
      }
    } else {
      let errorMessage = i18n.t('error.network')
      if (error.code === 'ERR_NETWORK' || error.message === 'Network Error') {
        errorMessage = `${i18n.t('error.network')} (网络连接失败，请检查 API 地址配置)`
      } else if (error.code === 'ECONNABORTED') {
        errorMessage = i18n.t('error.timeout')
      } else if (error.message) {
        errorMessage = error.message
      }

      if (!(error.config as RequestConfig | undefined)?.silent) {
        antdMessage.error(errorMessage)
        error.__handled = true
      }
    }

    if (typeof error === 'object' && error.__handled !== false && error.__handled !== true) {
      const url = error.config?.url || ''
      if (!isAuthEndpointUrl(url)) error.__handled = true
    }

    return Promise.reject(error)
  },
)

export default http as import('./http').TypedRequest
export { ERROR_CODES }
