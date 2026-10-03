import dayjs from 'dayjs'
import { getSevenDaysAgo } from '@/utils/dateUtils'
import { entityField } from '@/utils/normalize'

export interface OrderSearchForm {
  user_id: string
  order_no: string
  status: string
  min_amount: string | number | null
  max_amount: string | number | null
  start_time: string
  end_time: string
  [key: string]: unknown
}

export function createOrderInitialSearchForm(): OrderSearchForm {
  return {
    user_id: '',
    order_no: '',
    status: '',
    min_amount: null,
    max_amount: null,
    start_time: getSevenDaysAgo(),
    end_time: '',
  }
}

export function formatOrderAmount(amount: unknown): string {
  if (amount === null || amount === undefined || amount === '') return '-'
  const n = Number(amount)
  if (Number.isNaN(n)) return '-'
  return `¥${n.toFixed(2)}`
}

export function formatOrderTime(time: unknown): string {
  if (!time) return '-'
  if (typeof time === 'string') return time
  return dayjs(time as string | number | Date).format('YYYY-MM-DD HH:mm:ss')
}

export function getOrderStatusText(t: (key: string) => string, status?: string | null): string {
  const statusMap: Record<string, string> = {
    pending: t('order.status_pending'),
    paid: t('order.status_paid'),
    cancelled: t('order.status_cancelled'),
  }
  return (status && statusMap[status]) || status || '-'
}

export function getOrderStatusTagColor(status?: string | null): string {
  const typeMap: Record<string, string> = {
    pending: 'warning',
    paid: 'success',
    cancelled: 'error',
  }
  return (status && typeMap[status]) || 'default'
}

export function getOrderDetailField(
  order: Record<string, unknown> | null | undefined,
  field: string,
  defaultValue: unknown = '-',
): unknown {
  return entityField(order, field, defaultValue as never) ?? defaultValue
}

export function getOrderDetails(row: Record<string, unknown> | null | undefined): Record<string, unknown>[] {
  if (!row) return []
  const details = row.details ?? row.Details
  return Array.isArray(details) ? (details as Record<string, unknown>[]) : []
}

export const ORDER_STATUS_OPTIONS = [
  { value: 'pending', labelKey: 'order.status_pending' },
  { value: 'paid', labelKey: 'order.status_paid' },
  { value: 'cancelled', labelKey: 'order.status_cancelled' },
] as const
