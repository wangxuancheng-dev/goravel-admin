import { getField } from '@/utils/normalizeFormData'

export const exportInitialSearchForm = {
  type: '',
  filename: '',
  disk: '',
  status: '',
  start_time: '',
  end_time: ''
}

export function transformExportRow(item) {
  return {
    id: getField(item, 'id'),
    admin: item.admin || item.Admin || null,
    type: getField(item, 'type', ''),
    filename: getField(item, 'filename', ''),
    disk: getField(item, 'disk', ''),
    path: getField(item, 'path', ''),
    extension: getField(item, 'extension', ''),
    size: getField(item, 'size', 0),
    status: getField(item, 'status', 0),
    error_msg: getField(item, 'error_msg', ''),
    created_at: getField(item, 'created_at', ''),
    file_url: getField(item, 'file_url', '')
  }
}

export function createExportSearchFields(t) {
  return [
    {
      prop: 'type',
      label: t('export.type'),
      type: 'select',
      width: '150px',
      options: [
        { label: t('export.types.orders'), value: 'orders' },
        { label: t('export.types.payments'), value: 'payments' },
        { label: t('export.types.admins'), value: 'admins' },
        { label: t('export.types.users'), value: 'users' },
        { label: t('export.types.articles'), value: 'articles' },
        { label: t('export.types.operation_logs_archive'), value: 'operation_logs_archive' }
      ],
      clearable: true
    },
    { prop: 'filename', label: t('export.filename'), type: 'input', width: '200px' },
    {
      prop: 'disk',
      label: t('export.disk'),
      type: 'select',
      width: '150px',
      options: [
        { label: 'local', value: 'local' },
        { label: 'public', value: 'public' },
        { label: 's3', value: 's3' },
        { label: 'oss', value: 'oss' },
        { label: 'cos', value: 'cos' },
        { label: 'qiniu', value: 'qiniu' },
        { label: 'minio', value: 'minio' }
      ],
      clearable: true
    },
    {
      prop: 'status',
      label: t('log.status'),
      type: 'select',
      width: '150px',
      options: [
        { label: t('log.processing'), value: '0' },
        { label: t('log.success'), value: '1' },
        { label: t('log.failed'), value: '2' }
      ],
      clearable: true
    },
    { prop: 'start_time', label: t('log.start_time'), type: 'datetime', width: '180px', valueFormat: 'YYYY-MM-DD HH:mm:ss', advanced: true },
    { prop: 'end_time', label: t('log.end_time'), type: 'datetime', width: '180px', valueFormat: 'YYYY-MM-DD HH:mm:ss', advanced: true }
  ]
}

export function createExportTableColumns(t, formatters) {
  return [
    { type: 'checkbox', width: 60, key: 'checkbox' },
    { field: 'id', title: t('table.id'), width: 80, sortable: true, key: 'id' },
    { field: 'type', title: t('export.type'), width: 120, formatter: formatters.formatType, key: 'type' },
    { field: 'filename', title: t('export.filename'), minWidth: 200, key: 'filename' },
    { field: 'disk', title: t('export.disk'), width: 120, key: 'disk' },
    { field: 'path', title: t('export.path'), minWidth: 260, key: 'path' },
    { field: 'extension', title: t('export.extension'), width: 100, key: 'extension' },
    { field: 'size', title: t('export.size'), width: 140, formatter: formatters.formatSize, key: 'size' },
    { field: 'status', title: t('log.status'), width: 150, slot: 'status', key: 'status' },
    { field: 'error_msg', title: t('export.error_msg'), minWidth: 200, formatter: formatters.formatErrorMsg, key: 'error_msg' },
    { field: 'admin', title: t('log.admin'), width: 140, formatter: formatters.formatAdmin, key: 'admin' },
    { field: 'created_at', title: t('table.created_at'), width: 180, sortable: true, key: 'created_at' },
    { title: t('table.operation'), width: 200, fixed: 'right', slot: 'operation', key: 'operation' }
  ]
}

export function formatExportType(t, cellValue) {
  const type = cellValue || ''
  if (!type) return '-'
  const key = `export.types.${type}`
  const translated = t(key)
  return translated !== key ? translated : type
}

export function formatExportSize({ cellValue } = {}) {
  const size = Number(cellValue || 0)
  if (!size) return '-'
  if (size < 1024) return `${size} B`
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(2)} KB`
  if (size < 1024 * 1024 * 1024) return `${(size / 1024 / 1024).toFixed(2)} MB`
  return `${(size / 1024 / 1024 / 1024).toFixed(2)} GB`
}

export function formatExportAdmin({ row }) {
  const admin = row.admin
  if (admin) return getField(admin, 'username', '-')
  return '-'
}

export function formatExportErrorMsg({ row }) {
  const errorMsg = row.error_msg || ''
  if (!errorMsg) return '-'
  return Number(row.status) === 2 ? errorMsg : '-'
}

export function formatExportStatus(t, row) {
  const status = Number(row.status ?? 0)
  if (status === 1) return t('log.success')
  if (status === 0) return t('log.processing') || '处理中'
  return t('log.failed')
}

export function getExportStatusTagType(row) {
  const status = Number(row.status ?? 0)
  if (status === 1) return 'success'
  if (status === 0) return 'warning'
  return 'danger'
}

export function isExportCompleted(row) {
  return Number(row.status ?? 0) === 1
}
