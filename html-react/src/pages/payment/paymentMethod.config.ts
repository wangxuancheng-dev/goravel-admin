import type { TFunction } from 'i18next'

export type PaymentMethodType =
  | 'mock'
  | 'wechat'
  | 'alipay'
  | 'qq'
  | 'allinpay'
  | 'lakala'
  | 'paypal'
  | 'apple'
  | 'saobei'

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

export const PAYMENT_METHOD_TYPES: PaymentMethodType[] = [
  'mock',
  'wechat',
  'alipay',
  'qq',
  'allinpay',
  'lakala',
  'paypal',
  'apple',
  'saobei',
]

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
  qq: [
    { key: 'app_id', type: 'input', required: true, group: 'basic', labelKey: 'cfg_app_id', placeholderKey: 'cfg_qq_app_id_ph' },
    { key: 'app_key', type: 'input', inputType: 'password', required: true, group: 'basic', labelKey: 'cfg_app_key', placeholderKey: 'cfg_app_key_ph' },
    { key: 'mch_id', type: 'input', required: true, group: 'basic', labelKey: 'cfg_mch_id', placeholderKey: 'cfg_mch_id_ph' },
    { key: 'notify_url', type: 'input', group: 'advanced', labelKey: 'cfg_notify_url', placeholderKey: 'cfg_notify_url_ph', tipKey: 'tip_notify_url' },
  ],
  allinpay: [
    { key: 'merchant_id', type: 'input', required: true, group: 'basic', labelKey: 'cfg_merchant_id', placeholderKey: 'cfg_allinpay_merchant_ph' },
    { key: 'app_id', type: 'input', required: true, group: 'basic', labelKey: 'cfg_app_id', placeholderKey: 'cfg_allinpay_app_id_ph' },
    { key: 'app_key', type: 'input', inputType: 'password', required: true, group: 'basic', labelKey: 'cfg_app_key', placeholderKey: 'cfg_app_key_ph' },
    { key: 'gateway', type: 'input', group: 'advanced', labelKey: 'cfg_gateway', placeholderKey: 'cfg_gateway_ph' },
    { key: 'notify_url', type: 'input', group: 'advanced', labelKey: 'cfg_notify_url', placeholderKey: 'cfg_notify_url_ph', tipKey: 'tip_notify_url' },
  ],
  lakala: [
    { key: 'merchant_id', type: 'input', required: true, group: 'basic', labelKey: 'cfg_merchant_id', placeholderKey: 'cfg_lakala_merchant_ph' },
    { key: 'terminal_id', type: 'input', required: true, group: 'basic', labelKey: 'cfg_terminal_id', placeholderKey: 'cfg_terminal_id_ph' },
    { key: 'key', type: 'input', inputType: 'password', required: true, group: 'basic', labelKey: 'cfg_secret_key', placeholderKey: 'cfg_secret_key_ph' },
    { key: 'gateway', type: 'input', group: 'advanced', labelKey: 'cfg_gateway', placeholderKey: 'cfg_gateway_ph' },
    { key: 'notify_url', type: 'input', group: 'advanced', labelKey: 'cfg_notify_url', placeholderKey: 'cfg_notify_url_ph', tipKey: 'tip_notify_url' },
  ],
  paypal: [
    { key: 'client_id', type: 'input', required: true, group: 'basic', labelKey: 'cfg_client_id', placeholderKey: 'cfg_client_id_ph' },
    { key: 'client_secret', type: 'input', inputType: 'password', required: true, group: 'basic', labelKey: 'cfg_client_secret', placeholderKey: 'cfg_client_secret_ph' },
    {
      key: 'mode',
      type: 'select',
      group: 'advanced',
      labelKey: 'cfg_mode',
      placeholderKey: 'cfg_mode_ph',
      options: [
        { label: 'Sandbox', value: 'sandbox' },
        { label: 'Live', value: 'live' },
      ],
    },
    { key: 'notify_url', type: 'input', group: 'advanced', labelKey: 'cfg_notify_url', placeholderKey: 'cfg_notify_url_ph', tipKey: 'tip_notify_url' },
  ],
  apple: [
    { key: 'merchant_id', type: 'input', required: true, group: 'basic', labelKey: 'cfg_apple_merchant_id', placeholderKey: 'cfg_apple_merchant_id_ph' },
    { key: 'key_id', type: 'input', required: true, group: 'basic', labelKey: 'cfg_key_id', placeholderKey: 'cfg_key_id_ph' },
    { key: 'private_key', type: 'textarea', rows: 4, required: true, group: 'basic', labelKey: 'cfg_private_key', placeholderKey: 'cfg_private_key_ph' },
    { key: 'certificate', type: 'textarea', rows: 4, group: 'advanced', labelKey: 'cfg_certificate', placeholderKey: 'cfg_certificate_ph' },
  ],
  saobei: [
    { key: 'merchant_id', type: 'input', required: true, group: 'basic', labelKey: 'cfg_merchant_id', placeholderKey: 'cfg_saobei_merchant_ph' },
    { key: 'terminal_id', type: 'input', required: true, group: 'basic', labelKey: 'cfg_terminal_id', placeholderKey: 'cfg_terminal_id_ph' },
    { key: 'key', type: 'input', inputType: 'password', required: true, group: 'basic', labelKey: 'cfg_secret_key', placeholderKey: 'cfg_secret_key_ph' },
    { key: 'gateway', type: 'input', group: 'advanced', labelKey: 'cfg_gateway', placeholderKey: 'cfg_gateway_ph' },
    { key: 'notify_url', type: 'input', group: 'advanced', labelKey: 'cfg_notify_url', placeholderKey: 'cfg_notify_url_ph', tipKey: 'tip_notify_url' },
  ],
}

export function createPaymentMethodTypeOptions(t: TFunction) {
  return PAYMENT_METHOD_TYPES.map((value) => ({
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
