import { describe, expect, it } from 'vitest'
import { validateTimeRange } from './timeRangeValidator'

describe('validateTimeRange', () => {
  it('skips when either side empty', () => {
    expect(validateTimeRange('', '2026-01-01 00:00:00').valid).toBe(true)
  })

  it('rejects start after end', () => {
    const res = validateTimeRange('2026-03-01 00:00:00', '2026-01-01 00:00:00')
    expect(res.valid).toBe(false)
    expect(res.errorKey).toBe('start_time_after_end_time')
  })

  it('rejects range over max months', () => {
    const res = validateTimeRange('2026-01-01 00:00:00', '2026-05-01 00:00:00', 3)
    expect(res.valid).toBe(false)
    expect(res.errorKey).toBe('time_range_exceeded')
  })

  it('accepts range within limit', () => {
    expect(validateTimeRange('2026-01-01 00:00:00', '2026-02-15 00:00:00', 3).valid).toBe(true)
  })
})
