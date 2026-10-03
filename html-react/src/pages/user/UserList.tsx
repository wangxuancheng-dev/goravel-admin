import { useState } from 'react'
import { App, Button, Dropdown, Form, Input, InputNumber, Modal, Radio, Select, Space, Switch, Table } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import type { MenuProps } from 'antd'
import { DownOutlined, SettingOutlined } from '@ant-design/icons'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  createUser,
  deleteUser,
  exportUsers,
  getUserDetail,
  getUserList,
  resetUserPassword,
  updateBalance,
  updateUser,
} from '@/api/user'
import { useListPage } from '@/hooks/useListPage'
import { handlePaginatedTableChange } from '@/utils/tableChange'
import { useCrudActions } from '@/hooks/useCrudActions'
import { usePermission } from '@/hooks/usePermission'
import { useUnhandledError } from '@/hooks/useUnhandledError'
import { useColumnSetting } from '@/hooks/useColumnSetting'
import PageContainer from '@/components/PageContainer'
import ColumnSettingDialog from '@/components/ColumnSettingDialog'
import SearchForm from '@/components/SearchForm'
import PermissionButton from '@/components/PermissionButton'
import { entityField } from '@/utils/normalize'
import {
  formatUserBalance,
  transformUserRow,
  userInitialSearchForm,
  type UserRow,
} from './user.config'

