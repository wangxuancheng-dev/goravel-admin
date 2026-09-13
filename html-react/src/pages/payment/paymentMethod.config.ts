import type { TFunction } from 'i18next'

/** Backend-registered drivers only. Source of truth: RegisterPaymentGateway.
 * Adding a new channel:
 * 1) app/services/payment_gateway_<type>.go + init RegisterPaymentGateway
 * 2) append type to PAYMENT_GATEWAYS_ENABLED
 * 3) extend PaymentMethodType / PAYMENT_METHOD_TYPES / PAYMENT_TYPE_CONFIG_FIELDS + i18n `payment_method.type_<type>`
 * Unknown enabled types still appear via resolvePaymentMethodTypes (empty config fields).
 */
export type PaymentMethodType = 'mock' | 'wechat' | 'alipay'

export interface PaymentMethodSearchForm {
  name: string
  code: string
  type: string
  is_active: string | number
  description: string
  [key: string]: string | number
}

export interface PaymentMethodConfigField {
  key: string
  type: 'input' | 'textarea' | 'select'
  inputType?: 'password' | 'text'
  required?: boolean
  rows?: number
  group: 'basic' | 'advanced'
  labelKey: string
  placeholderKey: string
  tipKey?: string
  options?: { label: string; value: string }[]
}

export const PAYMENT_METHOD_TYPES: PaymentMethodType[] = ['mock', 'wechat', 'alipay']

export const CONFIG_KEY_ALIASES: Partial<Record<PaymentMethodType, Record<string, string>>> = {
  wechat: {
    api_key: 'api_v3_key',
    cert_path: 'private_key_path',
  },
}

export const paymentMethodInitialSearchForm: PaymentMethodSearchForm = {
  name: '',
  code: '',
  type: '',
  is_active: '',
  description: '',
}

/** Unknown enabled types get [] fields (custom drivers). */
export const PAYMENT_TYPE_CONFIG_FIELDS: Record<PaymentMethodType, PaymentMethodConfigField[]> = {
  mock: [
    { key: 'shared_secret', type: 'input', inputType: 'password', group: 'basic', labelKey: 'cfg_shared_secret', placeholderKey: 'cfg_shared_secret_ph', tipKey: 'tip_mock_shared_secret' },
    { key: 'notify_url', type: 'input', group: 'advanced', labelKey: 'cfg_notify_url', placeholderKey: 'cfg_notify_url_ph', tipKey: 'tip_notify_url' },
  ],
  wechat: [
    { key: 'app_id', type: 'input', required: true, group: 'basic', labelKey: 'cfg_app_id', placeholderKey: 'cfg_app_id_ph', tipKey: 'tip_wechat_app_id' },
    { key: 'mch_id', type: 'input', required: true, group: 'basic', labelKey: 'cfg_mch_id', placeholderKey: 'cfg_mch_id_ph', tipKey: 'tip_wechat_mch_id' },
    { key: 'api_v3_key', type: 'input', inputType: 'password', required: true, group: 'basic', labelKey: 'cfg_api_v3_key', placeholderKey: 'cfg_api_v3_key_ph', tipKey: 'tip_wechat_api_v3_key' },
    { key: 'cert_serial_no', type: 'input', required: true, group: 'basic', labelKey: 'cfg_cert_serial_no', placeholderKey: 'cfg_cert_serial_no_ph', tipKey: 'tip_wechat_cert_serial' },
    { key: 'private_key_path', type: 'input', group: 'advanced', labelKey: 'cfg_private_key_path', placeholderKey: 'cfg_private_key_path_ph', tipKey: 'tip_wechat_private_key' },
    { key: 'notify_url', type: 'input', group: 'advanced', labelKey: 'cfg_notify_url', placeholderKey: 'cfg_notify_url_ph', tipKey: 'tip_notify_url' },
  ],
  alipay: [
    { key: 'app_id', type: 'input', required: true, group: 'basic', labelKey: 'cfg_app_id', placeholderKey: 'cfg_alipay_app_id_ph', tipKey: 'tip_alipay_app_id' },
    { key: 'private_key', type: 'textarea', rows: 4, required: true, group: 'basic', labelKey: 'cfg_private_key', placeholderKey: 'cfg_private_key_ph', tipKey: 'tip_alipay_private_key' },
    { key: 'public_key', type: 'textarea', rows: 4, required: true, group: 'basic', labelKey: 'cfg_alipay_public_key', placeholderKey: 'cfg_alipay_public_key_ph', tipKey: 'tip_alipay_public_key' },
    { key: 'gateway', type: 'input', group: 'advanced', labelKey: 'cfg_gateway', placeholderKey: 'cfg_gateway_ph' },
    { key: 'notify_url', type: 'input', group: 'advanced', labelKey: 'cfg_notify_url', placeholderKey: 'cfg_notify_url_ph', tipKey: 'tip_notify_url' },
  ],
}

