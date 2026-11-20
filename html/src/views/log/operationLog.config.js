import { getSevenDaysAgo } from '@/utils/dateUtils'

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
    end_time: ''
  }
}

export function transformOperationLogRow(log) {
  let params = null
  try {
    if (log.request) {
      params = typeof log.request === 'string' ? JSON.parse(log.request) : log.request
    } else if (log.params) {
      params = typeof log.params === 'string' ? JSON.parse(log.params) : log.params
    }
  } catch {
    params = log.request || log.params || null
  }

  let changes = null
  try {
    const raw = log.changes
    if (raw) {
      changes = typeof raw === 'string' ? JSON.parse(raw) : raw
    }
  } catch {
    changes = null
  }

  return {
    id: log.id,
    admin: log.admin ? { username: log.admin.username || '' } : null,
    method: log.method || '',
    path: log.path || '',
    title: log.title || '',
    ip: log.ip || '',
    status_code: log.status_code ?? log.status ?? 0,
    created_at: log.created_at || '',
    params,
    request: log.request || null,
    response: log.response || null,
    changes,
    impersonator: impersonatorName(log.impersonator)
  }
}

/** Stored as "<platform_username>#<platform_admin_id>"; show the username. */
export function impersonatorName(raw) {
  const value = String(raw || '').trim()
  if (!value) return ''
  const hash = value.lastIndexOf('#')
  return hash > 0 ? value.slice(0, hash) : value
}

export function hasRequestParams(request) {
  if (!request) return false
  try {
    const parsed = typeof request === 'string' ? JSON.parse(request) : request
    if (typeof parsed === 'object' && parsed !== null) {
      return Object.keys(parsed).length > 0
    }
    return String(parsed).trim().length > 0
  } catch {
    return String(request).trim().length > 0
  }
}

export function getRequestPreview(request) {
  if (!request) return '-'
  try {
    const parsed = typeof request === 'string' ? JSON.parse(request) : request
    if (typeof parsed === 'object' && parsed !== null) {
      const keys = Object.keys(parsed)
      if (keys.length === 0) return '-'
      const previewKeys = keys.slice(0, 3)
      const preview = previewKeys.map((key) => {
        const value = parsed[key]
        if (typeof value === 'object' && value !== null) {
          return `${key}: {...}`
        }
        const valueStr = String(value)
        return `${key}: ${valueStr.length > 20 ? `${valueStr.substring(0, 20)}...` : valueStr}`
      }).join(', ')
      return keys.length > 3 ? `${preview} ... (${keys.length} fields)` : preview
    }
    const str = String(parsed)
    return str.length > 50 ? `${str.substring(0, 50)}...` : str
  } catch {
    const str = String(request)
    return str.length > 50 ? `${str.substring(0, 50)}...` : str
  }
}

export function formatRequestParamsFull(request) {
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

export function formatChangeValue(value) {
  if (value === null || value === undefined) return '-'
  if (typeof value === 'object') return JSON.stringify(value, null, 2)
  return String(value)
}

export function createOperationLogSearchFields(t, getMethodOptions, titleOptions, getOperationTitleFn) {
  const titleSelectOptions = [
    { label: t('common.all'), value: '' },
    ...titleOptions.map((title) => ({
      label: getOperationTitleFn(title),
      value: title
    }))
  ]

  return [
    { prop: 'username', label: t('log.username'), type: 'input', width: '200px', advanced: false },
    {
      prop: 'method',
      label: t('log.method'),
      type: 'select',
      width: '150px',
      options: getMethodOptions().filter((opt) => String(opt.value).toUpperCase() !== 'GET'),
      advanced: false
    },
    { prop: 'path', label: t('log.path'), type: 'input', width: '200px', advanced: false },
    {
      prop: 'title',
      label: t('log.title'),
      type: 'select',
      width: '200px',
      options: titleSelectOptions,
      filterable: true,
      clearable: true,
      advanced: false
    },
    { prop: 'ip', label: t('log.ip'), type: 'input', width: '150px', advanced: true },
    {
      prop: 'status',
      label: t('log.status'),
      type: 'select',
      width: '150px',
      options: [
        { label: t('log.success'), value: '1' },
        { label: t('log.failed'), value: '0' }
      ],
      clearable: true,
      advanced: true
    },
    { prop: 'request', label: t('log.request_params'), type: 'input', width: '200px', advanced: true },
    {
      prop: 'impersonated',
      label: t('log.impersonated'),
      type: 'select',
      width: '150px',
      options: [
        { label: t('log.impersonated_only'), value: '1' },
        { label: t('log.own_only'), value: '0' }
      ],
      clearable: true,
      advanced: true
    },
    {
      prop: 'start_time',
      label: t('log.start_time'),
      type: 'datetime',
      width: '180px',
      valueFormat: 'YYYY-MM-DD HH:mm:ss',
      advanced: true
    },
    {
      prop: 'end_time',
      label: t('log.end_time'),
      type: 'datetime',
      width: '180px',
      valueFormat: 'YYYY-MM-DD HH:mm:ss',
      advanced: true
    }
  ]
}

export function createOperationLogTableColumns(t) {
  return [
    { field: 'id', title: t('table.id'), width: 80, sortable: true, key: 'id' },
    { field: 'admin', title: t('log.admin'), slot: 'admin', sortable: false, key: 'admin', width: 140 },
    { field: 'title', title: t('log.title'), slot: 'title', sortable: false, width: 200, key: 'title' },
    { field: 'method', title: t('log.method'), width: 100, sortable: false, key: 'method' },
    { field: 'path', title: t('log.path'), sortable: false, key: 'path' },
    { field: 'ip', title: t('log.ip'), width: 150, sortable: false, key: 'ip' },
    {
      field: 'request',
      title: t('log.request_params'),
      slot: 'request',
      sortable: false,
      width: 250,
      showOverflow: 'tooltip',
      key: 'request'
    },
    { field: 'created_at', title: t('log.operation_time'), width: 180, sortable: true, key: 'created_at' },
    { title: t('table.operation'), width: 150, fixed: 'right', slot: 'operation', key: 'operation' }
  ]
}
