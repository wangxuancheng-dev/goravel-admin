import { create } from 'zustand'
import Storage from '@/utils/storage'
import logger from '@/utils/logger'
import {
  createNotificationWsTicket,
  fetchRecentNotifications,
  markAllNotificationsRead,
  markNotificationRead,
  postNotificationWsClientLog,
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
let pollTimer: ReturnType<typeof setInterval> | null = null
let initializing = false
let connecting = false

function etldPlusOne(host: string): string {
  const parts = String(host || '')
    .toLowerCase()
    .replace(/:\d+$/, '')
    .split('.')
    .filter(Boolean)
  if (parts.length <= 2) return parts.join('.')
  return parts.slice(-2).join('.')
}

function resolveConfiguredApiHost(): string {
  const raw = String(
    import.meta.env.VITE_WS_BASE_URL || import.meta.env.VITE_API_BASE_URL || '',
  ).trim()
  if (!raw) return ''
  try {
    const withScheme = /^https?:\/\//i.test(raw) || /^wss?:\/\//i.test(raw) ? raw : `https://${raw}`
    return new URL(withScheme).hostname.toLowerCase()
  } catch {
    return ''
  }
}

/** Page on vanity/custom apex (.xyz) talking to api on another apex (.top) is cross-site. */
function isCrossSiteApiHost(): boolean {
  const apiHost = resolveConfiguredApiHost()
  if (!apiHost || typeof window === 'undefined') return false
  return etldPlusOne(apiHost) !== etldPlusOne(window.location.hostname)
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

/** Last-resort only when same-origin WS proxy is missing. */
function startPolling(
  set: (partial: Partial<NotificationState>) => void,
  get: () => NotificationState,
) {
  stopPolling()
  set({ wsConnected: false })
  void get().refresh()
  pollTimer = setInterval(() => {
    void get().refresh()
  }, 15000)
}

function buildWsUrl(authQuery: string): string {
  const path = `/ws/admin/notifications?${authQuery}`
  const pageProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'

  // Split deploy: vanity on .xyz + API on api.*.top is cross-site; browser WSS never
  // reaches origin (see debug logs: ticket_issued without ws_attempt). Use same-origin
  // wss://<page-host>/ws/... and proxy /ws on the vanity CDN/Worker to the API.
  if (isCrossSiteApiHost()) {
    return `${pageProtocol}//${window.location.host}${path}`
  }

  const wsBaseURL = import.meta.env.VITE_WS_BASE_URL as string | undefined
  const apiBaseURL = import.meta.env.VITE_API_BASE_URL as string | undefined

  const toWs = (base: string) => {
    const cleaned = base.replace(/\/+$/, '')
    let url: string
    if (cleaned.startsWith('wss://') || cleaned.startsWith('ws://')) url = cleaned + path
    else if (cleaned.startsWith('https://')) url = cleaned.replace('https://', 'wss://') + path
    else if (cleaned.startsWith('http://')) url = cleaned.replace('http://', 'ws://') + path
    else {
      url = `${pageProtocol}//${cleaned}${path}`
    }
    if (window.location.protocol === 'https:' && url.startsWith('ws://')) {
      url = `wss://${url.slice('ws://'.length)}`
    }
    return url
  }

  if (wsBaseURL) return toWs(wsBaseURL)
  if (apiBaseURL) return toWs(apiBaseURL)
  return `${pageProtocol}//${window.location.host}${path}`
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
    stopPolling()
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
  if (ws || connecting || pollTimer) return
  const token = Storage.getItem<string>('token', '')
  if (!token) return

  const crossSite = isCrossSiteApiHost()
  connecting = true
  let authQuery = ''
  let ticketLen = 0
  try {
    const res = await createNotificationWsTicket()
    const ticket = (res.data as { ticket?: string } | undefined)?.ticket
    if (ticket) {
      ticketLen = ticket.length
      authQuery = `ticket=${encodeURIComponent(ticket)}`
    }
  } catch (error) {
    logger.warn('Create notification ws ticket failed:', error)
    connecting = false
    // #region agent log
    void postNotificationWsClientLog({
      hypothesisId: 'A',
      message: 'ticket_http_failed',
      data: { page_host: window.location.host, err: String(error) },
    }).catch(() => {})
    fetch('http://127.0.0.1:7270/ingest/65377cb3-0af9-4a20-9b22-07e57672183c',{method:'POST',headers:{'Content-Type':'application/json','X-Debug-Session-Id':'79bec9'},body:JSON.stringify({sessionId:'79bec9',location:'notification.ts:ticket',message:'ticket_http_failed',data:{host:window.location.host},timestamp:Date.now(),hypothesisId:'A'})}).catch(()=>{})
    // #endregion
    return
  }
  if (!authQuery) {
    connecting = false
    return
  }

  const url = buildWsUrl(authQuery)
  // #region agent log
  void postNotificationWsClientLog({
    hypothesisId: 'A',
    message: 'ws_connecting',
    data: {
      page_host: window.location.host,
      page_origin: window.location.origin,
      cross_site: crossSite,
      same_origin_ws: crossSite,
      url_len: url.length,
      ticket_len: ticketLen,
      ws_host: (() => { try { return new URL(url).host } catch { return '' } })(),
    },
  }).catch(() => {})
  fetch('http://127.0.0.1:7270/ingest/65377cb3-0af9-4a20-9b22-07e57672183c',{method:'POST',headers:{'Content-Type':'application/json','X-Debug-Session-Id':'79bec9'},body:JSON.stringify({sessionId:'79bec9',location:'notification.ts:connect',message:'ws_connecting',data:{pageHost:window.location.host,crossSite,urlLen:url.length,ticketLen,wsHost:(()=>{try{return new URL(url).host}catch{return ''}})()},timestamp:Date.now(),hypothesisId:'A'})}).catch(()=>{})
  // #endregion
  try {
    ws = new WebSocket(url)
  } catch (error) {
    connecting = false
    logger.warn('WebSocket construct failed:', error)
    if (crossSite) startPolling(set, get)
    return
  }
  connecting = false
  ws.onopen = () => {
    set({ wsConnected: true })
    retryCount = 0
    stopPolling()
    // #region agent log
    void postNotificationWsClientLog({
      hypothesisId: 'E',
      message: 'ws_open',
      data: { page_host: window.location.host, cross_site: crossSite, ws_host: (() => { try { return new URL(url).host } catch { return '' } })() },
    }).catch(() => {})
    fetch('http://127.0.0.1:7270/ingest/65377cb3-0af9-4a20-9b22-07e57672183c',{method:'POST',headers:{'Content-Type':'application/json','X-Debug-Session-Id':'79bec9'},body:JSON.stringify({sessionId:'79bec9',location:'notification.ts:open',message:'ws_open',data:{host:window.location.host,crossSite},timestamp:Date.now(),hypothesisId:'E'})}).catch(()=>{})
    // #endregion
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
  ws.onclose = (event) => {
    ws = null
    set({ wsConnected: false })
    // #region agent log
    void postNotificationWsClientLog({
      hypothesisId: 'A',
      message: 'ws_close',
      data: {
        page_host: window.location.host,
        code: event?.code,
        reason: event?.reason || '',
        was_clean: event?.wasClean,
        ticket_len: ticketLen,
        url_len: url.length,
      },
    }).catch(() => {})
    fetch('http://127.0.0.1:7270/ingest/65377cb3-0af9-4a20-9b22-07e57672183c',{method:'POST',headers:{'Content-Type':'application/json','X-Debug-Session-Id':'79bec9'},body:JSON.stringify({sessionId:'79bec9',location:'notification.ts:close',message:'ws_close',data:{code:event?.code,wasClean:event?.wasClean,host:window.location.host,ticketLen,urlLen:url.length},timestamp:Date.now(),hypothesisId:'A'})}).catch(()=>{})
    // #endregion
    if (retryCount > 8) {
      if (crossSite) startPolling(set, get)
      return
    }
    const delay = Math.min(1000 * 2 ** retryCount, 30000)
    retryCount += 1
    retryTimer = setTimeout(() => {
      void connectWs(set, get)
    }, delay)
  }
  ws.onerror = () => {
    set({ wsConnected: false })
    // #region agent log
    void postNotificationWsClientLog({
      hypothesisId: 'A',
      message: 'ws_error',
      data: { page_host: window.location.host, ticket_len: ticketLen, url_len: url.length },
    }).catch(() => {})
    fetch('http://127.0.0.1:7270/ingest/65377cb3-0af9-4a20-9b22-07e57672183c',{method:'POST',headers:{'Content-Type':'application/json','X-Debug-Session-Id':'79bec9'},body:JSON.stringify({sessionId:'79bec9',location:'notification.ts:error',message:'ws_error',data:{host:window.location.host,ticketLen,urlLen:url.length},timestamp:Date.now(),hypothesisId:'A'})}).catch(()=>{})
    // #endregion
  }
}