/**
 * When enabledTypes is null/undefined (info not loaded), fall back to registered catalog.
 * When array from server, filter + append unknown custom driver names.
 */
export function resolvePaymentMethodTypes(enabledTypes?: string[] | null): string[] {
  if (!Array.isArray(enabledTypes)) {
    return [...PAYMENT_METHOD_TYPES]
  }
  const wanted: string[] = []
  const seen = new Set<string>()
  for (const raw of enabledTypes) {
    const value = String(raw || '')
      .toLowerCase()
      .trim()
    if (!value || seen.has(value)) continue
    seen.add(value)
    wanted.push(value)
  }
  if (wanted.length === 0) {
    return []
  }
  const known: string[] = PAYMENT_METHOD_TYPES.filter((value) => seen.has(value))
  for (const value of wanted) {
    if (!known.includes(value)) {
      known.push(value)
    }
  }
  return known
}

export function createPaymentMethodTypeOptions(t: TFunction, enabledTypes?: string[] | null) {
  return resolvePaymentMethodTypes(enabledTypes).map((value) => ({
    label: t(`payment_method.type_${value}`),
    value,
  }))
}

export function getPaymentMethodTypeLabel(t: TFunction, type?: string): string {
  if (!type) return '-'
  const key = `payment_method.type_${type}`
  const label = t(key)
  return label === key ? String(type) : label
}

export function getConfigFieldsForType(type?: string): PaymentMethodConfigField[] {
  if (!type) return []
  return PAYMENT_TYPE_CONFIG_FIELDS[type as PaymentMethodType] || []
}

export function getConfigFieldsByGroup(type: string | undefined, group: 'basic' | 'advanced') {
  return getConfigFieldsForType(type).filter((f) => f.group === group)
}

export function createEmptyConfig(type?: string): Record<string, string> {
  const fields = getConfigFieldsForType(type)
  return Object.fromEntries(fields.map((field) => [field.key, '']))
}

export function generatePaymentCode(type?: string): string {
  const suffix = Date.now().toString(36).slice(-5)
  return `${type || 'pay'}_${suffix}`
}

export function normalizeConfigForForm(type: string | undefined, configData: Record<string, unknown> = {}) {
  const fields = getConfigFieldsForType(type)
  const aliases = CONFIG_KEY_ALIASES[(type || '') as PaymentMethodType] || {}
  const config = createEmptyConfig(type)

  fields.forEach((field) => {
    let value = configData[field.key]
    if ((value === undefined || value === null || value === '') && aliases) {
      const legacyKey = Object.keys(aliases).find((k) => aliases[k] === field.key)
      if (legacyKey && configData[legacyKey] != null) {
        value = configData[legacyKey]
      }
    }
    config[field.key] = value !== undefined && value !== null ? String(value) : ''
  })

  return config
}

export function collectConfigPayload(type: string | undefined, formConfig: Record<string, string> = {}) {
  const config: Record<string, string> = {}
  getConfigFieldsForType(type).forEach((field) => {
    const value = formConfig[field.key]
    if (value !== undefined && value !== null && String(value).trim() !== '') {
      config[field.key] = String(value).trim()
    }
  })
  return config
}
