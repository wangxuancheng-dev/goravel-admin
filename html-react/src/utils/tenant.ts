/**
 * Tenancy helpers for React admin.
 * Enable with VITE_TENANCY_ENABLED=true or VITE_TENANCY_DRIVER=database
 * (must match backend TENANCY_DRIVER=database).
 */
import Storage from './storage'

const STORAGE_KEY = 'tenant_code'

export function isTenancyEnabled(): boolean {
  const enabled = String(import.meta.env.VITE_TENANCY_ENABLED || '').toLowerCase()
  const driver = String(import.meta.env.VITE_TENANCY_DRIVER || '').toLowerCase()
  return enabled === 'true' || enabled === '1' || driver === 'database'
}

export function getTenantHeaderName(): string {
  return import.meta.env.VITE_TENANCY_HEADER || 'X-Tenant-ID'
}

export function getTenantCode(): string {
  return String(Storage.getItem<string>(STORAGE_KEY, '') || '').trim()
}

export function setTenantCode(code: string | undefined | null): string {
  const value = String(code || '').trim().toLowerCase()
  if (!value) {
    Storage.removeItem(STORAGE_KEY)
    return ''
  }
  Storage.setItem(STORAGE_KEY, value)
  return value
}

export function clearTenantCode(): void {
  Storage.removeItem(STORAGE_KEY)
}

/** Prefer ?tenant_code= / ?tenant= over stored value. */
export function resolveTenantCodeFromLocation(search = window.location.search): string {
  try {
    const params = new URLSearchParams(search)
    return String(params.get('tenant_code') || params.get('tenant') || '').trim().toLowerCase()
  } catch {
    return ''
  }
}

export function applyTenantHeader(headers: Record<string, unknown> | undefined): void {
  if (!headers) return
  const code = getTenantCode()
  if (!code) return
  headers[getTenantHeaderName()] = code
}

const ATTACHMENT_PUBLIC_HINT = /\/api\/admin\/public\/images\/|\/api\/public\/files\//

/**
 * Append tenant_code query for public asset URLs (img/src cannot send custom headers).
 * Apply whenever a local tenant code exists (same as applyTenantHeader), even if VITE_TENANCY_* is unset.
 */
export function withTenantQuery(url: string): string {
  const value = String(url || '').trim()
  if (!value) return value
  const code = getTenantCode()
  if (!code) return value
  if (!ATTACHMENT_PUBLIC_HINT.test(value)) return value
  try {
    const abs = value.startsWith('http') ? new URL(value) : new URL(value, 'http://local.invalid')
    if (!abs.searchParams.get('tenant_code') && !abs.searchParams.get('tenant_id')) {
      abs.searchParams.set('tenant_code', code)
    }
    if (value.startsWith('http')) return abs.toString()
    return `${abs.pathname}${abs.search}${abs.hash}`
  } catch {
    const sep = value.includes('?') ? '&' : '?'
    if (/[?&]tenant_code=/.test(value) || /[?&]tenant_id=/.test(value)) return value
    return `${value}${sep}tenant_code=${encodeURIComponent(code)}`
  }
}

/**
 * URL for "back to tenant admin login" from the platform console.
 * Prefer VITE_TENANT_DEMO_LOGIN_URL; hosted demo host falls back to the public tenant demo.
 */
export function getTenantAdminLoginUrl(): string {
  const fromEnv = String(import.meta.env.VITE_TENANT_DEMO_LOGIN_URL || '').trim()
  if (fromEnv) return fromEnv
  if (typeof window !== 'undefined' && window.location.hostname === 'admin.xuancheng888.top') {
    return 'https://acme.xuancheng888.top/login?tenant_code=acme'
  }
  return '/login'
}
