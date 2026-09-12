/** @typedef {'mock'|'wechat'|'alipay'|'qq'|'allinpay'|'lakala'|'paypal'|'apple'|'saobei'} PaymentMethodType */

/**
 * @typedef {Object} PaymentConfigField
 * @property {string} key
 * @property {'input'|'textarea'|'select'} type
 * @property {'password'|'text'=} inputType
 * @property {boolean=} required
 * @property {number=} rows
 * @property {'basic'|'advanced'} group
 * @property {string} labelKey
 * @property {string} placeholderKey
 * @property {string=} tipKey
 * @property {{label: string, value: string}[]=} options
 */

export const PAYMENT_METHOD_TYPES = [
  'mock',
  'wechat',
  'alipay',
  'qq',
  'allinpay',
  'lakala',
  'paypal',
  'apple',
  'saobei'
]

/** 旧字段 → 新字段（编辑回填兼容） */
export const CONFIG_KEY_ALIASES = {
  wechat: {
    api_key: 'api_v3_key',
    cert_path: 'private_key_path'
  }
}

/**
 * 各支付类型配置字段（客服可读标签走 i18n）
 * @type {Record<PaymentMethodType, PaymentConfigField[]>}
 */
export const PAYMENT_TYPE_CONFIG_FIELDS = {
  mock: [
    { key: 'shared_secret', type: 'input', inputType: 'password', required: false, group: 'basic', labelKey: 'cfg_shared_secret', placeholderKey: 'cfg_shared_secret_ph', tipKey: 'tip_mock_shared_secret' },
    { key: 'notify_url', type: 'input', required: false, group: 'advanced', labelKey: 'cfg_notify_url', placeholderKey: 'cfg_notify_url_ph', tipKey: 'tip_notify_url' }
  ],
  wechat: [
    { key: 'app_id', type: 'input', required: true, group: 'basic', labelKey: 'cfg_app_id', placeholderKey: 'cfg_app_id_ph', tipKey: 'tip_wechat_app_id' },
    { key: 'mch_id', type: 'input', required: true, group: 'basic', labelKey: 'cfg_mch_id', placeholderKey: 'cfg_mch_id_ph', tipKey: 'tip_wechat_mch_id' },
    { key: 'api_v3_key', type: 'input', inputType: 'password', required: true, group: 'basic', labelKey: 'cfg_api_v3_key', placeholderKey: 'cfg_api_v3_key_ph', tipKey: 'tip_wechat_api_v3_key' },
    { key: 'cert_serial_no', type: 'input', required: true, group: 'basic', labelKey: 'cfg_cert_serial_no', placeholderKey: 'cfg_cert_serial_no_ph', tipKey: 'tip_wechat_cert_serial' },
    { key: 'private_key_path', type: 'input', required: false, group: 'advanced', labelKey: 'cfg_private_key_path', placeholderKey: 'cfg_private_key_path_ph', tipKey: 'tip_wechat_private_key' },
    { key: 'notify_url', type: 'input', required: false, group: 'advanced', labelKey: 'cfg_notify_url', placeholderKey: 'cfg_notify_url_ph', tipKey: 'tip_notify_url' }
  ],
  alipay: [
    { key: 'app_id', type: 'input', required: true, group: 'basic', labelKey: 'cfg_app_id', placeholderKey: 'cfg_alipay_app_id_ph', tipKey: 'tip_alipay_app_id' },
    { key: 'private_key', type: 'textarea', rows: 4, required: true, group: 'basic', labelKey: 'cfg_private_key', placeholderKey: 'cfg_private_key_ph', tipKey: 'tip_alipay_private_key' },
    { key: 'public_key', type: 'textarea', rows: 4, required: true, group: 'basic', labelKey: 'cfg_alipay_public_key', placeholderKey: 'cfg_alipay_public_key_ph', tipKey: 'tip_alipay_public_key' },
    { key: 'gateway', type: 'input', required: false, group: 'advanced', labelKey: 'cfg_gateway', placeholderKey: 'cfg_gateway_ph' },
    { key: 'notify_url', type: 'input', required: false, group: 'advanced', labelKey: 'cfg_notify_url', placeholderKey: 'cfg_notify_url_ph', tipKey: 'tip_notify_url' }
  ],
  qq: [
    { key: 'app_id', type: 'input', required: true, group: 'basic', labelKey: 'cfg_app_id', placeholderKey: 'cfg_qq_app_id_ph' },
    { key: 'app_key', type: 'input', inputType: 'password', required: true, group: 'basic', labelKey: 'cfg_app_key', placeholderKey: 'cfg_app_key_ph' },
    { key: 'mch_id', type: 'input', required: true, group: 'basic', labelKey: 'cfg_mch_id', placeholderKey: 'cfg_mch_id_ph' },
    { key: 'notify_url', type: 'input', required: false, group: 'advanced', labelKey: 'cfg_notify_url', placeholderKey: 'cfg_notify_url_ph', tipKey: 'tip_notify_url' }
  ],
  allinpay: [
    { key: 'merchant_id', type: 'input', required: true, group: 'basic', labelKey: 'cfg_merchant_id', placeholderKey: 'cfg_allinpay_merchant_ph' },
    { key: 'app_id', type: 'input', required: true, group: 'basic', labelKey: 'cfg_app_id', placeholderKey: 'cfg_allinpay_app_id_ph' },
    { key: 'app_key', type: 'input', inputType: 'password', required: true, group: 'basic', labelKey: 'cfg_app_key', placeholderKey: 'cfg_app_key_ph' },
    { key: 'gateway', type: 'input', required: false, group: 'advanced', labelKey: 'cfg_gateway', placeholderKey: 'cfg_gateway_ph' },
    { key: 'notify_url', type: 'input', required: false, group: 'advanced', labelKey: 'cfg_notify_url', placeholderKey: 'cfg_notify_url_ph', tipKey: 'tip_notify_url' }
  ],
  lakala: [
    { key: 'merchant_id', type: 'input', required: true, group: 'basic', labelKey: 'cfg_merchant_id', placeholderKey: 'cfg_lakala_merchant_ph' },
    { key: 'terminal_id', type: 'input', required: true, group: 'basic', labelKey: 'cfg_terminal_id', placeholderKey: 'cfg_terminal_id_ph' },
    { key: 'key', type: 'input', inputType: 'password', required: true, group: 'basic', labelKey: 'cfg_secret_key', placeholderKey: 'cfg_secret_key_ph' },
    { key: 'gateway', type: 'input', required: false, group: 'advanced', labelKey: 'cfg_gateway', placeholderKey: 'cfg_gateway_ph' },
    { key: 'notify_url', type: 'input', required: false, group: 'advanced', labelKey: 'cfg_notify_url', placeholderKey: 'cfg_notify_url_ph', tipKey: 'tip_notify_url' }
  ],
  paypal: [
    { key: 'client_id', type: 'input', required: true, group: 'basic', labelKey: 'cfg_client_id', placeholderKey: 'cfg_client_id_ph' },
    { key: 'client_secret', type: 'input', inputType: 'password', required: true, group: 'basic', labelKey: 'cfg_client_secret', placeholderKey: 'cfg_client_secret_ph' },
    {
      key: 'mode',
      type: 'select',
      required: false,
      group: 'advanced',
      labelKey: 'cfg_mode',
      placeholderKey: 'cfg_mode_ph',
      options: [
        { label: 'Sandbox', value: 'sandbox' },
        { label: 'Live', value: 'live' }
      ]
    },
    { key: 'notify_url', type: 'input', required: false, group: 'advanced', labelKey: 'cfg_notify_url', placeholderKey: 'cfg_notify_url_ph', tipKey: 'tip_notify_url' }
  ],
  apple: [
    { key: 'merchant_id', type: 'input', required: true, group: 'basic', labelKey: 'cfg_apple_merchant_id', placeholderKey: 'cfg_apple_merchant_id_ph' },
    { key: 'key_id', type: 'input', required: true, group: 'basic', labelKey: 'cfg_key_id', placeholderKey: 'cfg_key_id_ph' },
    { key: 'private_key', type: 'textarea', rows: 4, required: true, group: 'basic', labelKey: 'cfg_private_key', placeholderKey: 'cfg_private_key_ph' },
    { key: 'certificate', type: 'textarea', rows: 4, required: false, group: 'advanced', labelKey: 'cfg_certificate', placeholderKey: 'cfg_certificate_ph' }
  ],
  saobei: [
    { key: 'merchant_id', type: 'input', required: true, group: 'basic', labelKey: 'cfg_merchant_id', placeholderKey: 'cfg_saobei_merchant_ph' },
    { key: 'terminal_id', type: 'input', required: true, group: 'basic', labelKey: 'cfg_terminal_id', placeholderKey: 'cfg_terminal_id_ph' },
    { key: 'key', type: 'input', inputType: 'password', required: true, group: 'basic', labelKey: 'cfg_secret_key', placeholderKey: 'cfg_secret_key_ph' },
    { key: 'gateway', type: 'input', required: false, group: 'advanced', labelKey: 'cfg_gateway', placeholderKey: 'cfg_gateway_ph' },
    { key: 'notify_url', type: 'input', required: false, group: 'advanced', labelKey: 'cfg_notify_url', placeholderKey: 'cfg_notify_url_ph', tipKey: 'tip_notify_url' }
  ]
}

