export const loginLogInitialSearchForm = {
  username: '',
  ip: '',
  status: '',
  start_time: '',
  end_time: ''
}

export function transformLoginLogRow(log) {
  return {
    id: log.id,
    admin: log.admin
      ? { username: log.admin.username || '' }
      : null,
    ip: log.ip || '',
    user_agent: log.user_agent || '',
    location: log.location || '',
    status: log.status || 0,
    message: log.message || '',
    request: log.request || '',
    created_at: log.created_at || ''
  }
}

export function createLoginLogSearchFields(t) {
  return [
    { prop: 'username', label: t('log.username'), type: 'input', width: '200px', advanced: false },
    { prop: 'ip', label: t('log.ip'), type: 'input', width: '150px', advanced: false },
    {
      prop: 'status',
      label: t('log.status'),
      type: 'select',
      width: '120px',
      options: [
        { label: t('log.success'), value: '1' },
        { label: t('log.failed'), value: '0' }
      ],
      advanced: false
    },
    { prop: 'start_time', label: t('log.start_time'), type: 'datetime', width: '180px', valueFormat: 'YYYY-MM-DD HH:mm:ss', advanced: true },
    { prop: 'end_time', label: t('log.end_time'), type: 'datetime', width: '180px', valueFormat: 'YYYY-MM-DD HH:mm:ss', advanced: true }
  ]
}

export function createLoginLogTableColumns(t) {
  return [
    { field: 'id', title: t('table.id'), width: 80, sortable: true, key: 'id' },
    { field: 'admin', title: t('log.admin'), width: 100, slot: 'admin', sortable: false, key: 'admin' },
    { field: 'ip', title: t('log.ip'), width: 150, sortable: false, key: 'ip' },
    { field: 'location', title: t('log.location'), width: 180, sortable: false, key: 'location' },
    { field: 'user_agent', title: t('log.user_agent'), sortable: false, key: 'user_agent' },
    { field: 'status', title: t('table.status'), width: 100, sortable: false, slot: 'status', key: 'status' },
    { field: 'message', title: t('log.message'), width: 100, sortable: false, slot: 'message', key: 'message' },
    { field: 'created_at', title: t('log.login_time'), width: 180, sortable: true, key: 'created_at' },
    { title: t('table.operation'), width: 260, fixed: 'right', slot: 'operation', sortable: false, key: 'operation' }
  ]
}

export function formatRequestPreview(request) {
  if (!request) return '-'
  try {
    return JSON.stringify(JSON.parse(request), null, 2)
  } catch {
    return request
  }
}
