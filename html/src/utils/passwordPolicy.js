// 密码策略（与后端 services.PasswordPolicy 一致，由 /info 的 config.password_policy 下发）。
// 前端只做即时提示，常见弱口令、包含用户名等仍由后端最终校验。

export const DEFAULT_PASSWORD_POLICY = {
  minLength: 8,
  requireLetter: true,
  requireNumber: true,
  requireSpecial: true
}

export function normalizePasswordPolicy(raw) {
  if (!raw || typeof raw !== 'object') return { ...DEFAULT_PASSWORD_POLICY }
  const min = Number(raw.min_length)
  const flag = (v, def) => (typeof v === 'boolean' ? v : def)
  return {
    minLength: Number.isFinite(min) && min > 0 ? min : DEFAULT_PASSWORD_POLICY.minLength,
    requireLetter: flag(raw.require_letter, DEFAULT_PASSWORD_POLICY.requireLetter),
    requireNumber: flag(raw.require_number, DEFAULT_PASSWORD_POLICY.requireNumber),
    requireSpecial: flag(raw.require_special, DEFAULT_PASSWORD_POLICY.requireSpecial)
  }
}

// 字符分类与后端 unicode.IsLetter / IsDigit / IsPunct||IsSymbol 对齐
const LETTER = /\p{L}/u
const NUMBER = /\p{Nd}/u
const SPECIAL = /[\p{P}\p{S}]/u

function requiredKinds(policy) {
  const kinds = []
  if (policy.requireLetter) kinds.push('letter')
  if (policy.requireNumber) kinds.push('number')
  if (policy.requireSpecial) kinds.push('special')
  return kinds
}

function joinItems(kinds, t) {
  return kinds.map((k) => t(`password_policy.${k}`)).join(t('password_policy.separator'))
}

/** 不满足策略时返回提示（一次列出所有缺少项），满足或为空时返回空串。 */
export function passwordPolicyError(password, policy, t) {
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
export function describePasswordPolicy(policy, t) {
  const kinds = requiredKinds(policy)
  const min = policy.minLength
  return kinds.length
    ? t('password_policy.hint_with_items', { min, items: joinItems(kinds, t) })
    : t('password_policy.hint', { min })
}

/** element-plus 表单规则；policy 可传 getter，便于异步加载的策略在校验时取最新值。 */
export function passwordPolicyRule(policy, t, trigger = 'blur') {
  return {
    trigger,
    validator: (_rule, value, callback) => {
      const msg = passwordPolicyError(value, typeof policy === 'function' ? policy() : policy, t)
      callback(msg ? new Error(msg) : undefined)
    }
  }
}
