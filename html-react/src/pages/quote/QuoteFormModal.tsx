import { useEffect, useState } from 'react'
import { App, Button, Form, Input, InputNumber, Modal, Space, Table } from 'antd'
import { MinusCircleOutlined, PlusOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import {
  createQuote,
  getQuoteDetail,
  updateQuote,
} from '@/api/quote'
import { useUnhandledError } from '@/hooks/useUnhandledError'
import { entityField } from '@/utils/normalize'

interface QuoteFormModalProps {
  open: boolean
  editId?: string | number | null
  onClose: () => void
  onSuccess: () => void
}

export default function QuoteFormModal({ open, editId, onClose, onSuccess }: QuoteFormModalProps) {
  const { t } = useTranslation()
  const { message } = App.useApp()
  const showError = useUnhandledError()
  const [form] = Form.useForm()
  const [loading, setLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    if (!open) return

    if (!editId) {
      form.setFieldsValue({

        quote_no: '',
        customer_name: '',
        status: 0,
        remark: '',
        details: [{}],
      })
      return
    }

    setLoading(true)
    getQuoteDetail(editId)
      .then((res) => {
        const raw = (res.data || {}) as Record<string, unknown>
        const data = (entityField(raw, 'quote', raw) || {}) as Record<string, unknown>
        const detailsRaw = (entityField(data, 'details', []) || []) as Array<Record<string, unknown>>
        form.setFieldsValue({

          quote_no: entityField(data, 'quote_no', ''),
          customer_name: entityField(data, 'customer_name', ''),
          status: Number(entityField(data, 'status', 0)),
          remark: entityField(data, 'remark', ''),
          details: detailsRaw.length
            ? detailsRaw.map((row) => ({
                product_name: entityField(row, 'product_name', ''),
                quantity: Number(entityField(row, 'quantity', 0)),
                unit_price: entityField(row, 'unit_price', ''),
              }))
            : [{}],
        })
      })
      .catch((error) => showError(error, t('common.query_failed')))
      .finally(() => setLoading(false))
  }, [open, editId, form, showError, t])

  const handleOk = async () => {
    try {
      // Drop blank placeholder rows before validation so required detail rules do not block submit.
      const rawDetails = (form.getFieldValue('details') || []) as Array<Record<string, unknown>>
      const details = rawDetails.filter((row) => {
        if (!row || typeof row !== 'object') return false
        return Object.values(row).some((v) => v !== undefined && v !== null && String(v).trim() !== '')
      })
      form.setFieldsValue({ details })
      const values = await form.validateFields()
      setSubmitting(true)
      const payload = {
        ...values,
        details,
      }
      if (editId) {
        
        await updateQuote(editId, payload)
        message.success(t('common.update_success'))
        
      } else {
        
        await createQuote(payload)
        message.success(t('common.create_success'))
        
      }
      onSuccess()
      onClose()
    } catch (error) {
      if ((error as { errorFields?: unknown })?.errorFields) return
      showError(error, t('common.operation_failed'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Modal
      open={open}
      title={editId ? t('quote.edit', { defaultValue: t('common.edit') }) : t('quote.add', { defaultValue: t('common.add') })}
      onCancel={onClose}
      onOk={() => void handleOk()}
      confirmLoading={submitting}
      destroyOnHidden
      width={900}
    >
      <Form form={form} layout="vertical" disabled={loading}>

        <Form.Item name="quote_no" label={t('quote_no', { defaultValue: 'quote number' })} rules={[{ required: true }]}>
          <Input />
        </Form.Item>
        
        <Form.Item name="customer_name" label={t('customer_name', { defaultValue: 'customer name' })} rules={[{ required: true }]}>
          <Input />
        </Form.Item>
        
        <Form.Item name="status" label={t('status', { defaultValue: '0 draft 1 confirmed' })} rules={[{ required: true }]}>
          <InputNumber style={{ width: '100%' }} />
        </Form.Item>
        
        <Form.Item name="remark" label={t('remark', { defaultValue: 'remark' })}>
          <Input />
        </Form.Item>
        
        <Form.List name="details">
          {(fields, { add, remove }) => (
            <>
              <Space style={{ marginBottom: 8 }} align="center">
                <strong>{t('common.detail_rows', { defaultValue: 'Detail rows' })}</strong>
                <Button type="dashed" icon={<PlusOutlined />} onClick={() => add({})}>
                  {t('common.add', { defaultValue: 'Add' })}
                </Button>
              </Space>
              <Table
                size="small"
                pagination={false}
                rowKey="key"
                dataSource={fields.map((field) => ({ ...field }))}
                columns={[
                  {
                    title: t('product_name', { defaultValue: 'product name' }),
                    dataIndex: 'product_name',
                    render: (_: unknown, field: { name: number }) => (
                      <Form.Item
                        name={[field.name, 'product_name']}
                        style={{ marginBottom: 0 }}
                        rules={[{ required: true }]}
                      >
                        
                        <Input />
                        
                      </Form.Item>
                    ),
                  },
                  {
                    title: t('quantity', { defaultValue: 'qty' }),
                    dataIndex: 'quantity',
                    render: (_: unknown, field: { name: number }) => (
                      <Form.Item
                        name={[field.name, 'quantity']}
                        style={{ marginBottom: 0 }}
                        rules={[{ required: true }]}
                      >
                        
                        <InputNumber style={{ width: '100%' }} />
                        
                      </Form.Item>
                    ),
                  },
                  {
                    title: t('unit_price', { defaultValue: 'unit price' }),
                    dataIndex: 'unit_price',
                    render: (_: unknown, field: { name: number }) => (
                      <Form.Item
                        name={[field.name, 'unit_price']}
                        style={{ marginBottom: 0 }}
                        rules={[{ required: true }]}
                      >
                        
                        <Input />
                        
                      </Form.Item>
                    ),
                  },
                  {
                    title: t('common.operation', { defaultValue: 'Actions' }),
                    width: 64,
                    render: (_: unknown, field: { name: number }) => (
                      <MinusCircleOutlined onClick={() => remove(field.name)} />
                    ),
                  },
                ]}
              />
            </>
          )}
        </Form.List>
      </Form>
    </Modal>
  )
}
