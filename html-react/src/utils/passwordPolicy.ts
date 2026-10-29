/** 密码策略（与后端 services.PasswordPolicy 一致，由 /info 的 config.password_policy 下发）。 */
export interface PasswordPolicy {
  minLength: number
  requireLetter: boolean
  requireNumber: boolean
  requireSpecial: boolean
}

type Translate = (key: string, options?: Record<string, unknown>) => string

export const DEFAULT_PASSWORD_POLICY: PasswordPolicy = {
  minLength: 8,
  requireLetter: true,
  requireNumber: true,
  requireSpecial: true,
}

export function normalizePasswordPolicy(raw: unknown): PasswordPolicy {
  if (!raw || typeof raw !== 'object') return { ...DEFAULT_PASSWORD_POLICY }
  const r = raw as Record<string, unknown>
  const min = Number(r.min_length)
  const flag = (v: unknown, def: boolean) => (typeof v === 'boolean' ? v : def)
  return {
    minLength: Number.isFinite(min) && min > 0 ? min : DEFAULT_PASSWORD_POLICY.minLength,
    requireLetter: flag(r.require_letter, DEFAULT_PASSWORD_POLICY.requireLetter),
    requireNumber: flag(r.require_number, DEFAULT_PASSWORD_POLICY.requireNumber),
    requireSpecial: flag(r.require_special, DEFAULT_PASSWORD_POLICY.requireSpecial),
  }
}

// 字符分类与后端 unicode.IsLetter / IsDigit / IsPunct||IsSymbol 对齐
const LETTER = /\p{L}/u
const NUMBER = /\p{Nd}/u
const SPECIAL = /[\p{P}\p{S}]/u

function joinItems(keys: string[], t: Translate): string {
  return keys.map((k) => t(`password_policy.${k}`)).join(t('password_policy.separator'))
}

function requiredKinds(policy: PasswordPolicy): string[] {
  const kinds: string[] = []
  if (policy.requireLetter) kinds.push('letter')
  if (policy.requireNumber) kinds.push('number')
  if (policy.requireSpecial) kinds.push('special')
  return kinds
}

/** 返回不满足策略时的提示（一次列出所有缺少项）；满足或为空时返回空串。常见弱口令等仍由后端拦截。 */
export function passwordPolicyError(password: string | undefined, policy: PasswordPolicy, t: Translate): string {
  if (!password) return ''
  const missing = requiredKinds(policy).filter((kind) => {
    if (kind === 'letter') return !LETTER.test(password)
    if (kind === 'number') return !NUMBER.test(password)
    return !SPECIAL.test(password)
  })
  const tooShort = [...password].length < policy.minLength
  const min = policy.minLength
  if (tooShort && missing.length) {
    return t('password_policy.too_short_and_missing', { min, items: joinItems(missing, t) })
  }
  if (tooShort) return t('password_policy.too_short', { min })
  if (missing.length) return t('password_policy.missing', { items: joinItems(missing, t) })
  return ''
}

/** 输入框下方的规则说明，如「至少 8 位，需包含字母、数字、特殊符号」。 */
export function describePasswordPolicy(policy: PasswordPolicy, t: Translate): string {
  const kinds = requiredKinds(policy)
  const min = policy.minLength
  return kinds.length
    ? t('password_policy.hint_with_items', { min, items: joinItems(kinds, t) })
    : t('password_policy.hint', { min })
}
