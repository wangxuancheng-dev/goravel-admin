import { describe, expect, it } from 'vitest'
import { escapeHtml, isSafeUrl } from './xss'

describe('xss helpers', () => {
  it('escapes html entities', () => {
    expect(escapeHtml(`<script>"x"'&`)).toBe('&lt;script&gt;&quot;x&quot;&#039;&amp;')
  })

  it('allows http(s) and relative urls', () => {
    expect(isSafeUrl('https://example.com')).toBe(true)
    expect(isSafeUrl('/path')).toBe(true)
    expect(isSafeUrl('javascript:alert(1)')).toBe(false)
  })
})
