import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('./storage', () => {
  const store = new Map<string, unknown>()
  return {
    default: {
      getItem: <T>(key: string, fallback: T | '' = '') =>
        (store.has(key) ? (store.get(key) as T) : fallback),
      setItem: (key: string, value: unknown) => {
        store.set(key, value)
        return true
      },
      removeItem: (key: string) => {
        store.delete(key)
      },
      _store: store,
    },
  }
})

import Storage from './storage'
import {
  applyTenantHeader,
  clearTenantCode,
  resolveTenantCodeFromLocation,
  setTenantCode,
  withTenantQuery,
} from './tenant'
import { isAuthEndpointUrl } from './authEndpoint'

describe('isAuthEndpointUrl', () => {
  it('matches login/logout but not login-logs', () => {
    expect(isAuthEndpointUrl('login')).toBe(true)
    expect(isAuthEndpointUrl('/api/admin/login')).toBe(true)
    expect(isAuthEndpointUrl('login/captcha')).toBe(true)
    expect(isAuthEndpointUrl('logout')).toBe(true)
    expect(isAuthEndpointUrl('login-logs')).toBe(false)
    expect(isAuthEndpointUrl('/api/admin/login-logs')).toBe(false)
  })
})

describe('tenant helpers', () => {
  beforeEach(() => {
    ;(Storage as unknown as { _store: Map<string, unknown> })._store.clear()
    vi.stubEnv('VITE_TENANCY_ENABLED', 'true')
    vi.stubEnv('VITE_TENANCY_DRIVER', '')
    vi.stubEnv('VITE_TENANCY_HEADER', '')
  })

  it('stores and applies X-Tenant-ID header', () => {
    expect(setTenantCode('Acme')).toBe('acme')
    const headers: Record<string, unknown> = {}
    applyTenantHeader(headers)
    expect(headers['X-Tenant-ID']).toBe('acme')
    clearTenantCode()
    const empty: Record<string, unknown> = {}
    applyTenantHeader(empty)
    expect(empty['X-Tenant-ID']).toBeUndefined()
  })

  it('resolves tenant from query string', () => {
    expect(resolveTenantCodeFromLocation('?tenant_code=Beta')).toBe('beta')
    expect(resolveTenantCodeFromLocation('?tenant=Gamma')).toBe('gamma')
  })

  it('appends tenant_code to public attachment URLs', () => {
    setTenantCode('demo')
    const url = withTenantQuery('/api/admin/public/images/1')
    expect(url).toContain('tenant_code=demo')
    expect(withTenantQuery('/api/admin/admins')).toBe('/api/admin/admins')
  })
})
