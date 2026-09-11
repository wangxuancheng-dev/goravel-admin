import { useEffect, useMemo, useState } from 'react'
import {
  Alert,
  App,
  Button,
  Collapse,
  Form,
  Input,
  InputNumber,
  Modal,
  Select,
  Space,
  Steps,
  Switch,
  Tag,
  Typography,
} from 'antd'
import { useTranslation } from 'react-i18next'
import { createPaymentMethod, getPaymentMethodDetail, updatePaymentMethod } from '@/api/paymentMethod'
import { useUnhandledError } from '@/hooks/useUnhandledError'
import { entityField } from '@/utils/normalize'
import {
  collectConfigPayload,
  createEmptyConfig,
  createPaymentMethodTypeOptions,
  generatePaymentCode,
  getConfigFieldsByGroup,
  getPaymentMethodTypeLabel,
  normalizeConfigForForm,
  type PaymentMethodConfigField,
  type PaymentMethodType,
} from './paymentMethod.config'

interface PaymentMethodFormModalProps {
  open: boolean
  editId?: string | number | null
  onClose: () => void
  onSuccess: () => void
}

interface FormValues {
  name: string
  code?: string
  type?: PaymentMethodType
  is_active: boolean
  sort: number
  description?: string
  config: Record<string, string>
}

function ConfigFieldItem({
  field,
  t,
}: {
  field: PaymentMethodConfigField
  t: (key: string, options?: Record<string, string>) => string
}) {
  const label = t(`payment_method.${field.labelKey}`)
  return (
    <Form.Item
      name={['config', field.key]}
      label={label}
      extra={field.tipKey ? t(`payment_method.${field.tipKey}`) : undefined}
      rules={
        field.required
          ? [{ required: true, message: t('payment_method.field_required', { field: label }) }]
          : undefined
      }
    >
      {field.type === 'select' ? (
        <Select
          allowClear
          placeholder={t(`payment_method.${field.placeholderKey}`)}
          options={field.options}
        />
      ) : field.type === 'textarea' ? (
        <Input.TextArea rows={field.rows || 3} placeholder={t(`payment_method.${field.placeholderKey}`)} />
      ) : (
        <Input
          type={field.inputType === 'password' ? 'password' : 'text'}
          placeholder={t(`payment_method.${field.placeholderKey}`)}
        />
      )}
    </Form.Item>
  )
}

