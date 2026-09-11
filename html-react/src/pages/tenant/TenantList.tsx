import { useMemo, useState } from 'react'
import { App, Form, Input, Modal, Select, Switch, Table, Button, Space } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useTranslation } from 'react-i18next'
import { SettingOutlined } from '@ant-design/icons'
import { createTenant, getTenantList, updateTenantStatus } from '@/api/tenant'
import { useListPage } from '@/hooks/useListPage'
import { handlePaginatedTableChange } from '@/utils/tableChange'
import { useCrudActions } from '@/hooks/useCrudActions'
import { usePermission } from '@/hooks/usePermission'
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
  database?: string
  schema?: string
  status?: number
  created_at?: string
}

export default function TenantList() {
  const { t } = useTranslation()
  const { message } = App.useApp()
  const showError = useUnhandledError()
  const { getButtonState } = usePermission()
  const [createOpen, setCreateOpen] = useState(false)
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
    fetchApi: getTenantList,
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
        database: String(entityField(record, 'database', '') ?? ''),
        schema: String(entityField(record, 'schema', '') ?? ''),
        status: Number(entityField(record, 'status', 0) ?? 0),
        created_at: String(entityField(record, 'created_at', '') ?? ''),
      }
    },
  })

  const { toolbar } = useCrudActions({
    createPermission: 'tenant.store',
    onRefresh: refresh,
    onCreate: () => {
      form.resetFields()
      form.setFieldsValue({ driver: 'mysql', isolation: 'database', migrate: true })
      setCreateOpen(true)
    },
  })

  const columns = useMemo<ColumnsType<TenantRow>>(
    () => [
      { title: t('table.id'), dataIndex: 'id', key: 'id', width: 80, sorter: true },
      { title: t('tenant.code'), dataIndex: 'code', key: 'code' },
      { title: t('tenant.name'), dataIndex: 'name', key: 'name' },
      { title: t('tenant.driver'), dataIndex: 'driver', key: 'driver', width: 100 },
      { title: t('tenant.isolation'), dataIndex: 'isolation', key: 'isolation', width: 110 },
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
            disabled={getButtonState('tenant.update_status').disabled}
            onChange={async (checked) => {
              try {
                await updateTenantStatus(row.id, checked ? 1 : 0)
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
    ],
    [t, getButtonState, message, refresh, showError],
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
  } = useColumnSetting('tenant', columns)

  const submitCreate = async () => {
    try {
      const values = await form.validateFields()
      setSaving(true)
      await createTenant(values)
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

  return (
    <PageContainer
      title={t('menu.tenant')}
      extra={
        <Space>
          {toolbar}
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
          <Form.Item name="migrate" label={t('tenant.migrate')} valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>
    </PageContainer>
  )
}
