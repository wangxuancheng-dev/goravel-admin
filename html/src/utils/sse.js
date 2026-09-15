/**
 * SSE (Server-Sent Events) 工具函数
 * 用于创建和管理 SSE 连接
 */

import Storage from './storage'
import { getTenantCode, resolveTenantCodeFromLocation } from './tenant'

/**
 * Match applyTenantHeader: send whenever a tenant hint exists, even if VITE_TENANCY_* is unset.
 * @returns {string}
 */
function resolveSSETenantHint() {
  return getTenantCode() || resolveTenantCodeFromLocation()
}

/**
 * EventSource cannot set headers; pass JWT and tenant via query (matches Tenant middleware ClientHint).
 * @param {string} fullURL
 * @param {string} token
 * @returns {string}
 */
export function buildSSEUrl(fullURL, token) {
  const abs = new URL(fullURL, 'http://local.invalid')
  abs.searchParams.set('_token', token.trim())
  const code = resolveSSETenantHint()
  if (code && !abs.searchParams.get('tenant_code') && !abs.searchParams.get('tenant_id')) {
    // Backend ClientHint reads tenant_id then tenant_code; set both for SSE (no custom headers).
    abs.searchParams.set('tenant_code', code)
    abs.searchParams.set('tenant_id', code)
  }
  if (fullURL.startsWith('http')) {
    return abs.toString()
  }
  return `${abs.pathname}${abs.search}${abs.hash}`
}

/**
 * 创建 SSE 连接
 * @param {string} url - SSE 端点 URL（相对路径，会自动添加 baseURL）
 * @param {Object} options - 配置选项
 * @param {Function} options.onMessage - 消息回调函数
 * @param {Function} options.onError - 错误回调函数
 * @param {Function} options.onOpen - 连接打开回调函数
 * @param {Function} options.onClose - 连接关闭回调函数
 * @returns {EventSource} EventSource 实例
 */
export function createSSEConnection(url, options = {}) {
  const {
    onMessage,
    onError,
    onOpen,
    onClose
  } = options

  // 获取 baseURL（与 request.js 中的逻辑保持一致）
  const getBaseURL = () => {
    const apiBaseURL = import.meta.env.VITE_API_BASE_URL
    const apiPrefix = import.meta.env.VITE_API_PREFIX || '/api/admin'
    
    if (apiBaseURL) {
      const base = apiBaseURL.replace(/\/+$/, '')
      const prefix = apiPrefix.startsWith('/') ? apiPrefix : `/${apiPrefix}`
      return `${base}${prefix}`
    }
    
    return apiPrefix
  }

  // 构建完整的 URL
  const baseURL = getBaseURL()
  const fullURL = url.startsWith('http') ? url : `${baseURL}${url.startsWith('/') ? url : '/' + url}`

  // 获取 token
  const token = Storage.getItem('token', '')
  if (!token || typeof token !== 'string') {
    throw new Error('Token is required for SSE connection')
  }

  // 创建 EventSource
  const eventSource = new EventSource(buildSSEUrl(fullURL, token))

  // 设置事件监听器
  if (onOpen) {
    eventSource.onopen = onOpen
  }

  if (onMessage) {
    eventSource.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        onMessage(data, event)
      } catch (error) {
        console.error('Failed to parse SSE message:', error, event.data)
        // 如果解析失败，仍然传递原始数据
        onMessage(event.data, event)
      }
    }
  }

  if (onError) {
    eventSource.onerror = (error) => {
      // EventSource 的错误事件会在连接失败时触发，但 EventSource 会自动重连
      // 这里不直接打印错误，让调用者决定如何处理
      // console.error('SSE connection error:', error)
      onError(error, eventSource)
    }
  }

  // 添加关闭监听器
  if (onClose) {
    // EventSource 没有直接的 onclose 事件，需要通过其他方式实现
    // 可以在 onError 中检测 readyState 来判断是否关闭
    const originalOnError = eventSource.onerror
    eventSource.onerror = (error) => {
      if (originalOnError) {
        originalOnError(error)
      }
      if (eventSource.readyState === EventSource.CLOSED) {
        onClose()
      }
    }
  }

  return eventSource
}

/**
 * 关闭 SSE 连接
 * @param {EventSource} eventSource - EventSource 实例
 */
export function closeSSEConnection(eventSource) {
  if (eventSource && eventSource.readyState !== EventSource.CLOSED) {
    eventSource.close()
  }
}

/**
 * SSE 连接状态枚举
 */
export const SSE_STATE = {
  CONNECTING: 0,
  OPEN: 1,
  CLOSED: 2
}

