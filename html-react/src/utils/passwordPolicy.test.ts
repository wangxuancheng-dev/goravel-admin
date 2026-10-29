import { describe, expect, it } from 'vitest'
import {
  DEFAULT_PASSWORD_POLICY,
  describePasswordPolicy,
  normalizePasswordPolicy,
  passwordPolicyError,
} from './passwordPolicy'

// 回显 key + 参数，便于断言
const t = (key: string, options?: Record<string, unknown>) =>
  key === 'password_policy.separator' ? ',' : options ? `${key}|${JSON.stringify(options)}` : key.split('.').pop()!

describe('passwordPolicyError', () => {
  const policy = DEFAULT_PASSWORD_POLICY

  it('skips empty value (left to required rule)', () => {
    expect(passwordPolicyError('', policy, t)).toBe('')
  })

  it('lists length and every missing kind at once', () => {
    expect(passwordPolicyError('123456', policy, t)).toBe(
      'password_policy.too_short_and_missing|{"min":8,"items":"letter,special"}',
    )
  })

  it('reports only missing kinds when long enough', () => {
    expect(passwordPolicyError('abc12345', policy, t)).toBe('password_policy.missing|{"items":"special"}')
  })

  it('reports only length when kinds satisfied', () => {
    expect(passwordPolicyError('a1!', policy, t)).toBe('password_policy.too_short|{"min":8}')
  })

  it('accepts unicode letters and symbols like the backend', () => {
    expect(passwordPolicyError('密码abc1+=~', policy, t)).toBe('')
  })

  it('honours relaxed policy', () => {
    const relaxed = normalizePasswordPolicy({ min_length: 6, require_letter: false, require_number: true, require_special: false })
    expect(passwordPolicyError('123456', relaxed, t)).toBe('')
  })
})

describe('normalizePasswordPolicy', () => {
  it('falls back to strict defaults', () => {
    expect(normalizePasswordPolicy(undefined)).toEqual(DEFAULT_PASSWORD_POLICY)
    expect(normalizePasswordPolicy({ min_length: 0 }).minLength).toBe(8)
  })
})

describe('describePasswordPolicy', () => {
  it('describes min length and required kinds', () => {
    expect(describePasswordPolicy(DEFAULT_PASSWORD_POLICY, t)).toBe(
      'password_policy.hint_with_items|{"min":8,"items":"letter,number,special"}',
    )
  })
})
