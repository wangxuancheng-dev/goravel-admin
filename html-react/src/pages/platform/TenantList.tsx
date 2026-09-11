import { useMemo, useState } from 'react'
import {
  Alert,
  App,
  Button,
  Divider,
  Form,
  Input,
  InputNumber,
  Modal,
  Select,
  Space,
  Switch,
  Table,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useTranslation } from 'react-i18next'
import { SettingOutlined } from '@ant-design/icons'
import {
  createPlatformTenant,
  getPlatformTenantList,
  updatePlatformTenant,
  updatePlatformTenantStatus,
} from '@/api/platform'
import { useListPage } from '@/hooks/useListPage'
import { handlePaginatedTableChange } from '@/utils/tableChange'
import { useUnhandledError } from '@/hooks/useUnhandledError'
import { useColumnSetting } from '@/hooks/useColumnSetting'
import PageContainer from '@/components/PageContainer'
import ColumnSettingDialog from '@/components/ColumnSettingDialog'
import SearchForm from '@/components/SearchForm'
import { entityField } from '@/utils/normalize'

interface TenantRow {
  id: number | string
  code?: string
  name?: string
  driver?: string
  isolation?: string
  host?: string
  port?: number
  database?: string
  schema?: string
  username?: string
  has_password?: boolean
  status?: number
  created_at?: string
}

