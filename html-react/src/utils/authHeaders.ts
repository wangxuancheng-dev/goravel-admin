import Storage from '@/utils/storage'
import i18n from '@/i18n'
import { applyTenantHeader } from '@/utils/tenant'

/** Headers for browser fetch() to /api/admin (EventSource/fetch cannot use axios interceptors). */
export function buildAdminAuthHeaders(extra?: Record<string, string>): Record<string, string> {
  const token = String(Storage.getItem<string>('token', '') ?? '').trim()
  const currentLocale = i18n.language || Storage.getItem<string>('language', 'zh-CN') || 'zh-CN'
  const headers: Record<string, string> = {
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    'Accept-Language': currentLocale === 'en-US' ? 'en-US' : 'zh-CN',
    ...(extra || {}),
  }
  applyTenantHeader(headers)
  return headers
}
