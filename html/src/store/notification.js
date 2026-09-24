import { defineStore } from 'pinia'
import { ElMessage } from 'element-plus'
import i18n from '../i18n'
import Storage from '../utils/storage'
import {
  fetchNotifications,
  fetchUnreadCount,
  fetchRecentNotifications,
  createNotificationWsTicket,
  markNotificationRead,
  markAllNotificationsRead
} from '../api/notification'
import { playNotificationSound } from '../utils/sound'

const { t } = i18n.global

export const useNotificationStore = defineStore('notification', {
  state: () => ({
    items: [],
    unreadCount: 0,
    loading: false,
    ws: null,
    wsConnected: false,
    lastWsCloseCode: null,
    initializing: false,
    retryCount: 0,
    retryTimer: null,
    wsAuthErrorNotified: false,
    soundDebounceTimer: null // 声音防抖定时器
  }),
  actions: {
    async init() {
      if (this.initializing) {
        return
      }
      this.initializing = true
      await this.refresh()
      this.connect()
    },
    async refresh(params = {}) {
      this.loading = true
      try {
        const { data } = await fetchRecentNotifications({
          limit: params.limit || 7
        })
        this.items = data.notifications || []
        this.unreadCount = data.unread_count || 0
      } catch (error) {
        console.error('Load notifications error:', error)
      } finally {
        this.loading = false
      }
    },
    async fetchUnread() {
      try {
        const { data } = await fetchUnreadCount()
        this.unreadCount = data.count || 0
      } catch (error) {
        console.error('Fetch unread count error:', error)
      }
    },
    async markAsRead(id) {
      try {
        await markNotificationRead(id)
        this.items = this.items.map(item =>
          item.id === id ? { ...item, is_read: true, read_at: new Date().toISOString() } : item
        )
        if (this.unreadCount > 0) {
          this.unreadCount -= 1
        }
      } catch (error) {
        console.error('Mark notification read failed:', error)
      }
    },
    async markAllRead() {
      try {
        await markAllNotificationsRead()
        this.items = this.items.map(item => ({ ...item, is_read: true, read_at: new Date().toISOString() }))
        this.unreadCount = 0
      } catch (error) {
        console.error('Mark all notifications read failed:', error)
      }
    },
    async connect() {
      if (this.ws || this.wsConnected) {
        return
      }
      const token = Storage.getItem('token', '')
      if (!token || typeof token !== 'string') {
        return
      }
      const authQuery = await this.resolveWsAuthQuery()
      if (!authQuery) {
        return
      }
      
      // 构建 WebSocket URL
      // 优先使用 VITE_WS_BASE_URL（单独的 WebSocket 域名）
      // 如果没有配置，则使用 VITE_API_BASE_URL
      let wsUrl
      const wsBaseURL = import.meta.env.VITE_WS_BASE_URL
      const apiBaseURL = import.meta.env.VITE_API_BASE_URL
      
      if (wsBaseURL) {
        const base = wsBaseURL.replace(/\/+$/, '')
        if (base.startsWith('wss://') || base.startsWith('ws://')) {
          wsUrl = `${base}/ws/admin/notifications?${authQuery}`
        } else if (base.startsWith('https://')) {
          wsUrl = base.replace('https://', 'wss://') + `/ws/admin/notifications?${authQuery}`
        } else if (base.startsWith('http://')) {
          wsUrl = base.replace('http://', 'ws://') + `/ws/admin/notifications?${authQuery}`
        } else {
          const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
          wsUrl = `${protocol}//${base}/ws/admin/notifications?${authQuery}`
        }
      } else if (apiBaseURL) {
        const base = apiBaseURL.replace(/\/+$/, '')
        if (base.startsWith('https://')) {
          wsUrl = base.replace('https://', 'wss://') + `/ws/admin/notifications?${authQuery}`
        } else if (base.startsWith('http://')) {
          wsUrl = base.replace('http://', 'ws://') + `/ws/admin/notifications?${authQuery}`
        } else {
          const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
          wsUrl = `${protocol}//${base}/ws/admin/notifications?${authQuery}`
        }
      } else {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
        const host = window.location.host
        wsUrl = `${protocol}//${host}/ws/admin/notifications?${authQuery}`
      }

      // HTTPS pages cannot open ws:// (mixed content); force wss.
      if (window.location.protocol === 'https:' && typeof wsUrl === 'string' && wsUrl.startsWith('ws://')) {
        wsUrl = 'wss://' + wsUrl.slice('ws://'.length)
      }

      this.ws = new WebSocket(wsUrl)
      this.ws.onopen = () => {
        this.wsConnected = true
        this.retryCount = 0
        this.lastWsCloseCode = null
        this.wsAuthErrorNotified = false
      }
      this.ws.onmessage = (event) => {
        try {
          const payload = JSON.parse(event.data)
          this.handleIncoming(payload)
        } catch (error) {
          console.error('Invalid notification payload:', error)
        }
      }
      this.ws.onclose = (event) => {
        this.wsConnected = false
        this.ws = null
        this.lastWsCloseCode = event?.code || null
        this.scheduleReconnect()
      }
      this.ws.onerror = () => {
        this.wsConnected = false
      }
    },
    async resolveWsAuthQuery() {
      try {
        const { data } = await createNotificationWsTicket()
        if (data?.ticket) {
          return `ticket=${encodeURIComponent(data.ticket)}`
        }
      } catch (error) {
        console.warn('Create notification ws ticket failed:', error)
      }
      return ''
    },
    scheduleReconnect() {
      if (!this.retryCount) {
        this.retryCount = 0
      }
      if (this.retryTimer) {
        clearTimeout(this.retryTimer)
      }
      const delay = Math.min(30000, 2000 * Math.pow(2, this.retryCount))
      this.retryTimer = setTimeout(() => {
        this.retryCount += 1
        const token = Storage.getItem('token', '')
        if (!token || typeof token !== 'string') {
          return
        }
        this.connect()
        // Browser WebSocket handshake failure usually closes with code 1006.
        if (!this.wsConnected && this.lastWsCloseCode === 1006 && !this.wsAuthErrorNotified) {
          this.wsAuthErrorNotified = true
          ElMessage.error(t('notification.ws_auth_error'))
          return
        }
        if (!this.wsConnected && this.retryCount === 3) {
          ElMessage.error(t('notification.ws_error'))
        }
      }, delay)
    },
    disconnect() {
      if (this.ws) {
        this.ws.close()
        this.ws = null
      }
      this.wsConnected = false
      this.initializing = false
      this.lastWsCloseCode = null
      this.wsAuthErrorNotified = false
      this.items = []
      this.unreadCount = 0
      if (this.retryTimer) {
        clearTimeout(this.retryTimer)
        this.retryTimer = null
      }
      if (this.soundDebounceTimer) {
        clearTimeout(this.soundDebounceTimer)
        this.soundDebounceTimer = null
      }
      this.retryCount = 0
    },
    handleIncoming(payload) {
      if (payload?.type === 'read_all') {
        const readAt = payload.read_at || new Date().toISOString()
        this.items = this.items.map(item => ({
          ...item,
          is_read: true,
          read_at: item.read_at || readAt
        }))
        this.unreadCount = 0
        return
      }

      const notification = payload
      if (notification?.id == null) {
        return
      }

      const exists = this.items.find(item => item.id === notification.id)
      const isNewNotification = !exists

      if (isNewNotification) {
        this.items.unshift(notification)
        if (!notification.is_read) {
          this.unreadCount += 1
          // 播放提示音（带防抖，1秒内只播放一次）
          this.playNotificationSoundWithDebounce()
        }
        // 增加到 20 条，以支持三个 tab 分别显示
        if (this.items.length > 20) {
          this.items = this.items.slice(0, 20)
        }
      } else {
        const wasUnread = !exists.is_read
        this.items = this.items.map(item => item.id === notification.id ? { ...item, ...notification } : item)
        if (wasUnread && notification.is_read && this.unreadCount > 0) {
          this.unreadCount -= 1
        }
      }
    },
    /**
     * 播放通知提示音（带防抖）
     * 如果1秒内收到多个通知，只播放一次声音
     */
    playNotificationSoundWithDebounce() {
      // 如果已经有待执行的定时器，说明在防抖窗口内，直接返回
      if (this.soundDebounceTimer) {
        return
      }
      
      // 立即播放一次声音
      playNotificationSound()
      
      // 设置防抖定时器，1秒后重置
      this.soundDebounceTimer = setTimeout(() => {
        this.soundDebounceTimer = null
      }, 1000) // 1秒防抖窗口
    }
  }
})

