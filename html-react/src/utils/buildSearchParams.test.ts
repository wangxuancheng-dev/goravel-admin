import { describe, expect, it } from 'vitest'
import { buildSearchParams } from './buildSearchParams'

describe('buildSearchParams', () => {
  it('keeps extras and drops empty search fields', () => {
    expect(
      buildSearchParams(
        { keyword: '  hello  ', status: '', page: null, active: true, count: 0 },
        { page: 1, page_size: 20 },
      ),
    ).toEqual({
      page: 1,
      page_size: 20,
      keyword: 'hello',
      active: true,
      count: 0,
    })
  })

  it('returns extras only when search form is empty', () => {
    expect(buildSearchParams({}, { sort: 'id:desc' })).toEqual({ sort: 'id:desc' })
  })
})
