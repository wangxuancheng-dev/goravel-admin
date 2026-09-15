import Storage from './storage'
import i18n from '../i18n'
import { applyTenantHeader } from './tenant'

/** Headers for browser fetch() to /api/admin (EventSource/fetch cannot use axios interceptors). */
export function buildAdminAuthHeaders(extra = {}) {
  const token = String(Storage.getItem('token', '') || '').trim()
  const currentLocale =
    i18n.global?.locale?.value || Storage.getItem('language', 'zh-CN') || 'zh-CN'
  const headers = {
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    'Accept-Language': currentLocale === 'en-US' ? 'en-US' : 'zh-CN',
    ...extra
  }
  applyTenantHeader(headers)
  return headers
}
