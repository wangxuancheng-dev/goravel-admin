import { normalizeListResponse } from '@/utils/normalize'
import request from '@/utils/request'
import type { ApiResponse, PaginatedData } from '@/types'

type NotificationListData = PaginatedData & {
  unread_count?: number
  pagination?: unknown
}

export async function getNotificationList(params: Record<string, unknown> = {}) {
  const res = await request({
    url: '/notifications',
    method: 'get',
    params,
  })

  if (!res?.data) return res as ApiResponse<NotificationListData>

  const payload = res.data as {
    notifications?: unknown[]
    pagination?: { total?: number }
    unread_count?: number
  }

  return normalizeListResponse({
    ...res,
    data: {
      list: payload.notifications || [],
      total: payload.pagination?.total || 0,
      unread_count: payload.unread_count || 0,
      pagination: payload.pagination,
    },
  }) as ApiResponse<NotificationListData>
}

export function fetchUnreadCount() {
  return request({ url: '/notifications/unread-count', method: 'get' })
}

export function fetchRecentNotifications(params: Record<string, unknown> = {}) {
  return request({ url: '/notifications/recent', method: 'get', params })
}

export function markNotificationRead(id: string | number) {
  return request({ url: `/notifications/${id}/read`, method: 'post' })
}

export function markAllNotificationsRead() {
  return request({ url: '/notifications/read-all', method: 'post' })
}

export function createNotificationWsTicket() {
  return request({ url: '/notifications/ws-ticket', method: 'post' })
}

export function postNotificationWsClientLog(data: Record<string, unknown>) {
  return request({ url: '/notifications/ws-client-log', method: 'post', data })
}

export function createNotification(data: Record<string, unknown>) {
  return request({ url: '/notifications', method: 'post', data })
}
