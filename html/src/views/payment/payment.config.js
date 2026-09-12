import { getSevenDaysAgo } from '@/utils/dateUtils'
import { getField } from '@/utils/normalizeFormData'

export function createPaymentInitialSearchForm() {
  return {
    payment_no: '',
    order_no: '',
    payment_method_id: '',
    user_id: '',
    status: '',
    time_field: 'created_at',
    start_time: getSevenDaysAgo(),
    end_time: ''
  }
}

export function createPaymentSearchFields(t) {
  return [
    { prop: 'payment_no', type: 'input', placeholder: t('payment.payment_no_placeholder') },
    { prop: 'order_no', type: 'input', placeholder: t('payment.order_no_placeholder') },
    {
      prop: 'payment_method_id',
      type: 'select',
      placeholder: t('payment.payment_method_id_placeholder'),
      apiUrl: '/options?type=payment_method',
      filterable: true
    },
    { prop: 'user_id', type: 'input', placeholder: t('payment.user_id_placeholder') },
    {
      prop: 'status',
      type: 'select',
      placeholder: t('payment.status_placeholder'),
      options: [
        { label: t('payment.status_pending'), value: 'pending' },
        { label: t('payment.status_paid'), value: 'paid' },
        { label: t('payment.status_failed'), value: 'failed' },
        { label: t('payment.status_cancelled'), value: 'cancelled' }
      ]
    },
    {
      prop: 'time_field',
      type: 'select',
      placeholder: t('payment.time_field_placeholder'),
      options: [
        { label: t('payment.time_field_created_at'), value: 'created_at' },
        { label: t('payment.time_field_pay_time'), value: 'pay_time' }
      ]
    },
    { prop: 'start_time', type: 'datetime', placeholder: t('payment.start_time_placeholder') },
    { prop: 'end_time', type: 'datetime', placeholder: t('payment.end_time_placeholder') }
  ]
}

export function createPaymentTableColumns(t) {
  return [
    { field: 'id', title: t('table.id'), width: 80, sortable: true },
    { field: 'payment_no', title: t('payment.payment_no'), minWidth: 300, sortable: true },
    { field: 'order_no', title: t('payment.order_no'), minWidth: 180, sortable: true },
    { field: 'payment_method', title: t('payment.payment_method'), width: 150, slot: 'payment_method' },
    { field: 'user_id', title: t('payment.user_id'), width: 100, sortable: true },
    { field: 'amount', title: t('payment.amount'), width: 120, align: 'right', slot: 'amount', sortable: true },
    { field: 'status', title: t('payment.status'), width: 100, slot: 'status', sortable: true },
    { field: 'pay_time', title: t('payment.pay_time'), width: 180, sortable: true },
    { field: 'created_at', title: t('table.created_at'), width: 180, sortable: true },
    { field: 'operation', title: t('table.operation'), width: 100, fixed: 'right', slot: 'operation', sortable: false }
  ]
}

export function getPaymentStatusType(status) {
  const statusMap = {
    pending: 'warning',
    paid: 'success',
    failed: 'danger',
    cancelled: 'info'
  }
  return statusMap[status] || undefined
}

export function getPaymentStatusText(t, status) {
  const statusMap = {
    pending: t('payment.status_pending'),
    paid: t('payment.status_paid'),
    failed: t('payment.status_failed'),
    cancelled: t('payment.status_cancelled')
  }
  return statusMap[status] || status
}

export function getPaymentMethodName(paymentMethod) {
  if (!paymentMethod) return '-'
  if (typeof paymentMethod === 'string') return paymentMethod
  if (typeof paymentMethod === 'object') {
    return getField(paymentMethod, 'name', '') || getField(paymentMethod, 'code', '') || '-'
  }
  return '-'
}

export function formatPaymentAmount(amount) {
  if (amount === null || amount === undefined) return '0.00'
  return Number(amount).toFixed(2)
}

export function formatPaymentDateTime(dateTime) {
  if (!dateTime) return '-'
  if (typeof dateTime === 'string') {
    return dateTime.replace('T', ' ').substring(0, 19)
  }
  return dateTime
}
