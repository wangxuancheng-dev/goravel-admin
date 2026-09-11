import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('./logger', () => ({
  default: { warn: vi.fn(), info: vi.fn(), error: vi.fn() },
}))

import { Storage } from './storage'

describe('Storage', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('round-trips JSON values', () => {
    expect(Storage.setItem('k', { a: 1 })).toBe(true)
    expect(Storage.getItem('k')).toEqual({ a: 1 })
    expect(Storage.removeItem('k')).toBe(true)
    expect(Storage.getItem('k', 'fallback')).toBe('fallback')
  })
})
