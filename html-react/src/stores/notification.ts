import { create } from 'zustand'
import Storage from '@/utils/storage'
import logger from '@/utils/logger'
import {
  createNotificationWsTicket,
  fetchRecentNotifications,
  markAllNotificationsRead,
  markNotificationRead,
} from '@/api/notification'

export interface NotificationItem {
  id: number | string
  title?: string
  content?: string
  is_read?: boolean
  read_at?: string
  created_at?: string
  type?: string
}

interface NotificationState {
  items: NotificationItem[]
  unreadCount: number
  loading: boolean
  wsConnected: boolean
  init: () => Promise<void>
  refresh: (limit?: number) => Promise<void>
  markAsRead: (id: string | number) => Promise<void>
  markAllRead: () => Promise<void>
  disconnect: () => void
}

let ws: WebSocket | null = null
let retryCount = 0
let retryTimer: ReturnType<typeof setTimeout> | null = null
let initializing = false

function buildWsUrl(authQuery: string): string {
  const wsBaseURL = import.meta.env.VITE_WS_BASE_URL as string | undefined
  const apiBaseURL = import.meta.env.VITE_API_BASE_URL as string | undefined
  const path = `/ws/admin/notifications?${authQuery}`

  const toWs = (base: string) => {
    const cleaned = base.replace(/\/+$/, '')
    let url: string
    if (cleaned.startsWith('wss://') || cleaned.startsWith('ws://')) url = cleaned + path
    else if (cleaned.startsWith('https://')) url = cleaned.replace('https://', 'wss://') + path
    else if (cleaned.startsWith('http://')) url = cleaned.replace('http://', 'ws://') + path
    else {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      url = `${protocol}//${cleaned}${path}`
    }
    // HTTPS pages cannot open ws:// (mixed content); force wss.
    if (window.location.protocol === 'https:' && url.startsWith('ws://')) {
      url = `wss://${url.slice('ws://'.length)}`
    }
    return url
  }

  if (wsBaseURL) return toWs(wsBaseURL)
  if (apiBaseURL) return toWs(apiBaseURL)
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${window.location.host}${path}`
}

export const useNotificationStore = create<NotificationState>((set, get) => ({
  items: [],
  unreadCount: 0,
  loading: false,
  wsConnected: false,

  refresh: async (limit = 7) => {
    set({ loading: true })
    try {
      const res = await fetchRecentNotifications({ limit })
      const data = (res.data || {}) as {
        notifications?: NotificationItem[]
        unread_count?: number
      }
      set({
        items: data.notifications || [],
        unreadCount: data.unread_count || 0,
      })
    } catch (error) {
      logger.error('Load notifications error:', error)
    } finally {
      set({ loading: false })
    }
  },

  markAsRead: async (id) => {
    try {
      await markNotificationRead(id)
      set((state) => ({
        items: state.items.map((item) =>
          item.id === id ? { ...item, is_read: true, read_at: new Date().toISOString() } : item,
        ),
        unreadCount: Math.max(0, state.unreadCount - 1),
      }))
    } catch (error) {
      logger.error('Mark notification read failed:', error)
    }
  },

  markAllRead: async () => {
    try {
      await markAllNotificationsRead()
      set((state) => ({
        items: state.items.map((item) => ({
          ...item,
          is_read: true,
          read_at: new Date().toISOString(),
        })),
        unreadCount: 0,
      }))
    } catch (error) {
      logger.error('Mark all notifications read failed:', error)
    }
  },

  disconnect: () => {
    if (retryTimer) clearTimeout(retryTimer)
    retryTimer = null
    if (ws) {
      ws.close()
      ws = null
    }
    set({ wsConnected: false })
  },

  init: async () => {
    if (initializing) return
    initializing = true
    try {
      await get().refresh()
      await connectWs(set, get)
    } finally {
      initializing = false
    }
  },
}))

async function connectWs(
  set: (partial: Partial<NotificationState>) => void,
  get: () => NotificationState,
) {
  if (ws) return
  const token = Storage.getItem<string>('token', '')
  if (!token) return

  let authQuery = ''
  try {
    const res = await createNotificationWsTicket()
    const ticket = (res.data as { ticket?: string } | undefined)?.ticket
    if (ticket) authQuery = `ticket=${encodeURIComponent(ticket)}`
  } catch (error) {
    logger.warn('Create notification ws ticket failed:', error)
    return
  }
  if (!authQuery) return

  const url = buildWsUrl(authQuery)
  ws = new WebSocket(url)
  ws.onopen = () => {
    set({ wsConnected: true })
    retryCount = 0
  }
  ws.onmessage = (event) => {
    try {
      const payload = JSON.parse(event.data as string) as NotificationItem & {
        type?: string
        notification?: NotificationItem
        data?: NotificationItem
        read_at?: string
      }

      if (payload.type === 'read_all') {
        const readAt = payload.read_at || new Date().toISOString()
        set({
          items: get().items.map((item) => ({
            ...item,
            is_read: true,
            read_at: item.read_at || readAt,
          })),
          unreadCount: 0,
        })
        return
      }

      const notification =
        payload.notification ||
        payload.data ||
        (payload.id != null ? (payload as NotificationItem) : null)

      if (notification?.id == null) {
        if (payload.type === 'refresh') {
          void get().refresh()
        }
        return
      }

      const existing = get().items.find((item) => item.id === notification.id)
      if (existing) {
        const wasUnread = !existing.is_read
        set({
          items: get().items.map((item) =>
            item.id === notification.id ? { ...item, ...notification } : item,
          ),
          unreadCount:
            wasUnread && notification.is_read
              ? Math.max(0, get().unreadCount - 1)
              : get().unreadCount,
        })
      } else {
        set({
          items: [notification, ...get().items].slice(0, 20),
          unreadCount: get().unreadCount + (notification.is_read ? 0 : 1),
        })
      }
    } catch (error) {
      logger.error('Invalid notification payload:', error)
    }
  }
  ws.onclose = () => {
    ws = null
    set({ wsConnected: false })
    if (retryCount > 8) return
    const delay = Math.min(1000 * 2 ** retryCount, 30000)
    retryCount += 1
    retryTimer = setTimeout(() => {
      void connectWs(set, get)
    }, delay)
  }
  ws.onerror = () => {
    set({ wsConnected: false })
  }
}
