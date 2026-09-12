import { useMemo, useState, type ReactNode } from 'react'
import { Button, Form, Input, InputNumber, Modal, Radio, Space, Table } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { App } from 'antd'
import { SettingOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { useListPage } from '@/hooks/useListPage'
import { useCrudActions } from '@/hooks/useCrudActions'
import { usePermission } from '@/hooks/usePermission'
import { useUnhandledError } from '@/hooks/useUnhandledError'
import { useColumnSetting } from '@/hooks/useColumnSetting'
import PageContainer from '@/components/PageContainer'
import SearchForm, { type SearchField } from '@/components/SearchForm'
import StatusTag from '@/components/StatusTag'
import PermissionButton from '@/components/PermissionButton'
import ColumnSettingDialog from '@/components/ColumnSettingDialog'
import { entityField } from '@/utils/normalize'
import { handlePaginatedTableChange } from '@/utils/tableChange'
import { promptSensitiveConfirm } from '@/utils/sensitiveConfirm'
import type { ListFetchFn } from '@/types'

export interface SimpleField {
  name: string
  label: string
  required?: boolean
  type?: 'input' | 'textarea' | 'number' | 'status' | 'password'
  hideOnEdit?: boolean
  hideOnCreate?: boolean
}

interface SimpleCrudRow {
  id: string | number
  name?: string
  status?: number
  created_at?: string
}

interface SimpleCrudPageProps<T extends SimpleCrudRow> {
  title: string
  permissions: { index?: string; store: string; update: string; destroy: string }
  fetchApi: ListFetchFn
  createApi: (data: unknown) => Promise<unknown>
  updateApi: (id: string | number, data: unknown) => Promise<unknown>
  deleteApi: (id: string | number, data?: Record<string, unknown>) => Promise<unknown>
  searchFields?: SearchField[]
  initialSearchForm?: Record<string, unknown>
  formFields: SimpleField[]
  columns?: ColumnsType<T>
  transformRow?: (row: Record<string, unknown>) => T
  createTitle: string
  editTitle: string
  isProtected?: (row: T) => boolean
  extraColumns?: ColumnsType<T>
  renderExtraForm?: (editing: boolean) => ReactNode
  columnSettingKey?: string
  /** Prompt for confirm_code when creating */
  requireSensitiveConfirmOnCreate?: boolean
  /** Prompt for confirm_code when deleting */
  requireSensitiveConfirmOnDelete?: boolean
}

export default function SimpleCrudPage<T extends SimpleCrudRow>(props: SimpleCrudPageProps<T>) {
  const { t } = useTranslation()
  const { message, modal } = App.useApp()
  const showError = useUnhandledError()
  const { getButtonState } = usePermission()
  const [form] = Form.useForm()
  const [open, setOpen] = useState(false)
  const [editId, setEditId] = useState<string | number | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const transformRow =
    props.transformRow ||
    ((row: Record<string, unknown>) =>
      ({
        id: entityField(row, 'id', '')!,
        name: String(entityField(row, 'name', '') ?? ''),
        slug: String(entityField(row, 'slug', '') ?? ''),
        description: String(entityField(row, 'description', '') ?? ''),
        status: Number(entityField(row, 'status', 0) ?? 0),
        sort: Number(entityField(row, 'sort', 0) ?? 0),
        created_at: String(entityField(row, 'created_at', '') ?? ''),
        ...row,
      }) as unknown as T)

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
  } = useListPage<T>({
    fetchApi: props.fetchApi,
    initialSearchForm: props.initialSearchForm || {},
    normalizeRows: true,
    transformData: transformRow,
  })

  const { toolbar, confirmDelete } = useCrudActions({
    createPermission: props.permissions.store,
    onRefresh: refresh,
    onCreate: () => {
      setEditId(null)
      form.resetFields()
      const defaults: Record<string, unknown> = { status: 1, sort: 0 }
      props.formFields.forEach((f) => {
        if (f.type === 'status') defaults[f.name] = 1
        if (f.type === 'number' && f.name === 'sort') defaults[f.name] = 0
      })
      form.setFieldsValue(defaults)
      setOpen(true)
    },
    deleteApi: props.deleteApi,
    requireSensitiveConfirm: props.requireSensitiveConfirmOnDelete,
  })

  const columns = useMemo<ColumnsType<T>>(() => {
    const operationColumn: ColumnsType<T>[number] = {
      title: t('common.operation'),
      key: 'operation',
      width: 160,
      fixed: 'end',
      render: (_, row) => {
        const protectedRow = props.isProtected?.(row)
        return (
          <Space>
            {getButtonState(props.permissions.update).show && (
              <PermissionButton
                permission={props.permissions.update}
                type="link"
                onClick={() => {
                  setEditId(row.id)
                  form.setFieldsValue(row)
                  setOpen(true)
                }}
              >
                {t('common.edit')}
              </PermissionButton>
            )}
            {getButtonState(props.permissions.destroy).show && !protectedRow && (
              <PermissionButton
                permission={props.permissions.destroy}
                type="link"
                danger
                onClick={() => confirmDelete(row.id, row.name)}
              >
                {t('common.delete')}
              </PermissionButton>
            )}
          </Space>
        )
      },
    }

    if (props.columns) {
      return [...props.columns, operationColumn]
    }

    return [
      { title: t('table.id'), dataIndex: 'id', width: 80, sorter: true },
      { title: t('common.name'), dataIndex: 'name' },
      {
        title: t('common.status'),
        dataIndex: 'status',
        width: 100,
        render: (status: number) => <StatusTag status={status} />,
      },
      { title: t('table.created_at'), dataIndex: 'created_at', width: 180, sorter: true },
      ...(props.extraColumns || []),
      operationColumn,
    ]
  }, [props, t, getButtonState, confirmDelete, form])

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
  } = useColumnSetting(props.columnSettingKey || '', columns)

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setSubmitting(true)
      if (editId) {
        await props.updateApi(editId, values)
        message.success(t('common.update_success'))
      } else {
        if (props.requireSensitiveConfirmOnCreate) {
          const confirmCode = await promptSensitiveConfirm(modal, t)
          values.confirm_code = confirmCode
        }
        await props.createApi(values)
        message.success(t('common.create_success'))
      }
      setOpen(false)
      await refresh()
    } catch (error) {
      if ((error as { errorFields?: unknown })?.errorFields) return
      if ((error as Error)?.message === 'cancel') return
      showError(error, t('common.operation_failed'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <PageContainer
      title={props.title}
      extra={
        props.columnSettingKey ? (
          <Space>
            {toolbar}
            <Button icon={<SettingOutlined />} onClick={openColumnSetting}>
              {t('common.column_setting')}
            </Button>
          </Space>
        ) : (
          toolbar
        )
      }
    >
      {props.searchFields && props.searchFields.length > 0 && (
        <SearchForm
          fields={props.searchFields}
          values={searchForm}
          onChange={onSearchFormChange}
          onSearch={handleSearch}
          onReset={handleReset}
        />
      )}
      <Table<T>
        rowKey="id"
        loading={loading}
        columns={props.columnSettingKey ? filteredColumns : columns}
        dataSource={tableData}
        scroll={{ x: 960 }}
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
        open={open}
        title={editId ? props.editTitle : props.createTitle}
        onCancel={() => setOpen(false)}
        onOk={() => void handleSubmit()}
        confirmLoading={submitting}
        destroyOnHidden
        width={560}
      >
        <Form form={form} layout="vertical">
          {props.formFields.map((field) => {
            if (editId && field.hideOnEdit) return null
            if (!editId && field.hideOnCreate) return null
            if (field.type === 'status') {
              return (
                <Form.Item key={field.name} name={field.name} label={field.label}>
                  <Radio.Group
                    options={[
                      { label: t('common.enabled'), value: 1 },
                      { label: t('common.disabled'), value: 0 },
                    ]}
                  />
                </Form.Item>
              )
            }
            if (field.type === 'number') {
              return (
                <Form.Item key={field.name} name={field.name} label={field.label}>
                  <InputNumber style={{ width: '100%' }} min={0} />
                </Form.Item>
              )
            }
            if (field.type === 'textarea') {
              return (
                <Form.Item key={field.name} name={field.name} label={field.label}>
                  <Input.TextArea rows={3} />
                </Form.Item>
              )
            }
            if (field.type === 'password') {
              return (
                <Form.Item
                  key={field.name}
                  name={field.name}
                  label={field.label}
                  rules={field.required ? [{ required: true, message: t('common.required', { defaultValue: field.label }) }] : undefined}
                >
                  <Input.Password />
                </Form.Item>
              )
            }
            return (
              <Form.Item
                key={field.name}
                name={field.name}
                label={field.label}
                rules={field.required ? [{ required: true, message: `${field.label}` }] : undefined}
              >
                <Input />
              </Form.Item>
            )
          })}
          {props.renderExtraForm?.(!!editId)}
        </Form>
      </Modal>

      {props.columnSettingKey && (
        <ColumnSettingDialog
          open={columnSettingOpen}
          onClose={closeColumnSetting}
          allColumns={allColumns}
          visibleColumns={visibleColumns}
          columnOrder={columnOrder}
          fixedColumns={fixedColumns}
          onConfirm={handleColumnSettingConfirm}
        />
      )}
    </PageContainer>
  )
}
