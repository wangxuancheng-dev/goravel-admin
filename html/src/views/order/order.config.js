import { getSevenDaysAgo } from '@/utils/dateUtils'
import { getField } from '@/utils/normalizeFormData'

export function createOrderInitialSearchForm() {
  return {
    user_id: '',
    order_no: '',
    status: '',
    min_amount: null,
    max_amount: null,
    start_time: getSevenDaysAgo(),
    end_time: ''
  }
}

export function createOrderSearchFields(t) {
  return [
    { prop: 'user_id', label: t('order.user_id'), type: 'input', width: '150px', advanced: false },
    { prop: 'order_no', label: t('order.order_no'), type: 'input', width: '200px', advanced: false },
    {
      prop: 'status',
      label: t('order.status'),
      type: 'select',
      width: '150px',
      options: [
        { label: t('order.status_pending'), value: 'pending' },
        { label: t('order.status_paid'), value: 'paid' },
        { label: t('order.status_cancelled'), value: 'cancelled' }
      ],
      advanced: false
    },
    { prop: 'min_amount', label: t('order.min_amount'), type: 'number', width: '150px', advanced: true, min: 0, step: 0.01 },
    { prop: 'max_amount', label: t('order.max_amount'), type: 'number', width: '150px', advanced: true, min: 0, step: 0.01 },
    { prop: 'start_time', label: t('order.start_time'), type: 'datetime', width: '200px', advanced: true },
    { prop: 'end_time', label: t('order.end_time'), type: 'datetime', width: '200px', advanced: true }
  ]
}

export function createOrderTableColumns(t) {
  return [
    { field: 'id', title: t('table.id'), width: 80, sortable: true },
    { field: 'order_no', title: t('order.order_no'), sortable: false },
    { field: 'user_id', title: t('order.user_id'), width: 100, sortable: false },
    { field: 'amount', title: t('order.amount'), width: 120, sortable: true, slot: 'amount' },
    { field: 'status', title: t('order.status'), width: 100, sortable: false, slot: 'status' },
    { field: 'created_at', title: t('order.created_at'), width: 180, sortable: true },
    {
      field: 'expire_at',
      title: t('order.expire_at'),
      width: 180,
      sortable: false,
      formatter: ({ cellValue }) => (cellValue ? formatOrderTime(cellValue) : '-')
    },
    {
      field: 'remark',
      title: t('order.remark'),
      sortable: false,
      width: 160,
      formatter: ({ cellValue }) => cellValue || '-'
    },
    { title: t('table.operation'), width: 220, fixed: 'right', slot: 'operation' }
  ]
}

export function formatOrderAmount(amount) {
  if (amount === null || amount === undefined) return '-'
  return `¥${Number(amount).toFixed(2)}`
}

export function formatOrderTime(time) {
  if (!time) return '-'
  return typeof time === 'string' ? time : new Date(time).toLocaleString('zh-CN')
}

export function getOrderStatusText(t, status) {
  const statusMap = {
    pending: t('order.status_pending'),
    paid: t('order.status_paid'),
    cancelled: t('order.status_cancelled')
  }
  return statusMap[status] || status || '-'
}

export function getOrderStatusTagType(status) {
  const typeMap = {
    pending: 'warning',
    paid: 'success',
    cancelled: 'danger'
  }
  return typeMap[status] || 'info'
}

export function getOrderDetailField(order, field, defaultValue = '-') {
  return getField(order, field, defaultValue)
}

export function getOrderDetails(row) {
  return row.details || row.Details || []
}