export default function PlatformTenantList() {
  const { t } = useTranslation()
  const { message } = App.useApp()
  const showError = useUnhandledError()
  const [createOpen, setCreateOpen] = useState(false)
  const [editing, setEditing] = useState<TenantRow | null>(null)
  const [saving, setSaving] = useState(false)
  const [form] = Form.useForm()

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
  } = useListPage<TenantRow>({
    fetchApi: getPlatformTenantList,
    initialSearchForm: { code: '', name: '', status: '' },
    defaultSort: 'id:desc',
    normalizeRows: false,
    transformData: (row) => {
      const record = row
      return {
        id: entityField(record, 'id', '')!,
        code: String(entityField(record, 'code', '') ?? ''),
        name: String(entityField(record, 'name', '') ?? ''),
        driver: String(entityField(record, 'driver', '') ?? ''),
        isolation: String(entityField(record, 'isolation', '') ?? ''),
        host: String(entityField(record, 'host', '') ?? ''),
        port: Number(entityField(record, 'port', 0) ?? 0),
        database: String(entityField(record, 'database', '') ?? ''),
        schema: String(entityField(record, 'schema', '') ?? ''),
        username: String(entityField(record, 'username', '') ?? ''),
        has_password: Boolean(entityField(record, 'has_password', false)),
        status: Number(entityField(record, 'status', 0) ?? 0),
        created_at: String(entityField(record, 'created_at', '') ?? ''),
      }
    },
  })

  const columns = useMemo<ColumnsType<TenantRow>>(
    () => [
      { title: t('table.id'), dataIndex: 'id', key: 'id', width: 80, sorter: true },
      { title: t('tenant.code'), dataIndex: 'code', key: 'code' },
      { title: t('tenant.name'), dataIndex: 'name', key: 'name' },
      { title: t('tenant.driver'), dataIndex: 'driver', key: 'driver', width: 100 },
      { title: t('tenant.isolation'), dataIndex: 'isolation', key: 'isolation', width: 110 },
      { title: t('tenant.host'), dataIndex: 'host', key: 'host' },
      { title: t('tenant.database'), dataIndex: 'database', key: 'database' },
      { title: t('tenant.schema'), dataIndex: 'schema', key: 'schema' },
      {
        title: t('common.status'),
        dataIndex: 'status',
        key: 'status',
        width: 110,
        render: (status, row) => (
          <Switch
            checked={Number(status) === 1}
            onChange={async (checked) => {
              try {
                await updatePlatformTenantStatus(row.id, checked ? 1 : 0)
                message.success(t('common.update_success'))
                await refresh()
              } catch (error) {
                showError(error, t('common.operation_failed'))
              }
            }}
          />
        ),
      },
      { title: t('table.created_at'), dataIndex: 'created_at', key: 'created_at', width: 180 },
      {
        title: t('common.operation'),
        key: 'actions',
        width: 100,
        render: (_, row) => (
          <Button
            type="link"
            onClick={() => {
              setEditing(row)
              form.setFieldsValue({
                name: row.name,
                host: row.host,
                port: row.port || 0,
                username: row.username,
                password: '',
                database: row.database,
                schema: row.schema,
              })
            }}
          >
            {t('common.edit')}
          </Button>
        ),
      },
    ],
    [t, message, refresh, showError, form],
  )

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
  } = useColumnSetting('platform-tenant', columns)

  const submitCreate = async () => {
    try {
      const values = await form.validateFields()
      setSaving(true)
      await createPlatformTenant(values)
      message.success(t('common.create_success'))
      setCreateOpen(false)
      await refresh()
    } catch (error) {
      if ((error as { errorFields?: unknown })?.errorFields) return
      showError(error, t('common.operation_failed'))
    } finally {
      setSaving(false)
    }
  }

  const submitEdit = async () => {
    if (!editing) return
    try {
      const values = await form.validateFields()
      setSaving(true)
      const payload: Record<string, unknown> = {
        name: values.name,
        host: values.host,
        port: values.port || 0,
        username: values.username,
        database: values.database,
        schema: values.schema,
      }
      if (values.password) payload.password = values.password
      await updatePlatformTenant(editing.id, payload)
      message.success(t('common.update_success'))
      setEditing(null)
      await refresh()
    } catch (error) {
      if ((error as { errorFields?: unknown })?.errorFields) return
      showError(error, t('common.operation_failed'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <PageContainer
      title={t('menu.tenant')}
      extra={
        <Space>
          <Button
            type="primary"
            onClick={() => {
              form.resetFields()
              form.setFieldsValue({
                driver: 'mysql',
                isolation: 'database',
                skip_create: false,
                port: 0,
              })
              setCreateOpen(true)
            }}
          >
            {t('tenant.add')}
          </Button>
          <Button icon={<SettingOutlined />} onClick={openColumnSetting}>
            {t('common.column_setting')}
          </Button>
        </Space>
      }
    >
      <SearchForm
        fields={[
          { name: 'code', label: t('tenant.code') },
          { name: 'name', label: t('tenant.name') },
          {
            name: 'status',
            label: t('common.status'),
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
        dataSource={tableData}
        columns={filteredColumns}
        pagination={{
          current: pagination.page,
          pageSize: pagination.pageSize,
          total: pagination.total,
          showSizeChanger: true,
        }}
        onChange={(pager, _f, sorter) =>
          handlePaginatedTableChange({ pager, sorter, pagination, loadData, handleSortChange })
        }
      />

      <ColumnSettingDialog
        open={columnSettingOpen}
        onClose={closeColumnSetting}
        allColumns={allColumns}
        visibleColumns={visibleColumns}
        columnOrder={columnOrder}
        fixedColumns={fixedColumns}
        onConfirm={handleColumnSettingConfirm}
      />

      <Modal
        title={t('tenant.add')}
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        onOk={() => void submitCreate()}
        confirmLoading={saving}
        destroyOnHidden
        width={640}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="code"
            label={t('tenant.code')}
            rules={[{ required: true, message: t('tenant.code_required') }]}
          >
            <Input placeholder={t('tenant.code_placeholder')} />
          </Form.Item>
          <Form.Item
            name="name"
            label={t('tenant.name')}
            rules={[{ required: true, message: t('tenant.name_required') }]}
          >
            <Input />
          </Form.Item>
          <Form.Item name="driver" label={t('tenant.driver')}>
            <Select
              options={[
                { value: 'mysql', label: 'mysql' },
                { value: 'postgres', label: 'postgres' },
              ]}
            />
          </Form.Item>
          <Form.Item name="isolation" label={t('tenant.isolation')}>
            <Select
              options={[
                { value: 'database', label: 'database' },
                { value: 'schema', label: 'schema' },
              ]}
            />
          </Form.Item>
          <Form.Item name="database" label={t('tenant.database')}>
            <Input placeholder={t('tenant.database_placeholder')} />
          </Form.Item>
          <Form.Item name="schema" label={t('tenant.schema')}>
            <Input placeholder={t('tenant.schema_placeholder')} />
          </Form.Item>
          <Divider>{t('tenant.remote_connection')}</Divider>
          <Form.Item name="host" label={t('tenant.host')}>
            <Input placeholder={t('tenant.host_placeholder')} />
          </Form.Item>
          <Form.Item name="port" label={t('tenant.port')}>
            <InputNumber min={0} max={65535} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="username" label={t('tenant.username')}>
            <Input placeholder={t('tenant.username_placeholder')} />
          </Form.Item>
          <Form.Item name="password" label={t('tenant.password')}>
            <Input.Password placeholder={t('tenant.password_placeholder')} />
          </Form.Item>
          <Alert
            type="info"
            showIcon
            style={{ marginBottom: 12 }}
            message={t('tenant.migrate_cli_tip')}
          />
          <Form.Item
            name="skip_create"
            label={t('tenant.skip_create')}
            valuePropName="checked"
            extra={t('tenant.skip_create_tip')}
          >
            <Switch />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={t('tenant.edit')}
        open={!!editing}
        onCancel={() => setEditing(null)}
        onOk={() => void submitEdit()}
        confirmLoading={saving}
        destroyOnHidden
        width={640}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="name"
            label={t('tenant.name')}
            rules={[{ required: true, message: t('tenant.name_required') }]}
          >
            <Input />
          </Form.Item>
          <Divider>{t('tenant.remote_connection')}</Divider>
          <Form.Item name="host" label={t('tenant.host')}>
            <Input placeholder={t('tenant.host_placeholder')} />
          </Form.Item>
          <Form.Item name="port" label={t('tenant.port')}>
            <InputNumber min={0} max={65535} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="username" label={t('tenant.username')}>
            <Input placeholder={t('tenant.username_placeholder')} />
          </Form.Item>
          <Form.Item name="password" label={t('tenant.password')}>
            <Input.Password
              placeholder={
                editing?.has_password ? t('tenant.password_keep') : t('tenant.password_placeholder')
              }
            />
          </Form.Item>
          <Form.Item name="database" label={t('tenant.database')}>
            <Input />
          </Form.Item>
          <Form.Item name="schema" label={t('tenant.schema')}>
            <Input />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  )
}