export default function PaymentMethodFormModal({
  open,
  editId,
  onClose,
  onSuccess,
}: PaymentMethodFormModalProps) {
  const { t } = useTranslation()
  const { message } = App.useApp()
  const showError = useUnhandledError()
  const [form] = Form.useForm<FormValues>()
  const [loading, setLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [step, setStep] = useState(0)
  const isEdit = Boolean(editId)

  const typeValue = Form.useWatch('type', form)
  const typeOptions = useMemo(() => createPaymentMethodTypeOptions(t), [t])
  const basicFields = useMemo(() => getConfigFieldsByGroup(typeValue, 'basic'), [typeValue])
  const advancedFields = useMemo(() => getConfigFieldsByGroup(typeValue, 'advanced'), [typeValue])

  useEffect(() => {
    if (!open) return

    if (!editId) {
      form.setFieldsValue({
        name: '',
        code: '',
        type: undefined,
        is_active: true,
        sort: 0,
        description: '',
        config: {},
      })
      setStep(0)
      return
    }

    setLoading(true)
    getPaymentMethodDetail(editId)
      .then((res) => {
        const data = (res.data || {}) as Record<string, unknown>
        const type = String(entityField(data, 'type', '') ?? '') as PaymentMethodType
        const configData = (entityField(data, 'config', {}) || {}) as Record<string, unknown>

        form.setFieldsValue({
          name: String(entityField(data, 'name', '') ?? ''),
          code: String(entityField(data, 'code', '') ?? ''),
          type: type || undefined,
          is_active: Boolean(entityField(data, 'is_active', true)),
          sort: Number(entityField(data, 'sort', 0) ?? 0),
          description: String(entityField(data, 'description', '') ?? ''),
          config: normalizeConfigForForm(type, configData),
        })
        setStep(1)
      })
      .catch((error) => showError(error, t('common.query_failed')))
      .finally(() => setLoading(false))
  }, [open, editId, form, showError, t])

  const selectType = (type: PaymentMethodType) => {
    const currentName = form.getFieldValue('name')
    const prevType = form.getFieldValue('type')
    const prevLabel = prevType ? getPaymentMethodTypeLabel(t, prevType) : ''
    form.setFieldsValue({
      type,
      config: createEmptyConfig(type),
      code: generatePaymentCode(type),
      name: !currentName || currentName === prevLabel ? getPaymentMethodTypeLabel(t, type) : currentName,
    })
    setStep(1)
  }

  const goNext = () => {
    const type = form.getFieldValue('type')
    if (!type) {
      message.warning(t('payment_method.type_required'))
      return
    }
    setStep(1)
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      const config = collectConfigPayload(values.type, values.config || {})

      if (Object.keys(config).length === 0) {
        message.error(t('payment_method.config_required'))
        return
      }

      setSubmitting(true)
      const payload: Record<string, unknown> = {
        name: values.name.trim(),
        is_active: values.is_active,
        sort: values.sort ?? 0,
        config,
        description: values.description?.trim() || '',
      }

      if (isEdit) {
        await updatePaymentMethod(editId!, payload)
        message.success(t('payment_method.update_success'))
      } else {
        await createPaymentMethod({
          ...payload,
          code: values.code || generatePaymentCode(values.type),
          type: values.type,
        })
        message.success(t('payment_method.create_success'))
      }

      onClose()
      onSuccess()
    } catch (error) {
      if ((error as { errorFields?: unknown })?.errorFields) return
      showError(error, t('common.operation_failed'))
    } finally {
      setSubmitting(false)
    }
  }

  const footer = (
    <Space>
      <Button onClick={onClose}>{t('common.cancel')}</Button>
      {!isEdit && step === 1 ? <Button onClick={() => setStep(0)}>{t('payment_method.prev_step')}</Button> : null}
      {!isEdit && step === 0 ? (
        <Button type="primary" disabled={!typeValue} onClick={goNext}>
          {t('payment_method.next_step')}
        </Button>
      ) : null}
      {isEdit || step === 1 ? (
        <Button type="primary" loading={submitting} onClick={() => void handleSubmit()}>
          {t('common.confirm')}
        </Button>
      ) : null}
    </Space>
  )

  return (
    <Modal
      open={open}
      title={isEdit ? t('payment_method.edit_payment_method') : t('payment_method.add_payment_method')}
      width={760}
      footer={footer}
      confirmLoading={submitting}
      onCancel={onClose}
      destroyOnClose
    >
      {!isEdit ? (
        <Steps
          current={step}
          style={{ marginBottom: 20 }}
          items={[
            { title: t('payment_method.step_choose_type') },
            { title: t('payment_method.step_fill_config') },
          ]}
        />
      ) : null}

      {!isEdit && step === 0 ? (
        <div>
          <Typography.Paragraph type="secondary">{t('payment_method.choose_type_hint')}</Typography.Paragraph>
          <div
            style={{
              display: 'grid',
              gridTemplateColumns: 'repeat(2, minmax(0, 1fr))',
              gap: 12,
            }}
          >
            {typeOptions.map((opt) => {
              const active = typeValue === opt.value
              return (
                <button
                  key={opt.value}
                  type="button"
                  onClick={() => selectType(opt.value as PaymentMethodType)}
                  style={{
                    textAlign: 'left',
                    padding: '14px 16px',
                    borderRadius: 10,
                    border: active ? '1px solid #1677ff' : '1px solid #d9d9d9',
                    background: active ? '#e6f4ff' : '#fff',
                    cursor: 'pointer',
                  }}
                >
                  <div style={{ fontWeight: 600, marginBottom: 4 }}>{opt.label}</div>
                  <div style={{ fontSize: 12, color: 'rgba(0,0,0,0.45)' }}>
                    {t(`payment_method.type_desc_${opt.value}`)}
                  </div>
                </button>
              )
            })}
          </div>
        </div>
      ) : (
        <Form form={form} layout="vertical" disabled={loading}>
          {typeValue ? (
            <Alert
              style={{ marginBottom: 16 }}
              type="info"
              showIcon
              message={t('payment_method.guide_title', {
                type: getPaymentMethodTypeLabel(t, typeValue),
              })}
              description={t(`payment_method.guide_${typeValue}`)}
            />
          ) : null}

          {isEdit ? (
            <Form.Item label={t('payment_method.type')}>
              <Space>
                <Tag color="blue">{getPaymentMethodTypeLabel(t, typeValue)}</Tag>
                <Typography.Text type="secondary">
                  {t('payment_method.code')}：{form.getFieldValue('code')}
                </Typography.Text>
              </Space>
            </Form.Item>
          ) : null}

          <Form.Item
            name="name"
            label={t('payment_method.name')}
            extra={t('payment_method.name_tip')}
            rules={[{ required: true, message: t('payment_method.name_required') }]}
          >
            <Input maxLength={50} showCount placeholder={t('payment_method.name_placeholder')} />
          </Form.Item>

          <Form.Item name="is_active" label={t('table.status')} valuePropName="checked">
            <Switch checkedChildren={t('common.enabled')} unCheckedChildren={t('common.disabled')} />
          </Form.Item>

          <Typography.Title level={5} style={{ marginTop: 8 }}>
            {t('payment_method.config_basic')}
          </Typography.Title>
          <Typography.Paragraph type="secondary" style={{ marginTop: -8 }}>
            {t('payment_method.config_basic_hint')}
          </Typography.Paragraph>

          {basicFields.map((field) => (
            <ConfigFieldItem key={field.key} field={field} t={t} />
          ))}

          <Collapse
            ghost
            items={[
              {
                key: 'advanced',
                label: (
                  <span>
                    {t('payment_method.advanced_settings')}
                    <Typography.Text type="secondary" style={{ marginLeft: 8, fontWeight: 400 }}>
                      {t('payment_method.advanced_settings_hint')}
                    </Typography.Text>
                  </span>
                ),
                children: (
                  <>
                    {advancedFields.map((field) => (
                      <ConfigFieldItem key={field.key} field={field} t={t} />
                    ))}
                    <Form.Item name="sort" label={t('table.sort')} extra={t('payment_method.sort_tip')}>
                      <InputNumber min={0} style={{ width: '100%' }} />
                    </Form.Item>
                    <Form.Item name="description" label={t('table.description')}>
                      <Input.TextArea rows={2} placeholder={t('payment_method.description_placeholder')} />
                    </Form.Item>
                  </>
                ),
              },
            ]}
          />

          <Form.Item name="code" hidden>
            <Input />
          </Form.Item>
          <Form.Item name="type" hidden>
            <Input />
          </Form.Item>
        </Form>
      )}
    </Modal>
  )
}
