import type { TFunction } from 'i18next'
import { getSevenDaysAgo } from '@/utils/dateUtils'
import { entityField } from '@/utils/normalize'

export type PaymentStatus = 'pending' | 'paid' | 'failed' | 'cancelled'

export interface PaymentSearchForm {
  payment_no: string
  order_no: string
  payment_method_id: string | number
  user_id: string
  status: string
  time_field: string
  start_time: string
  end_time: string
  [key: string]: string | number
}

export function createPaymentInitialSearchForm(): PaymentSearchForm {
  return {
    payment_no: '',
    order_no: '',
    payment_method_id: '',
    user_id: '',
    status: '',
    time_field: 'created_at',
    start_time: getSevenDaysAgo(),
    end_time: '',
  }
}

export function getPaymentStatusColor(status?: string): 'warning' | 'success' | 'error' | 'default' {
  const map: Record<string, 'warning' | 'success' | 'error' | 'default'> = {
    pending: 'warning',
    paid: 'success',
    failed: 'error',
    cancelled: 'default',
  }
  return map[status || ''] || 'default'
}

export function getPaymentStatusText(t: TFunction, status?: string): string {
  const map: Record<string, string> = {
    pending: t('payment.status_pending'),
    paid: t('payment.status_paid'),
    failed: t('payment.status_failed'),
    cancelled: t('payment.status_cancelled'),
  }
  return map[status || ''] || String(status ?? '-')
}

export function getPaymentMethodName(paymentMethod: unknown): string {
  if (!paymentMethod) return '-'
  if (typeof paymentMethod === 'string') return paymentMethod
  if (typeof paymentMethod === 'object') {
    const record = paymentMethod as Record<string, unknown>
    return (
      String(entityField(record, 'name', '') || entityField(record, 'code', '') || '') || '-'
    )
  }
  return '-'
}

export function formatPaymentAmount(amount: unknown): string {
  if (amount === null || amount === undefined) return '0.00'
  return Number(amount).toFixed(2)
}

export function formatPaymentDateTime(dateTime: unknown): string {
  if (!dateTime) return '-'
  if (typeof dateTime === 'string') {
    return dateTime.replace('T', ' ').substring(0, 19)
  }
  return String(dateTime)
}

export function createPaymentStatusOptions(t: TFunction) {
  return [
    { label: t('payment.status_pending'), value: 'pending' },
    { label: t('payment.status_paid'), value: 'paid' },
    { label: t('payment.status_failed'), value: 'failed' },
    { label: t('payment.status_cancelled'), value: 'cancelled' },
  ]
}
