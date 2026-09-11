import { useState } from 'react'
import { Space, Table, Tag } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useTranslation } from 'react-i18next'
import { deletePaymentMethod, getPaymentMethodList } from '@/api/paymentMethod'
import { useListPage } from '@/hooks/useListPage'
import { handlePaginatedTableChange } from '@/utils/tableChange'
import { useCrudActions } from '@/hooks/useCrudActions'
import { usePermission } from '@/hooks/usePermission'
import PageContainer from '@/components/PageContainer'
import SearchForm from '@/components/SearchForm'
import PermissionButton from '@/components/PermissionButton'
import { entityField } from '@/utils/normalize'
import PaymentMethodFormModal from './PaymentMethodFormModal'
import {
  createPaymentMethodTypeOptions,
  getPaymentMethodTypeLabel,
  paymentMethodInitialSearchForm,
  type PaymentMethodSearchForm,
} from './paymentMethod.config'

interface PaymentMethodRow {
  id: number | string
  name: string
  code: string
  type: string
  is_active: boolean
  sort: number
  description?: string
  created_at?: string
}

export default function PaymentMethodList() {
  const { t } = useTranslation()
  const { getButtonState } = usePermission()
  const [open, setOpen] = useState(false)
  const [editId, setEditId] = useState<string | number | null>(null)

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
  } = useListPage<PaymentMethodRow, PaymentMethodSearchForm>({
    fetchApi: getPaymentMethodList,
    initialSearchForm: paymentMethodInitialSearchForm,
    defaultSort: 'sort:asc,id:desc',
    normalizeRows: true,
    transformData: (row) => {
      const record = row
      return {
        id: entityField(record, 'id', '')!,
        name: String(entityField(record, 'name', '') ?? ''),
        code: String(entityField(record, 'code', '') ?? ''),
        type: String(entityField(record, 'type', '') ?? ''),
        is_active: Boolean(entityField(record, 'is_active', false)),
        sort: Number(entityField(record, 'sort', 0) ?? 0),
        description: String(entityField(record, 'description', '') ?? ''),
        created_at: String(entityField(record, 'created_at', '') ?? ''),
      }
    },
  })

  const { toolbar, confirmDelete } = useCrudActions({
    createPermission: 'payment_method.store',
    onRefresh: refresh,
    onCreate: () => {
      setEditId(null)
      setOpen(true)
    },
    deleteApi: deletePaymentMethod,
  })

  const columns: ColumnsType<PaymentMethodRow> = [
    { title: t('table.id'), dataIndex: 'id', width: 80, sorter: true },
    { title: t('payment_method.name'), dataIndex: 'name', width: 160, ellipsis: true, sorter: true },
    {
      title: t('payment_method.type'),
      dataIndex: 'type',
      width: 140,
      sorter: true,
      render: (type: string) => <Tag>{getPaymentMethodTypeLabel(t, type)}</Tag>,
    },
    {
      title: t('table.status'),
      dataIndex: 'is_active',
      width: 100,
      render: (isActive: boolean) => (
        <Tag color={isActive ? 'success' : 'error'}>
          {isActive ? t('common.enabled') : t('common.disabled')}
        </Tag>
      ),
    },
    { title: t('table.sort'), dataIndex: 'sort', width: 100, sorter: true },
    { title: t('table.description'), dataIndex: 'description', ellipsis: true },
    { title: t('table.created_at'), dataIndex: 'created_at', width: 180, sorter: true },
    {
      title: t('common.operation'),
      key: 'operation',
      width: 150,
      fixed: 'end',
      render: (_, row) => (
        <Space>
          {getButtonState('payment_method.update').show && (
            <PermissionButton
              permission="payment_method.update"
              type="link"
              onClick={() => {
                setEditId(row.id)
                setOpen(true)
              }}
            >
              {t('common.edit')}
            </PermissionButton>
          )}
          {getButtonState('payment_method.destroy').show && (
            <PermissionButton
              permission="payment_method.destroy"
              type="link"
              danger
              onClick={() => confirmDelete(row.id, row.name)}
            >
              {t('common.delete')}
            </PermissionButton>
          )}
        </Space>
      ),
    },
  ]

  return (
    <PageContainer title={t('menu.payment_method')} extra={toolbar}>
      <SearchForm
        fields={[
          {
            name: 'name',
            label: t('payment_method.name'),
            placeholder: t('payment_method.name_placeholder'),
          },
          {
            name: 'type',
            label: t('payment_method.type'),
            type: 'select',
            placeholder: t('payment_method.type_placeholder'),
            options: createPaymentMethodTypeOptions(t),
          },
          {
            name: 'is_active',
            label: t('table.status'),
            type: 'select',
            placeholder: t('payment_method.is_active_placeholder'),
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
      <Table<PaymentMethodRow>
        rowKey="id"
        loading={loading}
        columns={columns}
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
      <PaymentMethodFormModal
        open={open}
        editId={editId}
        onClose={() => setOpen(false)}
        onSuccess={() => void refresh()}
      />
    </PageContainer>
  )
}
