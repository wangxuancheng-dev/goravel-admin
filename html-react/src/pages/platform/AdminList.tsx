import { useMemo, useState } from 'react'
import { Button, Form, Input, Modal, Select, Space, Table, Tag, message } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useTranslation } from 'react-i18next'
import {
  createPlatformAdmin,
  deletePlatformAdmin,
  getPlatformAdminList,
  resetPlatformAdminPassword,
  updatePlatformAdmin,
} from '@/api/platform'
import { useListPage } from '@/hooks/useListPage'
import { handlePaginatedTableChange } from '@/utils/tableChange'
import { entityField } from '@/utils/normalize'
import PageContainer from '@/components/PageContainer'
import SearchForm from '@/components/SearchForm'
import { getPlatformAdmin } from '@/utils/platformRequest'
import { useUnhandledError } from '@/hooks/useUnhandledError'

interface AdminRow {
  id: number | string
  username?: string
  name?: string
  role?: string
  status?: number | string
  created_at?: string
}

export default function PlatformAdminList() {
  const { t } = useTranslation()
  const showError = useUnhandledError()
  const me = getPlatformAdmin()
  const isOwner = me?.role !== 'viewer'
  const currentId = me?.id

  const {
    tableData,
    loading,
    pagination,
    searchForm,
    onSearchFormChange,
    loadData,
    handleSearch,
    handleReset,
    handleSortChange,
  } = useListPage<AdminRow>({
    fetchApi: getPlatformAdminList,
    initialSearchForm: { username: '', role: '', status: '' },
    defaultSort: 'id:desc',
    normalizeRows: false,
    transformData: (row) => ({
      id: entityField(row, 'id', '')!,
      username: String(entityField(row, 'username', '') ?? ''),
      name: String(entityField(row, 'name', '') ?? ''),
      role: String(entityField(row, 'role', '') ?? ''),
      status: entityField(row, 'status', 0) as number | string,
      created_at: String(entityField(row, 'created_at', '') ?? ''),
    }),
  })

  const [formOpen, setFormOpen] = useState(false)
  const [saving, setSaving] = useState(false)
  const [editing, setEditing] = useState<AdminRow | null>(null)
  const [form] = Form.useForm()
  const isSelfEdit = Boolean(editing && Number(editing.id) === Number(currentId))

  const [resetOpen, setResetOpen] = useState(false)
  const [resetSaving, setResetSaving] = useState(false)
  const [resetTarget, setResetTarget] = useState<AdminRow | null>(null)
  const [resetForm] = Form.useForm()

  const openCreate = () => {
    setEditing(null)
    form.setFieldsValue({ username: '', password: '', name: '', role: 'viewer', status: 1 })
    setFormOpen(true)
  }

  const openEdit = (row: AdminRow) => {
    setEditing(row)
    form.setFieldsValue({
      username: row.username,
      name: row.name,
      role: row.role === 'viewer' ? 'viewer' : 'owner',
      status: Number(row.status) === 1 ? 1 : 0,
    })
    setFormOpen(true)
  }

  const submitForm = async () => {
    try {
      const values = await form.validateFields()
      setSaving(true)
      if (editing) {
        const payload: { name?: string; role?: string; status?: number } = { name: values.name }
        if (!isSelfEdit) {
          payload.role = values.role
          payload.status = values.status
        }
        await updatePlatformAdmin(editing.id, payload)
      } else {
        await createPlatformAdmin({
          username: values.username,
          password: values.password,
          name: values.name,
          role: values.role,
        })
      }
      message.success(t('common.success'))
      setFormOpen(false)
      loadData()
    } catch (e) {
      if (e && typeof e === 'object' && 'errorFields' in e) return
      showError(e, t('menu.platform_admin'))
    } finally {
      setSaving(false)
    }
  }

  const onDelete = (row: AdminRow) => {
    Modal.confirm({
      title: t('common.warning'),
      content: t('platform_admin.delete_confirm'),
      onOk: async () => {
        try {
          await deletePlatformAdmin(row.id)
          message.success(t('common.success'))
          loadData()
        } catch (e) {
          showError(e, t('common.delete'))
        }
      },
    })
  }

  const openReset = (row: AdminRow) => {
    setResetTarget(row)
    resetForm.resetFields()
    setResetOpen(true)
  }

  const submitReset = async () => {
    try {
      const values = await resetForm.validateFields()
      if (!resetTarget) return
      setResetSaving(true)
      await resetPlatformAdminPassword(resetTarget.id, {
        password: values.password,
        confirm_password: values.confirm_password,
      })
      message.success(t('common.success'))
      setResetOpen(false)
    } catch (e) {
      if (e && typeof e === 'object' && 'errorFields' in e) return
      showError(e, t('platform_admin.reset_password'))
    } finally {
      setResetSaving(false)
    }
  }

  const columns: ColumnsType<AdminRow> = useMemo(
    () => [
      { title: t('table.id'), dataIndex: 'id', width: 70, sorter: true },
      { title: t('platform_admin.username'), dataIndex: 'username', width: 140 },
      { title: t('platform_admin.name'), dataIndex: 'name', width: 140 },
      {
        title: t('platform_admin.role'),
        dataIndex: 'role',
        width: 110,
        render: (v: string) => (
          <Tag color={v === 'viewer' ? 'default' : 'success'}>
            {v === 'viewer' ? t('platform.role_viewer') : t('platform.role_owner')}
          </Tag>
        ),
      },
      {
        title: t('table.status'),
        dataIndex: 'status',
        width: 100,
        render: (v) => (
          <Tag color={Number(v) === 1 ? 'success' : 'error'}>
            {Number(v) === 1 ? t('common.enabled') : t('common.disabled')}
          </Tag>
        ),
      },
      { title: t('common.created_at'), dataIndex: 'created_at', width: 170 },
      {
        title: t('table.operation') || t('common.operation'),
        key: 'action',
        width: 240,
        fixed: 'right' as const,
        render: (_, row) =>
          isOwner ? (
            <Space size="small">
              <Button type="link" onClick={() => openEdit(row)}>
                {t('common.edit')}
              </Button>
              <Button type="link" onClick={() => openReset(row)}>
                {t('platform_admin.reset_password')}
              </Button>
              <Button
                type="link"
                danger
                disabled={Number(row.id) === Number(currentId)}
                onClick={() => onDelete(row)}
              >
                {t('common.delete')}
              </Button>
            </Space>
          ) : (
            '—'
          ),
      },
    ],
    [t, isOwner, currentId],
  )

  return (
    <PageContainer
      title={t('menu.platform_admin')}
      extra={
        isOwner ? (
          <Button type="primary" onClick={openCreate}>
            {t('common.add')}
          </Button>
        ) : null
      }
    >
      <SearchForm
        fields={[
          { name: 'username', label: t('platform_admin.username'), type: 'input' },
          {
            name: 'role',
            label: t('platform_admin.role'),
            type: 'select',
            options: [
              { label: t('platform.role_owner'), value: 'owner' },
              { label: t('platform.role_viewer'), value: 'viewer' },
            ],
          },
          {
            name: 'status',
            label: t('table.status'),
            type: 'select',
            options: [
              { label: t('common.enabled'), value: '1' },
              { label: t('common.disabled'), value: '0' },
            ],
          },
        ]}
        values={searchForm}
        onChange={onSearchFormChange}
        onSearch={handleSearch}
        onReset={handleReset}
      />
      <Table
        rowKey="id"
        loading={loading}
        columns={columns}
        dataSource={tableData}
        pagination={{
          current: pagination.page,
          pageSize: pagination.pageSize,
          total: pagination.total,
          showSizeChanger: true,
        }}
        scroll={{ x: 1000 }}
        onChange={(pager, _f, sorter) => {
          handlePaginatedTableChange({ pager, sorter, pagination, loadData, handleSortChange })
        }}
      />

      <Modal
        title={editing ? t('platform_admin.edit') : t('platform_admin.create')}
        open={formOpen}
        onCancel={() => setFormOpen(false)}
        onOk={() => void submitForm()}
        confirmLoading={saving}
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          {!editing ? (
            <>
              <Form.Item
                name="username"
                label={t('platform_admin.username')}
                rules={[{ required: true, message: t('platform_admin.username') }]}
              >
                <Input autoComplete="off" />
              </Form.Item>
              <Form.Item
                name="password"
                label={t('platform_admin.password')}
                rules={[
                  { required: true, message: t('platform_admin.password') },
                  { min: 6, message: t('platform_admin.password_min') },
                ]}
              >
                <Input.Password autoComplete="new-password" />
              </Form.Item>
            </>
          ) : null}
          <Form.Item name="name" label={t('platform_admin.name')}>
            <Input />
          </Form.Item>
          <Form.Item
            name="role"
            label={t('platform_admin.role')}
            rules={[{ required: true, message: t('platform_admin.role') }]}
          >
            <Select disabled={isSelfEdit}>
              <Select.Option value="owner">{t('platform.role_owner')}</Select.Option>
              <Select.Option value="viewer">{t('platform.role_viewer')}</Select.Option>
            </Select>
          </Form.Item>
          {editing ? (
            <Form.Item name="status" label={t('table.status')}>
              <Select disabled={isSelfEdit}>
                <Select.Option value={1}>{t('common.enabled')}</Select.Option>
                <Select.Option value={0}>{t('common.disabled')}</Select.Option>
              </Select>
            </Form.Item>
          ) : null}
          {isSelfEdit ? <div style={{ color: 'var(--ant-color-text-secondary, #64748b)' }}>{t('platform_admin.self_edit_hint')}</div> : null}
        </Form>
      </Modal>

      <Modal
        title={t('platform_admin.reset_password')}
        open={resetOpen}
        onCancel={() => setResetOpen(false)}
        onOk={() => void submitReset()}
        confirmLoading={resetSaving}
        destroyOnClose
      >
        <Form form={resetForm} layout="vertical">
          <Form.Item
            name="password"
            label={t('platform_admin.password')}
            rules={[
              { required: true, message: t('platform_admin.password') },
              { min: 6, message: t('platform_admin.password_min') },
            ]}
          >
            <Input.Password />
          </Form.Item>
          <Form.Item
            name="confirm_password"
            label={t('platform.confirm_password')}
            dependencies={['password']}
            rules={[
              { required: true, message: t('platform.confirm_password') },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('password') === value) return Promise.resolve()
                  return Promise.reject(new Error(t('platform.confirm_password')))
                },
              }),
            ]}
          >
            <Input.Password />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  )
}
