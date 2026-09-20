import { useEffect, useMemo, useState } from 'react'
import { App, Button, Form, Input, InputNumber, Modal, Space, Table } from 'antd'
import { PlusOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { createOrder } from '@/api/order'
import { useUnhandledError } from '@/hooks/useUnhandledError'
import { formatAmount } from './order.config'

interface ProductLine {
  key: string
  product_id: number | null
  product_name: string
  price: number
  quantity: number
}

interface OrderFormModalProps {
  open: boolean
  onClose: () => void
  onSuccess: () => void
}

function emptyProduct(): ProductLine {
  return {
    key: `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
    product_id: null,
    product_name: '',
    price: 0,
    quantity: 1,
  }
}

export default function OrderFormModal({ open, onClose, onSuccess }: OrderFormModalProps) {
  const { t } = useTranslation()
  const { message } = App.useApp()
  const showError = useUnhandledError()
  const [form] = Form.useForm()
  const [submitting, setSubmitting] = useState(false)
  const [products, setProducts] = useState<ProductLine[]>([emptyProduct()])

  useEffect(() => {
    if (open) {
      form.resetFields()
      form.setFieldsValue({ user_id: undefined, remark: '', expire_in_seconds: 60 })
      setProducts([emptyProduct()])
    }
  }, [open, form])

  const totalAmount = useMemo(
    () => products.reduce((sum, p) => sum + (Number(p.price) || 0) * (Number(p.quantity) || 0), 0),
    [products],
  )

  const updateProduct = (key: string, patch: Partial<ProductLine>) => {
    setProducts((prev) => prev.map((p) => (p.key === key ? { ...p, ...patch } : p)))
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      if (!products.length) {
        message.warning(t('order.products_required'))
        return
      }
      for (let i = 0; i < products.length; i += 1) {
        const product = products[i]
        if (!product.product_id || product.product_id <= 0) {
          message.warning(t('order.product_id_required', { index: i + 1 }))
          return
        }
        if (!product.product_name?.trim()) {
          message.warning(t('order.product_name_required', { index: i + 1 }))
          return
        }
        if (!product.price || product.price <= 0) {
          message.warning(t('order.price_required', { index: i + 1 }))
          return
        }
        if (!product.quantity || product.quantity <= 0) {
          message.warning(t('order.quantity_required', { index: i + 1 }))
          return
        }
      }
      if (totalAmount <= 0) {
        message.warning(t('order.amount_required'))
        return
      }

      setSubmitting(true)
      const payload: Record<string, unknown> = {
        user_id: values.user_id,
        amount: totalAmount,
        products: products.map((p) => ({
          product_id: p.product_id,
          product_name: p.product_name,
          price: p.price,
          quantity: p.quantity,
        })),
        remark: values.remark || '',
        expire_in_seconds: Number(values.expire_in_seconds) > 0 ? Number(values.expire_in_seconds) : 0,
      }
      await createOrder(payload)
      message.success(t('order.create_success'))
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
      title={t('order.add_order')}
      width={1000}
      destroyOnHidden
      onCancel={onClose}
      onOk={() => void handleSubmit()}
      confirmLoading={submitting}
      okText={t('common.confirm')}
      cancelText={t('common.cancel')}
    >
      <Form form={form} layout="vertical">
        <Form.Item
          name="user_id"
          label={t('order.user_id')}
          rules={[
            { required: true, message: t('order.user_id_required') },
            {
              type: 'number',
              min: 1,
              message: t('order.user_id_min'),
            },
          ]}
        >
          <InputNumber min={1} style={{ width: 220 }} placeholder={t('order.user_id')} />
        </Form.Item>

        <Form.Item label={t('order.products')} required>
          <Space direction="vertical" style={{ width: '100%' }} size="middle">
            <Button type="primary" size="small" icon={<PlusOutlined />} onClick={() => setProducts((p) => [...p, emptyProduct()])}>
              {t('order.add_product')}
            </Button>
            <Table<ProductLine>
              rowKey="key"
              size="small"
              pagination={false}
              dataSource={products}
              scroll={{ x: 700 }}
              columns={[
                {
                  title: t('order.product_id'),
                  width: 130,
                  render: (_, row) => (
                    <InputNumber
                      min={1}
                      controls={false}
                      style={{ width: '100%' }}
                      value={row.product_id ?? undefined}
                      onChange={(v) => updateProduct(row.key, { product_id: v ?? null })}
                    />
                  ),
                },
                {
                  title: t('order.product_name'),
                  render: (_, row) => (
                    <Input
                      allowClear
                      value={row.product_name}
                      placeholder={t('order.product_name_placeholder')}
                      onChange={(e) => updateProduct(row.key, { product_name: e.target.value })}
                    />
                  ),
                },
                {
                  title: t('order.price'),
                  width: 140,
                  render: (_, row) => (
                    <InputNumber
                      min={0}
                      precision={2}
                      controls={false}
                      style={{ width: '100%' }}
                      value={row.price}
                      onChange={(v) => updateProduct(row.key, { price: Number(v) || 0 })}
                    />
                  ),
                },
                {
                  title: t('order.quantity'),
                  width: 120,
                  render: (_, row) => (
                    <InputNumber
                      min={1}
                      controls={false}
                      style={{ width: '100%' }}
                      value={row.quantity}
                      onChange={(v) => updateProduct(row.key, { quantity: Number(v) || 1 })}
                    />
                  ),
                },
                {
                  title: t('order.subtotal'),
                  width: 120,
                  align: 'right',
                  render: (_, row) => (
                    <span style={{ fontWeight: 600 }}>{formatAmount((row.price || 0) * (row.quantity || 0))}</span>
                  ),
                },
                {
                  title: t('common.operation'),
                  width: 90,
                  fixed: 'end',
                  render: (_, row) => (
                    <Button
                      type="link"
                      danger
                      disabled={products.length <= 1}
                      onClick={() => setProducts((prev) => prev.filter((p) => p.key !== row.key))}
                    >
                      {t('common.delete')}
                    </Button>
                  ),
                },
              ]}
            />
          </Space>
        </Form.Item>

        <Form.Item label={t('order.amount')}>
          <Input value={formatAmount(totalAmount)} disabled />
        </Form.Item>

        <Form.Item
          name="expire_in_seconds"
          label={t('order.expire_in_seconds')}
          extra={t('order.expire_in_seconds_tip')}
          initialValue={60}
        >
          <InputNumber
            min={1}
            max={3600}
            style={{ width: 220 }}
            placeholder="60"
          />
        </Form.Item>

        <Form.Item name="remark" label={t('order.remark')}>
          <Input.TextArea rows={3} placeholder={t('order.remark_placeholder')} />
        </Form.Item>
      </Form>
    </Modal>
  )
}
