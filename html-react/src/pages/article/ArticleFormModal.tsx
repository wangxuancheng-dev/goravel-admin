import { useEffect, useState } from 'react'
import { App, Form, Input, Modal, Select, Radio } from 'antd'
import { useTranslation } from 'react-i18next'
import {
  createArticle,
  getArticleDetail,
  updateArticle,
} from '@/api/article'
import { useUnhandledError } from '@/hooks/useUnhandledError'
import { entityField } from '@/utils/normalize'

import WangEditor from '@/components/WangEditor'

interface ArticleFormModalProps {
  open: boolean
  editId?: string | number | null
  onClose: () => void
  onSuccess: () => void
}

export default function ArticleFormModal({ open, editId, onClose, onSuccess }: ArticleFormModalProps) {
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

        admin_id: '',
        title: '',
        content: '',
        status: '',
      })
      return
    }

    setLoading(true)
    getArticleDetail(editId)
      .then((res) => {
        const raw = (res.data || {}) as Record<string, unknown>
        const data = (entityField(raw, 'article', raw) || {}) as Record<string, unknown>
        form.setFieldsValue({

          admin_id: entityField(data, 'admin_id', ''),
          title: entityField(data, 'title', ''),
          content: entityField(data, 'content', ''),
          status: entityField(data, 'status', ''),
        })
      })
      .catch((error) => showError(error, t('common.query_failed')))
      .finally(() => setLoading(false))
  }, [open, editId, form, showError, t])

  const handleOk = async () => {
    try {
      const values = await form.validateFields()
      setSubmitting(true)
      const payload = {
        ...values,

      }
      if (editId) {
        
        await updateArticle(editId, payload)
        message.success(t('common.update_success'))
        
      } else {
        
        await createArticle(payload)
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
      title={editId ? t('article.edit', { defaultValue: t('common.edit') }) : t('article.add', { defaultValue: t('common.add') })}
      onCancel={onClose}
      onOk={() => void handleOk()}
      confirmLoading={submitting}
      destroyOnHidden
      width={800}
    >
      <Form form={form} layout="vertical" disabled={loading}>

        <Form.Item name="admin_id" label={t('admin_id', { defaultValue: '管理员ID' })} rules={[{ required: true }]}>
          <Select allowClear />
        </Form.Item>
        
        <Form.Item name="title" label={t('title', { defaultValue: '标题' })} rules={[{ required: true }]}>
          <Input />
        </Form.Item>
        
        <Form.Item name="content" label={t('content', { defaultValue: '内容' })}>
          <WangEditor height={400} placeholder={t('content', { defaultValue: '内容' })} />
        </Form.Item>
        
        <Form.Item name="status" label={t('status', { defaultValue: '0:未发布 1:发布' })} rules={[{ required: true }]}>
          <Radio.Group />
        </Form.Item>
        
      </Form>
    </Modal>
  )
}