export const paymentMethodInitialSearchForm = {
  name: '',
  code: '',
  type: '',
  is_active: '',
  description: ''
}

export function createPaymentMethodTypeOptions(t) {
  return PAYMENT_METHOD_TYPES.map((value) => ({
    label: t(`payment_method.type_${value}`),
    value
  }))
}

export function getPaymentMethodTypeLabel(t, type) {
  if (!type) return '-'
  const key = `payment_method.type_${type}`
  const label = t(key)
  return label === key ? String(type) : label
}

export function getConfigFieldsForType(type) {
  if (!type) return []
  return PAYMENT_TYPE_CONFIG_FIELDS[type] || []
}

export function getConfigFieldsByGroup(type, group) {
  return getConfigFieldsForType(type).filter((f) => f.group === group)
}

export function createEmptyConfig(type) {
  const fields = getConfigFieldsForType(type)
  return Object.fromEntries(fields.map((f) => [f.key, '']))
}

/** 生成客服无需手填的内部编码 */
export function generatePaymentCode(type) {
  const suffix = Date.now().toString(36).slice(-5)
  return `${type || 'pay'}_${suffix}`
}

/** 将后端 config 映射到当前 schema（含旧字段兼容） */
export function normalizeConfigForForm(type, configData = {}) {
  const fields = getConfigFieldsForType(type)
  const aliases = CONFIG_KEY_ALIASES[type] || {}
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

export function collectConfigPayload(type, formConfig = {}) {
  const config = {}
  getConfigFieldsForType(type).forEach((field) => {
    const value = formConfig[field.key]
    if (value !== undefined && value !== null && String(value).trim() !== '') {
      config[field.key] = String(value).trim()
    }
  })
  return config
}

export function createPaymentMethodSearchFields(t) {
  return [
    { prop: 'name', type: 'input', placeholder: t('payment_method.name_placeholder') },
    {
      prop: 'type',
      type: 'select',
      placeholder: t('payment_method.type_placeholder'),
      options: createPaymentMethodTypeOptions(t)
    },
    {
      prop: 'is_active',
      type: 'select',
      placeholder: t('payment_method.is_active_placeholder'),
      options: [
        { label: t('common.enabled'), value: '1' },
        { label: t('common.disabled'), value: '0' }
      ]
    }
  ]
}

export function createPaymentMethodTableColumns(t) {
  return [
    { field: 'id', title: t('table.id'), width: 80, sortable: true, key: 'id' },
    { field: 'name', title: t('payment_method.name'), minWidth: 160, sortable: true, key: 'name' },
    { field: 'type', title: t('payment_method.type'), width: 120, sortable: true, slot: 'type', key: 'type' },
    { field: 'is_active', title: t('table.status'), width: 100, slot: 'is_active', key: 'is_active' },
    { field: 'sort', title: t('table.sort'), width: 90, sortable: true, key: 'sort' },
    { field: 'description', title: t('table.description'), minWidth: 180, key: 'description' },
    { field: 'created_at', title: t('table.created_at'), width: 180, sortable: true, key: 'created_at' },
    { title: t('table.operation'), width: 150, fixed: 'right', slot: 'operation', sortable: false, key: 'operation' }
  ]
}
