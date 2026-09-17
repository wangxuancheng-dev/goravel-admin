import Storage from '@/utils/storage'
import { resolveEffectiveTenantCode } from '@/utils/tenant'

export interface SSEOptions {
  onMessage?: (data: unknown, event: MessageEvent) => void
  onError?: (error: Event, eventSource: EventSource) => void
  onOpen?: (event: Event) => void
  onClose?: () => void
}

function getBaseURL() {
  const apiBaseURL = import.meta.env.VITE_API_BASE_URL as string | undefined
  const apiPrefix = (import.meta.env.VITE_API_PREFIX as string | undefined) || '/api/admin'

  if (apiBaseURL) {
    const base = apiBaseURL.replace(/\/+$/, '')
    const prefix = apiPrefix.startsWith('/') ? apiPrefix : `/${apiPrefix}`
    return `${base}${prefix}`
  }
  return apiPrefix
}

function resolveSSETenantHint(): string {
  // Match applyTenantHeader: send whenever a tenant hint exists, even if VITE_TENANCY_* is unset.
  return resolveEffectiveTenantCode()
}

/** EventSource cannot set headers; pass JWT and tenant via query (matches Tenant middleware ClientHint). */
export function buildSSEUrl(fullURL: string, token: string): string {
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

export function createSSEConnection(url: string, options: SSEOptions = {}) {
  const { onMessage, onError, onOpen, onClose } = options
  const baseURL = getBaseURL()
  const fullURL = url.startsWith('http') ? url : `${baseURL}${url.startsWith('/') ? url : `/${url}`}`

  const token = Storage.getItem<string>('token', '')
  if (!token || typeof token !== 'string') {
    throw new Error('Token is required for SSE connection')
  }

  const eventSource = new EventSource(buildSSEUrl(fullURL, token))

  if (onOpen) eventSource.onopen = onOpen

  if (onMessage) {
    eventSource.onmessage = (event) => {
      try {
        onMessage(JSON.parse(event.data), event)
      } catch {
        onMessage(event.data, event)
      }
    }
  }

  eventSource.onerror = (error) => {
    onError?.(error, eventSource)
    if (eventSource.readyState === EventSource.CLOSED) {
      onClose?.()
    }
  }

  return eventSource
}

export function closeSSEConnection(eventSource: EventSource | null | undefined) {
  if (eventSource && eventSource.readyState !== EventSource.CLOSED) {
    eventSource.close()
  }
}

export const SSE_STATE = {
  CONNECTING: 0,
  OPEN: 1,
  CLOSED: 2,
} as const
