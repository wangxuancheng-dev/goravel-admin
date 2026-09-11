/**
 * Tenancy helpers for Vue admin.
 * Enable with VITE_TENANCY_ENABLED=true or VITE_TENANCY_DRIVER=database
 * (must match backend TENANCY_DRIVER=database).
 */
import Storage from './storage'

const STORAGE_KEY = 'tenant_code'

export function isTenancyEnabled() {
  const enabled = String(import.meta.env.VITE_TENANCY_ENABLED || '').toLowerCase()
  const driver = String(import.meta.env.VITE_TENANCY_DRIVER || '').toLowerCase()
  return enabled === 'true' || enabled === '1' || driver === 'database'
}

export function getTenantHeaderName() {
  return import.meta.env.VITE_TENANCY_HEADER || 'X-Tenant-ID'
}

export function getTenantCode() {
  return String(Storage.getItem(STORAGE_KEY, '') || '').trim()
}

export function setTenantCode(code) {
  const value = String(code || '').trim().toLowerCase()
  if (!value) {
    Storage.removeItem(STORAGE_KEY)
    return ''
  }
  Storage.setItem(STORAGE_KEY, value)
  return value
}

export function clearTenantCode() {
  Storage.removeItem(STORAGE_KEY)
}

/** Prefer ?tenant_code= / ?tenant= over stored value. */
export function resolveTenantCodeFromLocation(search = window.location.search) {
  try {
    const params = new URLSearchParams(search)
    return String(params.get('tenant_code') || params.get('tenant') || '').trim().toLowerCase()
  } catch {
    return ''
  }
}

export function applyTenantHeader(headers) {
  if (!headers) return
  const code = getTenantCode()
  if (!code) return
  headers[getTenantHeaderName()] = code
}
