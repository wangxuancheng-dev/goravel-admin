import { describe, expect, it } from 'vitest'
import {
  allowedFormTypes,
  defaultFormType,
  normalizeFieldControls,
} from './fieldFormOptions'

describe('fieldFormOptions', () => {
  it('excludes rich text and checkbox for integer fields', () => {
    const allowed = allowedFormTypes('integer')
    expect(allowed).toContain('number')
    expect(allowed).not.toContain('editor')
    expect(allowed).not.toContain('markdown')
    expect(allowed).not.toContain('checkbox')
    expect(allowed).not.toContain('image-upload')
  })

  it('normalizes invalid form_type when type changes to decimal', () => {
    const next = normalizeFieldControls({
      type: 'decimal',
      form_type: 'editor',
      search_type: 'like',
      search_ui_type: 'datetimerange',
    })
    expect(next.form_type).toBe('number')
    expect(next.search_type).toBe('=')
    expect(next.search_ui_type).toBe('input')
  })

  it('defaults boolean to switch', () => {
    expect(defaultFormType('boolean')).toBe('switch')
    expect(allowedFormTypes('boolean')).toEqual(['switch', 'radio', 'select'])
  })
})
