/** @typedef {'mock'|'wechat'|'alipay'} PaymentMethodType */

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

/** Backend-registered drivers only (mock/wechat/alipay). Source of truth: RegisterPaymentGateway. */
export const PAYMENT_METHOD_TYPES = [
  'mock',
  'wechat',
  'alipay'
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
 * Unknown enabled types (custom drivers) get empty fields — config can still be edited as free-form later.
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
  ]
}

export const paymentMethodInitialSearchForm = {
  name: '',
  code: '',
  type: '',
  is_active: '',
  description: ''
}

/**
 * @param {string[]|null|undefined} enabledTypes from /info config.payment_gateways
 * When null/undefined (info not loaded), fall back to registered catalog — never phantom types.
 */
export function resolvePaymentMethodTypes(enabledTypes) {
  if (!Array.isArray(enabledTypes)) {
    return PAYMENT_METHOD_TYPES
  }
  const wanted = []
  const seen = new Set()
  for (const raw of enabledTypes) {
    const value = String(raw || '').toLowerCase().trim()
    if (!value || seen.has(value)) continue
    seen.add(value)
    wanted.push(value)
  }
  if (wanted.length === 0) {
    return []
  }
  const known = PAYMENT_METHOD_TYPES.filter((value) => seen.has(value))
  for (const value of wanted) {
    if (!known.includes(value)) {
      known.push(value)
    }
  }
  return known
}

export function createPaymentMethodTypeOptions(t, enabledTypes) {
  return resolvePaymentMethodTypes(enabledTypes).map((value) => ({
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

export function createPaymentMethodSearchFields(t, enabledTypes) {
  return [
    { prop: 'name', type: 'input', placeholder: t('payment_method.name_placeholder') },
    {
      prop: 'type',
      type: 'select',
      placeholder: t('payment_method.type_placeholder'),
      options: createPaymentMethodTypeOptions(t, enabledTypes)
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
