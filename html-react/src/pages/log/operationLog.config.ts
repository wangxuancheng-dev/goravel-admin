import type { SearchField } from '@/components/SearchForm'
import { getSevenDaysAgo } from '@/utils/dateUtils'
import type { TFunction } from 'i18next'

export interface OperationLogRow {
  id: number | string
  admin?: { username?: string } | null
  method?: string
  path?: string
  title?: string
  ip?: string
  status_code?: number
  created_at?: string
  params?: unknown
  request?: unknown
  response?: unknown
  changes?: Array<{ field?: string; old?: unknown; new?: unknown }> | null
  name?: string
  /** Platform account (username) that acted via "sign in as tenant admin"; empty = the admin. */
  impersonator?: string
}

export interface OperationLogChange {
  field?: string
  old?: unknown
  new?: unknown
}

export function createOperationLogInitialSearchForm() {
  return {
    username: '',
    method: '',
    path: '',
    title: '',
    ip: '',
    status: '',
    request: '',
    impersonated: '',
    start_time: getSevenDaysAgo(),
    end_time: '',
  }
}

export function transformOperationLogRow(log: Record<string, unknown>): OperationLogRow {
  let params: unknown = null
  try {
    if (log.request) {
      params = typeof log.request === 'string' ? JSON.parse(log.request) : log.request
    } else if (log.params) {
      params = typeof log.params === 'string' ? JSON.parse(String(log.params)) : log.params
    }
  } catch {
    params = log.request || log.params || null
  }

  let changes: OperationLogChange[] | null = null
  try {
    const raw = log.changes
    if (raw) {
      changes = typeof raw === 'string' ? JSON.parse(raw) : (raw as OperationLogChange[])
    }
  } catch {
    changes = null
  }

  const adminRaw = log.admin as Record<string, unknown> | undefined

  return {
    id: log.id as number | string,
    admin: adminRaw ? { username: String(adminRaw.username || '') } : null,
    method: String(log.method || ''),
    path: String(log.path || ''),
    title: String(log.title || ''),
    ip: String(log.ip || ''),
    status_code: Number(log.status_code ?? log.status ?? 0),
    created_at: String(log.created_at || ''),
    params,
    request: log.request || null,
    response: log.response || null,
    changes,
    name: String(log.title || ''),
    impersonator: impersonatorName(log.impersonator),
  }
}

/** Stored as "<platform_username>#<platform_admin_id>"; show the username. */
export function impersonatorName(raw: unknown): string {
  const value = String(raw || '').trim()
  if (!value) return ''
  const hash = value.lastIndexOf('#')
  return hash > 0 ? value.slice(0, hash) : value
}

export function hasRequestParams(request: unknown): boolean {
  if (!request) return false
  try {
    const parsed = typeof request === 'string' ? JSON.parse(request) : request
    if (typeof parsed === 'object' && parsed !== null) {
      return Object.keys(parsed as object).length > 0
    }
    return String(parsed).trim().length > 0
  } catch {
    return String(request).trim().length > 0
  }
}

export function getRequestPreview(request: unknown): string {
  if (!request) return '-'
  try {
    const parsed = typeof request === 'string' ? JSON.parse(request) : request
    if (typeof parsed === 'object' && parsed !== null) {
      const keys = Object.keys(parsed as object)
      if (keys.length === 0) return '-'
      const previewKeys = keys.slice(0, 3)
      const preview = previewKeys
        .map((key) => {
          const value = (parsed as Record<string, unknown>)[key]
          if (typeof value === 'object' && value !== null) {
            return `${key}: {...}`
          }
          const valueStr = String(value)
          return `${key}: ${valueStr.length > 20 ? `${valueStr.substring(0, 20)}...` : valueStr}`
        })
        .join(', ')
      return keys.length > 3 ? `${preview} ... (${keys.length} fields)` : preview
    }
    const str = String(parsed)
    return str.length > 50 ? `${str.substring(0, 50)}...` : str
  } catch {
    const str = String(request)
    return str.length > 50 ? `${str.substring(0, 50)}...` : str
  }
}

export function formatRequestParamsFull(request: unknown): string {
  if (!request) return '-'
  try {
    const parsed = typeof request === 'string' ? JSON.parse(request) : request
    if (typeof parsed === 'object' && parsed !== null) {
      return JSON.stringify(parsed, null, 2)
    }
    return String(parsed)
  } catch {
    return String(request)
  }
}

export function formatChangeValue(value: unknown): string {
  if (value === null || value === undefined) return '-'
  if (typeof value === 'object') return JSON.stringify(value, null, 2)
  return String(value)
}

export function getMethodOptions() {
  return [
    { label: 'POST', value: 'POST' },
    { label: 'PUT', value: 'PUT' },
    { label: 'DELETE', value: 'DELETE' },
    { label: 'PATCH', value: 'PATCH' },
  ]
}

export function createOperationLogSearchFields(
  t: TFunction,
  titleOptions: string[],
  translateTitle: (title: string) => string,
): SearchField[] {
  const titleSelectOptions = [
    { label: t('common.all', { defaultValue: '全部' }), value: '' },
    ...titleOptions.map((title) => ({
      label: translateTitle(title),
      value: title,
    })),
  ]

  return [
    { name: 'username', label: t('log.username'), span: 6 },
    {
      name: 'method',
      label: t('log.method'),
      type: 'select',
      span: 6,
      options: getMethodOptions(),
    },
    { name: 'path', label: t('log.path'), span: 6 },
    {
      name: 'title',
      label: t('log.title'),
      type: 'select',
      span: 6,
      options: titleSelectOptions,
    },
    { name: 'ip', label: t('log.ip'), span: 6, advanced: true },
    {
      name: 'status',
      label: t('log.status'),
      type: 'select',
      span: 6,
      advanced: true,
      options: [
        { label: t('log.success'), value: '1' },
        { label: t('log.failed'), value: '0' },
      ],
    },
    { name: 'request', label: t('log.request_params'), span: 6, advanced: true },
    {
      name: 'impersonated',
      label: t('log.impersonated'),
      type: 'select',
      span: 6,
      advanced: true,
      options: [
        { label: t('log.impersonated_only'), value: '1' },
        { label: t('log.own_only'), value: '0' },
      ],
    },
    { name: 'start_time', label: t('log.start_time'), type: 'datetime', span: 6, advanced: true },
    { name: 'end_time', label: t('log.end_time'), type: 'datetime', span: 6, advanced: true },
  ]
}

export function extractLogDetail<T>(data: Record<string, unknown> | undefined | null, keys: string[]): T | null {
  if (!data) return null
  for (const key of keys) {
    const value = data[key]
    if (value && typeof value === 'object') {
      return value as T
    }
  }
  return data as T
}
