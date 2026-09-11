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