export default function UserList() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { message, modal } = App.useApp()
  const showError = useUnhandledError()
  const { getButtonState } = usePermission()
  const [form] = Form.useForm()
  const [pwdForm] = Form.useForm()
  const [balanceForm] = Form.useForm()
  const [formOpen, setFormOpen] = useState(false)
  const [editId, setEditId] = useState<string | number | null>(null)
  const [submitting, setSubmitting] = useState(false)
  const [loadingDetail, setLoadingDetail] = useState(false)
  const [exporting, setExporting] = useState(false)
  const [balanceOpen, setBalanceOpen] = useState(false)
  const [balanceUser, setBalanceUser] = useState<UserRow | null>(null)

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
    refresh,
  } = useListPage<UserRow, typeof userInitialSearchForm>({
    fetchApi: getUserList,
    initialSearchForm: userInitialSearchForm,
    defaultSort: 'id:desc',
    transformData: transformUserRow,
  })

  const { toolbar, confirmDelete } = useCrudActions({
    createPermission: 'user.store',
    onRefresh: refresh,
    onCreate: () => {
      setEditId(null)
      form.setFieldsValue({
        username: '',
        password: '',
        nickname: '',
        email: '',
        phone: '',
        status: 1,
      })
      setFormOpen(true)
    },
    deleteApi: deleteUser,
  })

  const openEdit = async (row: UserRow) => {
    setEditId(row.id)
    setFormOpen(true)
    setLoadingDetail(true)
    try {
      const res = await getUserDetail(row.id)
      const raw = (res.data || {}) as Record<string, unknown>
      const data = (entityField(raw, 'user', raw) || {}) as Record<string, unknown>
      form.setFieldsValue({
        username: entityField(data, 'username', ''),
        nickname: entityField(data, 'nickname', ''),
        email: entityField(data, 'email', ''),
        phone: entityField(data, 'phone', ''),
        status: Number(entityField(data, 'status', 1)),
      })
    } catch (error) {
      showError(error, t('common.query_failed'))
    } finally {
      setLoadingDetail(false)
    }
  }

  const handleStatusChange = async (row: UserRow, checked: boolean) => {
    try {
      await updateUser(row.id, { status: checked ? 1 : 0 })
      message.success(t('common.update_success'))
      await refresh()
    } catch (error) {
      showError(error, t('common.operation_failed'))
    }
  }

  const handleResetPassword = (row: UserRow) => {
    pwdForm.resetFields()
    modal.confirm({
      title: t('user.reset_password'),
      content: (
        <Form form={pwdForm} layout="vertical" style={{ marginTop: 16 }}>
          <Form.Item
            name="password"
            label={t('user.new_password')}
            rules={[
              { required: true },
              { min: 6, message: t('user.password_min_length') },
            ]}
          >
            <Input.Password />
          </Form.Item>
        </Form>
      ),
      onOk: async () => {
        const values = await pwdForm.validateFields()
        try {
          await resetUserPassword(row.id, { password: values.password })
          message.success(t('user.reset_password_success'))
        } catch (error) {
          showError(error, t('common.operation_failed'))
          throw error
        }
      },
    })
  }

  const openBalance = (row: UserRow) => {
    setBalanceUser(row)
    balanceForm.setFieldsValue({ amount: undefined, type: 'income' })
    setBalanceOpen(true)
  }

  const handleBalanceOk = async () => {
    if (!balanceUser) return
    try {
      const values = await balanceForm.validateFields()
      const amount = Math.abs(Number(values.amount))
      if (!amount) {
        message.error(t('user.amount_cannot_be_zero'))
        return
      }
      await updateBalance(balanceUser.id, {
        amount,
        type: values.type,
        source: 'manual',
        description: t('user.manual_balance_adjustment'),
      })
      message.success(t('user.balance_update_success'))
      setBalanceOpen(false)
      await refresh()
    } catch (error) {
      if ((error as { errorFields?: unknown })?.errorFields) return
      showError(error, t('common.operation_failed'))
    }
  }

  const handleExport = async () => {
    setExporting(true)
    try {
      await exportUsers({ ...searchForm, order_by: 'id:desc' })
      message.success(t('common.queued'))
    } catch (error) {
      showError(error, t('common.operation_failed'))
    } finally {
      setExporting(false)
    }
  }

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitting(true)
      const payload = { ...values, status: Number(values.status) }
      if (editId) {
        delete payload.password
        await updateUser(editId, payload)
        message.success(t('common.update_success'))
      } else {
        await createUser(payload)
        message.success(t('common.create_success'))
      }
      setFormOpen(false)
      await refresh()
    } catch (error) {
      if ((error as { errorFields?: unknown })?.errorFields) return
      showError(error, t('common.operation_failed'))
    } finally {
      setSubmitting(false)
    }
  }

  const moreMenu = (row: UserRow): MenuProps['items'] =>
    [
      getButtonState('user.password').show
        ? {
            key: 'resetPassword',
            label: t('user.reset_password'),
            onClick: () => handleResetPassword(row),
          }
        : null,
      getButtonState('user_balance_log.index').show
        ? {
            key: 'balanceLogs',
            label: t('user.balance_logs'),
            onClick: () => navigate(`/user-balance-logs?user_id=${row.id}`),
          }
        : null,
      getButtonState('user.destroy').show
        ? {
            key: 'delete',
            danger: true,
            label: t('common.delete'),
            onClick: () => confirmDelete(row.id, row.username),
          }
        : null,
    ].filter(Boolean) as MenuProps['items']

  const columns: ColumnsType<UserRow> = [
    { title: t('table.id'), dataIndex: 'id', width: 80, sorter: true },
    { title: t('table.username'), dataIndex: 'username' },
    { title: t('table.nickname'), dataIndex: 'nickname' },
    { title: t('table.email'), dataIndex: 'email' },
    { title: t('table.phone'), dataIndex: 'phone' },
    {
      title: t('table.balance'),
      dataIndex: 'balance',
      width: 140,
      render: (balance, row) => (
        <span style={{ color: 'var(--ant-color-primary)', fontWeight: 600 }}>
          {formatUserBalance(balance, row.currency)}
        </span>
      ),
    },
    {
      title: t('common.status'),
      dataIndex: 'status',
      width: 100,
      render: (status: number, row) => (
        <Switch
          checked={status === 1}
          disabled={getButtonState('user.update').disabled}
          onChange={(checked) => void handleStatusChange(row, checked)}
        />
      ),
    },
    { title: t('table.created_at'), dataIndex: 'created_at', width: 180, sorter: true },
    {
      title: t('common.operation'),
      key: 'operation',
      width: 280,
      fixed: 'end',
      render: (_, row) => (
        <Space size={0} wrap={false} style={{ whiteSpace: 'nowrap' }}>
          {getButtonState('user.update').show && (
            <PermissionButton permission="user.update" type="link" onClick={() => void openEdit(row)}>
              {t('common.edit')}
            </PermissionButton>
          )}
          {getButtonState('user.update_balance').show && (
            <PermissionButton permission="user.update_balance" type="link" onClick={() => openBalance(row)}>
              {t('user.update_balance')}
            </PermissionButton>
          )}
          <Dropdown menu={{ items: moreMenu(row) }}>
            <a onClick={(e) => e.preventDefault()} style={{ whiteSpace: 'nowrap' }}>
              {t('common.more')} <DownOutlined />
            </a>
          </Dropdown>
        </Space>
      ),
    },
  ]

  const {
    filteredColumns,
    open: columnSettingOpen,
    openColumnSetting,
    closeColumnSetting,
    allColumns,
    visibleColumns,
    columnOrder,
    fixedColumns,
    handleConfirm: handleColumnSettingConfirm,
  } = useColumnSetting('user', columns)

  return (
    <PageContainer
      title={t('menu.user')}
      extra={
        <Space>
          {toolbar}
          <PermissionButton permission="user.export" loading={exporting} onClick={() => void handleExport()}>
            {t('common.export')}
          </PermissionButton>
          <Button icon={<SettingOutlined />} onClick={openColumnSetting}>
            {t('common.column_setting')}
          </Button>
        </Space>
      }
    >
      <SearchForm
        fields={[
          { name: 'username', label: t('table.username') },
          {
            name: 'status',
            label: t('common.status'),
            type: 'select',
            options: [
              { label: t('common.enabled'), value: 1 },
              { label: t('common.disabled'), value: 0 },
            ],
          },
        ]}
        values={searchForm}
        onChange={onSearchFormChange}
        onSearch={handleSearch}
        onReset={handleReset}
      />
      <Table<UserRow>
        rowKey="id"
        loading={loading}
        columns={filteredColumns}
        dataSource={tableData}
        scroll={{ x: 1200 }}
        pagination={{
          current: pagination.page,
          pageSize: pagination.pageSize,
          total: pagination.total,
          showSizeChanger: true,
          showTotal: (total) => t('common.total', { total }),
        }}
        onChange={(pager, _f, sorter) =>
          handlePaginatedTableChange({ pager, sorter, pagination, loadData, handleSortChange })
        }
      />

      <Modal
        open={formOpen}
        title={editId ? t('user.edit') : t('user.add')}
        onCancel={() => setFormOpen(false)}
        onOk={() => void handleSubmit()}
        confirmLoading={submitting}
        destroyOnHidden
      >
        <Form form={form} layout="vertical" disabled={loadingDetail}>
          <Form.Item name="username" label={t('table.username')} rules={[{ required: true }]}>
            <Input disabled={!!editId} />
          </Form.Item>
          {!editId && (
            <Form.Item name="password" label={t('common.password')} rules={[{ required: true }, { min: 6 }]}>
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
          <Form.Item name="status" label={t('common.status')}>
            <Radio.Group
              options={[
                { label: t('common.enabled'), value: 1 },
                { label: t('common.disabled'), value: 0 },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        open={balanceOpen}
        title={t('user.update_balance')}
        onCancel={() => setBalanceOpen(false)}
        onOk={() => void handleBalanceOk()}
        destroyOnHidden
      >
        <Form form={balanceForm} layout="vertical">
          <Form.Item label={t('table.username')}>
            <Input value={balanceUser?.username} disabled />
          </Form.Item>
          <Form.Item label={t('table.balance')}>
            <Input value={formatUserBalance(balanceUser?.balance, balanceUser?.currency)} disabled />
          </Form.Item>
          <Form.Item
            name="amount"
            label={t('user.amount')}
            rules={[{ required: true, message: t('user.amount_required') }]}
            extra={t('user.update_balance_prompt')}
          >
            <InputNumber style={{ width: '100%' }} min={0.01} precision={2} />
          </Form.Item>
          <Form.Item name="type" label={t('user.change_type')} rules={[{ required: true }]}>
            <Select
              options={[
                { label: t('user.balance_income'), value: 'income' },
                { label: t('user.balance_expense'), value: 'expense' },
                { label: t('user.balance_refund'), value: 'refund' },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>
      <ColumnSettingDialog
        open={columnSettingOpen}
        onClose={closeColumnSetting}
        allColumns={allColumns}
        visibleColumns={visibleColumns}
        columnOrder={columnOrder}
        fixedColumns={fixedColumns}
        onConfirm={handleColumnSettingConfirm}
      />
    </PageContainer>
  )
}
