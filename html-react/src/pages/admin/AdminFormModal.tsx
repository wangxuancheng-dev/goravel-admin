import { useEffect, useMemo, useState } from 'react'
import { App, Form, Input, Modal, Radio, Select, TreeSelect } from 'antd'
import { useTranslation } from 'react-i18next'
import { createAdmin, getAdminDetail, updateAdmin } from '@/api/admin'
import { normalizeOptions, toOptionTreeData, useOptions } from '@/hooks/useOptions'
import { getOptions, type OptionItem } from '@/api/option'
import { useUnhandledError } from '@/hooks/useUnhandledError'
import { entityField } from '@/utils/normalize'

interface AdminFormModalProps {
  open: boolean
  editId?: string | number | null
  onClose: () => void
  onSuccess: () => void
}

export default function AdminFormModal({ open, editId, onClose, onSuccess }: AdminFormModalProps) {
  const { t } = useTranslation()
  const { message } = App.useApp()
  const showError = useUnhandledError()
  const [form] = Form.useForm()
  const [loading, setLoading] = useState(false)
  const [submitting, setSubmitting] = useState(false)
  const [isSuperAdmin, setIsSuperAdmin] = useState(false)
  const { selectOptions: roleOptions } = useOptions('role', open)
  const { selectOptions: positionOptions } = useOptions('position', open)
  const [deptTree, setDeptTree] = useState<OptionItem[]>([])

  useEffect(() => {
    if (!open) return
    getOptions('department')
      .then((res) => setDeptTree(normalizeOptions(res.data)))
      .catch(() => setDeptTree([]))
  }, [open])

  const deptTreeData = useMemo(() => toOptionTreeData(deptTree), [deptTree])

  useEffect(() => {
    if (!open) return

    if (!editId) {
      setIsSuperAdmin(false)
      form.setFieldsValue({
        username: '',
        password: '',
        nickname: '',
        email: '',
        phone: '',
        department_id: undefined,
        position_id: undefined,
        role_ids: [],
        status: 1,
      })
      return
    }

    setLoading(true)
    getAdminDetail(editId)
      .then((res) => {
        const raw = (res.data || {}) as Record<string, unknown>
        const data = (entityField(raw, 'admin', raw) || {}) as Record<string, unknown>
        const roles = (entityField(data, 'roles', []) as Array<Record<string, unknown>>) || []
        const deptId = Number(entityField(data, 'department_id', 0) || 0)
        const posId = Number(entityField(data, 'position_id', 0) || 0)
        const superAdminFlag = entityField(data, 'is_super_admin', false)
        const superAdmin =
          Boolean(superAdminFlag) ||
          roles.some((r) => String(entityField(r, 'slug', '')) === 'super-admin')
        setIsSuperAdmin(superAdmin)
        form.setFieldsValue({
          username: entityField(data, 'username', ''),
          nickname: entityField(data, 'nickname', ''),
          email: entityField(data, 'email', ''),
          phone: entityField(data, 'phone', ''),
          status: Number(entityField(data, 'status', 1)),
          department_id: deptId || undefined,
          position_id: posId || undefined,
          role_ids: roles.map((role) => entityField(role, 'id')).filter((id) => id != null && id !== ''),
        })
      })
      .catch((error) => showError(error, t('common.query_failed')))
      .finally(() => setLoading(false))
  }, [open, editId, form, showError, t])

  const handleOk = async () => {
    try {
      const values = await form.validateFields()
      setSubmitting(true)
      const payload: Record<string, unknown> = {
        ...values,
        status: Number(values.status),
        role_ids: values.role_ids || [],
        department_id: values.department_id ?? 0,
        position_id: values.position_id ?? 0,
      }
      if (editId) {
        delete payload.password
        await updateAdmin(editId, payload)
        message.success(t('common.update_success'))
      } else {
        await createAdmin(payload)
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
      title={editId ? t('admin.edit_admin') : t('admin.add_admin')}
      onCancel={onClose}
      onOk={() => void handleOk()}
      confirmLoading={submitting}
      destroyOnHidden
      width={600}
    >
      <Form form={form} layout="vertical" disabled={loading}>
        <Form.Item
          name="username"
          label={t('table.username')}
          rules={[{ required: true, message: t('admin.username_required') }]}
        >
          <Input disabled={!!editId} />
        </Form.Item>
        {!editId && (
          <Form.Item
            name="password"
            label={t('common.password')}
            rules={[{ required: true, message: t('admin.password_required') }]}
          >
            <Input.Password />
          </Form.Item>
        )}
        <Form.Item name="nickname" label={t('table.nickname')}>
          <Input />
        </Form.Item>
        <Form.Item name="email" label={t('table.email')}>
          <Input />
        </Form.Item>
        <Form.Item name="phone" label={t('table.phone')}>
          <Input />
        </Form.Item>
        <Form.Item name="department_id" label={t('table.department')}>
          <TreeSelect
            allowClear
            treeData={deptTreeData}
            disabled={isSuperAdmin}
            placeholder={t('table.department')}
            treeDefaultExpandAll
          />
        </Form.Item>
        <Form.Item name="position_id" label={t('table.position')}>
          <Select
            allowClear
            options={positionOptions}
            disabled={isSuperAdmin}
            optionFilterProp="label"
            showSearch
          />
        </Form.Item>
        <Form.Item name="role_ids" label={t('admin.roles')}>
          <Select
            mode="multiple"
            allowClear
            options={roleOptions}
            disabled={isSuperAdmin}
            optionFilterProp="label"
          />
        </Form.Item>
        <Form.Item name="status" label={t('common.status')}>
          <Radio.Group
            disabled={isSuperAdmin}
            options={[
              { label: t('common.enabled'), value: 1 },
              { label: t('common.disabled'), value: 0 },
            ]}
          />
        </Form.Item>
      </Form>
    </Modal>
  )
}
