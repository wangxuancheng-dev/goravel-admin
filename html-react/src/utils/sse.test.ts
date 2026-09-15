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
import { setTenantCode } from './tenant'
import { buildSSEUrl } from './sse'

describe('buildSSEUrl', () => {
  beforeEach(() => {
    ;(Storage as unknown as { _store: Map<string, unknown> })._store.clear()
    vi.stubEnv('VITE_TENANCY_ENABLED', '')
    vi.stubEnv('VITE_TENANCY_DRIVER', '')
  })

  it('appends _token and tenant hints even when VITE_TENANCY_* is unset', () => {
    setTenantCode('acme')
    const url = buildSSEUrl('/api/admin/monitor/system-info/stream?interval=2', ' tok ')
    expect(url).toContain('/api/admin/monitor/system-info/stream?')
    expect(url).toContain('interval=2')
    expect(url).toContain('_token=tok')
    expect(url).toContain('tenant_code=acme')
    expect(url).toContain('tenant_id=acme')
  })

  it('does not duplicate tenant_code when already present', () => {
    setTenantCode('acme')
    const url = buildSSEUrl('/api/admin/x?tenant_code=other', 't')
    expect(url).toContain('tenant_code=other')
    expect(url.match(/tenant_code=/g)?.length).toBe(1)
    expect(url).not.toContain('tenant_id=')
  })

  it('skips tenant hints when no tenant code is stored', () => {
    const url = buildSSEUrl('/api/admin/x', 't')
    expect(url).toContain('_token=t')
    expect(url).not.toContain('tenant_code=')
    expect(url).not.toContain('tenant_id=')
  })
})
