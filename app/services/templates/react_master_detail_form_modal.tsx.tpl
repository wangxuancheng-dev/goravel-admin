import { useEffect, useState } from 'react'
import { App, Button, Form, Input, InputNumber, Modal, Space, Table } from 'antd'
import { MinusCircleOutlined, PlusOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import {
  <<if .HasCreate>>create<<.ModelName>>,<<end>>
  get<<.ModelName>>Detail,
  <<if .HasEdit>>update<<.ModelName>>,<<end>>
} from '@/api/<<.ModuleNameK>>'
import { useUnhandledError } from '@/hooks/useUnhandledError'
import { entityField } from '@/utils/normalize'

interface <<.ModelName>>FormModalProps {
  open: boolean
  editId?: string | number | null
  onClose: () => void
  onSuccess: () => void
}

export default function <<.ModelName>>FormModal({ open, editId, onClose, onSuccess }: <<.ModelName>>FormModalProps) {
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
<<range .FormFields>>
<<- if and .ShowInForm (ne .Name "id") (ne .Name "created_at") (ne .Name "updated_at") (ne .Name "deleted_at")>>
        <<.Name>>: <<if eq .FormType "switch">>true<<else if eq .FormType "number">>0<<else>>''<<end>>,
<<- end>>
<<- end>>
        details: [{}],
      })
      return
    }

    setLoading(true)
    get<<.ModelName>>Detail(editId)
      .then((res) => {
        const raw = (res.data || {}) as Record<string, unknown>
        const data = (entityField(raw, '<<.ModuleName>>', raw) || {}) as Record<string, unknown>
        const detailsRaw = (entityField(data, 'details', []) || []) as Array<Record<string, unknown>>
        form.setFieldsValue({
<<range .FormFields>>
<<- if and .ShowInForm (ne .Name "id") (ne .Name "created_at") (ne .Name "updated_at") (ne .Name "deleted_at")>>
          <<.Name>>: <<if eq .FormType "number">>Number(entityField(data, '<<.Name>>', 0))<<else>>entityField(data, '<<.Name>>', '')<<end>>,
<<- end>>
<<- end>>
          details: detailsRaw.length
            ? detailsRaw.map((row) => ({
<<- range .DetailFormFields>>
                <<.Name>>: <<if eq .FormType "number">>Number(entityField(row, '<<.Name>>', 0))<<else>>entityField(row, '<<.Name>>', '')<<end>>,
<<- end>>
              }))
            : [{}],
        })
      })
      .catch((error) => showError(error, t('common.query_failed')))
      .finally(() => setLoading(false))
  }, [open, editId, form, showError, t])

  const isBlankDetailRow = (row: Record<string, unknown> | null | undefined) => {
    if (!row || typeof row !== 'object') return true
    return !Object.values(row).some((v) => v !== undefined && v !== null && String(v).trim() !== '')
  }

  const handleOk = async () => {
    try {
      // Validate master fields only; Form.List placeholder rows must not block submit.
      const masterNames = [
<<- range .FormFields>>
<<- if and .ShowInForm (ne .Name "id") (ne .Name "created_at") (ne .Name "updated_at") (ne .Name "deleted_at")>>
        '<<.Name>>',
<<- end>>
<<- end>>
      ]
      const values = await form.validateFields(masterNames)
      const rawDetails = (form.getFieldValue('details') || []) as Array<Record<string, unknown>>
      const details = rawDetails.filter((row) => !isBlankDetailRow(row))
<<- range .DetailFormFields>>
<<- if .Required>>
      for (let i = 0; i < details.length; i++) {
        const v = details[i]['<<.Name>>']
        if (v === undefined || v === null || String(v).trim() === '') {
          message.warning(`${t('common.required', { defaultValue: 'Required' })}: ${t('<<.Name>>', { defaultValue: '<<.Label>>' })}`)
          return
        }
      }
<<- end>>
<<- end>>
      setSubmitting(true)
      const payload = {
        ...values,
        details,
      }
      if (editId) {
        <<if .HasEdit>>
        await update<<.ModelName>>(editId, payload)
        message.success(t('common.update_success'))
        <<end>>
      } else {
        <<if .HasCreate>>
        await create<<.ModelName>>(payload)
        message.success(t('common.create_success'))
        <<end>>
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
      title={editId ? t('<<.ModuleName>>.edit', { defaultValue: t('common.edit') }) : t('<<.ModuleName>>.add', { defaultValue: t('common.add') })}
      onCancel={onClose}
      onOk={() => void handleOk()}
      confirmLoading={submitting}
      destroyOnHidden
      width={900}
    >
      <Form form={form} layout="vertical" disabled={loading}>
<<range .FormFields>>
<<- if and .ShowInForm (ne .Name "id") (ne .Name "created_at") (ne .Name "updated_at") (ne .Name "deleted_at")>>
        <<if eq .FormType "textarea">>
        <Form.Item name="<<.Name>>" label={t('<<.Name>>', { defaultValue: '<<.Label>>' })}<<if .Required>> rules={[{ required: true }]}<<end>>>
          <Input.TextArea rows={3} />
        </Form.Item>
        <<else if eq .FormType "number">>
        <Form.Item name="<<.Name>>" label={t('<<.Name>>', { defaultValue: '<<.Label>>' })}<<if .Required>> rules={[{ required: true }]}<<end>>>
          <InputNumber style={{ width: '100%' }} />
        </Form.Item>
        <<else>>
        <Form.Item name="<<.Name>>" label={t('<<.Name>>', { defaultValue: '<<.Label>>' })}<<if .Required>> rules={[{ required: true }]}<<end>>>
          <Input />
        </Form.Item>
        <<end>>
<<- end>>
<<- end>>

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
<<- range .DetailFormFields>>
                  {
                    title: t('<<.Name>>', { defaultValue: '<<.Label>>' }),
                    dataIndex: '<<.Name>>',
                    render: (_: unknown, field: { name: number }) => (
                      <Form.Item
                        name={[field.name, '<<.Name>>']}
                        style={{ marginBottom: 0 }}
                      >
                        <<if eq .FormType "number">>
                        <InputNumber style={{ width: '100%' }} />
                        <<else>>
                        <Input />
                        <<end>>
                      </Form.Item>
                    ),
                  },
<<- end>>
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
