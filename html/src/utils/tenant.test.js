import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('./storage', () => {
  const store = new Map()
  return {
    default: {
      getItem: (key, fallback = '') => (store.has(key) ? store.get(key) : fallback),
      setItem: (key, value) => {
        store.set(key, value)
        return true
      },
      removeItem: (key) => {
        store.delete(key)
      },
      _store: store,
    },
  }
})

import Storage from './storage'
import {
  applyTenantHeader,
  buildAdminLoginPath,
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
    Storage._store.clear()
    vi.stubEnv('VITE_TENANCY_ENABLED', 'true')
    vi.stubEnv('VITE_TENANCY_DRIVER', '')
    vi.stubEnv('VITE_TENANCY_HEADER', '')
  })

  it('stores and applies X-Tenant-ID header', () => {
    expect(setTenantCode('Acme')).toBe('acme')
    const headers = {}
    applyTenantHeader(headers)
    expect(headers['X-Tenant-ID']).toBe('acme')
    clearTenantCode()
    const empty = {}
    applyTenantHeader(empty)
    expect(empty['X-Tenant-ID']).toBeUndefined()
  })

  it('resolves tenant from query string', () => {
    expect(resolveTenantCodeFromLocation('?tenant_code=Beta')).toBe('beta')
    expect(resolveTenantCodeFromLocation('?tenant=Gamma')).toBe('gamma')
  })

  it('builds admin login path with tenant_code', () => {
    expect(buildAdminLoginPath()).toBe('/login')
    expect(buildAdminLoginPath('Acme')).toBe('/login?tenant_code=acme')
    setTenantCode('demo')
    expect(buildAdminLoginPath()).toBe('/login?tenant_code=demo')
    clearTenantCode()
    expect(buildAdminLoginPath('kept')).toBe('/login?tenant_code=kept')
  })

  it('appends tenant_code to public attachment URLs', () => {
    setTenantCode('demo')
    const url = withTenantQuery('/api/admin/public/images/1')
    expect(url).toContain('tenant_code=demo')
    expect(withTenantQuery('/api/admin/admins')).toBe('/api/admin/admins')
  })

  it('appends tenant_code even when VITE_TENANCY_* is unset', () => {
    vi.stubEnv('VITE_TENANCY_ENABLED', '')
    vi.stubEnv('VITE_TENANCY_DRIVER', '')
    setTenantCode('acme')
    expect(withTenantQuery('/api/admin/public/images/9')).toContain('tenant_code=acme')
  })
})
