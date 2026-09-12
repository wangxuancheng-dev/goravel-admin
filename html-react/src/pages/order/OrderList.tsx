import { useMemo, useState } from 'react'
import {
  App,
  Descriptions,
  Divider,
  Drawer,
  Form,
  Input,
  Modal,
  Select,
  Space,
  Table,
  Tag,
  Upload,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import type { UploadProps } from 'antd/es/upload'
import { UploadOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import {
  deleteOrder,
  exportOrder,
  getOrderDetail,
  getOrderList,
  importOrder,
  updateOrder,
} from '@/api/order'
import { useListPage } from '@/hooks/useListPage'
import { handlePaginatedTableChange } from '@/utils/tableChange'
import { useCrudActions } from '@/hooks/useCrudActions'
import { usePermission } from '@/hooks/usePermission'
import { useUnhandledError } from '@/hooks/useUnhandledError'
import { useQueuedExport } from '@/hooks/useQueuedExport'
import PageContainer from '@/components/PageContainer'
import SearchForm from '@/components/SearchForm'
import PermissionButton from '@/components/PermissionButton'
import { entityField } from '@/utils/normalize'
import OrderFormModal from './OrderFormModal'
import {
  createOrderInitialSearchForm,
  formatOrderAmount,
  formatOrderTime,
  getOrderDetailField,
  getOrderDetails,
  getOrderStatusTagColor,
  getOrderStatusText,
  ORDER_STATUS_OPTIONS,
  type OrderSearchForm,
} from './order.config'

interface OrderRow {
  id: number | string
  order_no?: string
  user_id?: number | string
  amount?: number | string
  status?: string
  remark?: string
  created_at?: string
  details?: Record<string, unknown>[]
  name?: string
}

interface OrderDetailPayload {
  order?: Record<string, unknown>
  details?: Record<string, unknown>[]
}

const initialSearch = createOrderInitialSearchForm()

export default function OrderList() {
  const { t } = useTranslation()
  const { message, modal } = App.useApp()
  const showError = useUnhandledError()
  const { getButtonState } = usePermission()
  const [editForm] = Form.useForm()

  const [createOpen, setCreateOpen] = useState(false)
  const [detailOpen, setDetailOpen] = useState(false)
  const [orderDetail, setOrderDetail] = useState<OrderDetailPayload | null>(null)
  const [editOpen, setEditOpen] = useState(false)
  const [editSubmitting, setEditSubmitting] = useState(false)
  const [currentEditOrder, setCurrentEditOrder] = useState<Record<string, unknown> | null>(null)
  const [importing, setImporting] = useState(false)

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
  } = useListPage<OrderRow, OrderSearchForm>({
    fetchApi: getOrderList,
    initialSearchForm: initialSearch,
    defaultSort: 'created_at:desc',
    normalizeRows: true,
    transformData: (row) => {
      const record = row
      return {
        id: entityField(record, 'id', '')!,
        order_no: String(entityField(record, 'order_no', '') ?? ''),
        user_id: entityField(record, 'user_id', '') as number | string,
        amount: entityField(record, 'amount', 0) as number | string,
        status: String(entityField(record, 'status', '') ?? ''),
        remark: String(entityField(record, 'remark', '') ?? ''),
        created_at: String(entityField(record, 'created_at', '') ?? ''),
        details: getOrderDetails(record),
        name: String(entityField(record, 'order_no', '') ?? ''),
      }
    },
  })

  const { toolbar } = useCrudActions({
    createPermission: 'order.store',
    onRefresh: refresh,
    onCreate: () => setCreateOpen(true),
  })

  const { exporting, handleExport } = useQueuedExport({
    exportApi: exportOrder,
    getParams: () => ({ ...searchForm }),
    redirectPath: '/exports',
  })

  const statusOptions = useMemo(
    () =>
      ORDER_STATUS_OPTIONS.map((item) => ({
        label: t(item.labelKey),
        value: item.value,
      })),
    [t],
  )

  const openDetail = async (row: OrderRow) => {
    try {
      const orderNo = row.order_no
      const res = await getOrderDetail(row.id, orderNo ? { order_no: orderNo } : {})
      setOrderDetail((res.data || {}) as OrderDetailPayload)
      setDetailOpen(true)
    } catch (error) {
      showError(error, t('common.query_failed'))
    }
  }

  const openEdit = async (row: OrderRow) => {
    try {
      const orderNo = row.order_no
      const res = await getOrderDetail(row.id, orderNo ? { order_no: orderNo } : {})
      const data = (res.data || {}) as OrderDetailPayload
      if (!data.order) {
        message.error(t('common.query_failed'))
        return
      }
      setCurrentEditOrder(data.order)
      editForm.setFieldsValue({
        status: String(getOrderDetailField(data.order, 'status', 'pending')),
        remark: String(getOrderDetailField(data.order, 'remark', '') || ''),
      })
      setEditOpen(true)
    } catch (error) {
      showError(error, t('common.query_failed'))
    }
  }

  const closeEdit = () => {
    setEditOpen(false)
    setCurrentEditOrder(null)
    editForm.resetFields()
  }

  const handleEditSubmit = async () => {
    if (!currentEditOrder) return
    try {
      const values = await editForm.validateFields()
      setEditSubmitting(true)
      const orderNo = String(getOrderDetailField(currentEditOrder, 'order_no', '') || '')
      const id = getOrderDetailField(currentEditOrder, 'id', '') as string | number
      await updateOrder(id, {
        status: values.status,
        remark: values.remark || '',
        order_no: orderNo,
      })
      message.success(t('common.update_success'))
      closeEdit()
      await refresh()
    } catch (error) {
      if ((error as { errorFields?: unknown })?.errorFields) return
      showError(error, t('common.operation_failed'))
    } finally {
      setEditSubmitting(false)
    }
  }

  const handleDelete = (row: OrderRow) => {
    modal.confirm({
      title: t('common.delete_confirm'),
      content: row.order_no || undefined,
      okType: 'danger',
      onOk: async () => {
        try {
          await deleteOrder(row.id, row.order_no ? { order_no: row.order_no } : {})
          message.success(t('common.delete_success'))
          await refresh()
        } catch (error) {
          showError(error, t('common.operation_failed'))
          throw error
        }
      },
    })
  }

  const uploadProps: UploadProps = {
    accept: '.csv',
    showUploadList: false,
    beforeUpload: (file) => {
      if (!file.name.toLowerCase().endsWith('.csv')) {
        message.error(t('common.invalid_file_type'))
        return false
      }
      setImporting(true)
      void importOrder(file as File)
        .then((res) => {
          const payload = (res.data || {}) as Record<string, unknown>
          const result = ((payload.data || payload) as Record<string, unknown>) || {}
          if (result.async && result.import_id) {
            message.success(t('export.task_submitted'))
            return
          }
          const successCount = Number(result.success_count || 0)
          const failedCount = Number(result.failed_count || 0)
          const errors = Array.isArray(result.errors) ? (result.errors as string[]) : []

          if (successCount > 0) {
            message.success(t('common.import_success'))
            if (failedCount > 0 && errors.length) {
              message.warning(errors.slice(0, 10).join('\n'))
            }
            void refresh()
          } else {
            message.warning(t('common.import_no_data'))
            if (errors.length) {
              message.error(errors.slice(0, 10).join('\n'))
            }
          }
        })
        .catch((error) => showError(error, t('common.operation_failed')))
        .finally(() => setImporting(false))
      return false
    },
  }

  const detailColumns: ColumnsType<Record<string, unknown>> = [
    {
      title: t('order.product_name'),
      dataIndex: 'product_name',
      width: 200,
      render: (_, detail) => String(entityField(detail, 'product_name', '-') ?? '-'),
    },
    {
      title: t('order.price'),
      dataIndex: 'price',
      width: 120,
      render: (_, detail) => formatOrderAmount(entityField(detail, 'price')),
    },
    {
      title: t('order.quantity'),
      dataIndex: 'quantity',
      width: 100,
      render: (_, detail) => Number(entityField(detail, 'quantity', 0) ?? 0),
    },
    {
      title: t('order.subtotal'),
      dataIndex: 'subtotal',
      width: 120,
      render: (_, detail) => formatOrderAmount(entityField(detail, 'subtotal')),
    },
  ]

  const columns: ColumnsType<OrderRow> = [
    { title: t('table.id'), dataIndex: 'id', width: 80, sorter: true },
    { title: t('order.order_no'), dataIndex: 'order_no' },
    { title: t('order.user_id'), dataIndex: 'user_id', width: 100 },
    {
      title: t('order.amount'),
      dataIndex: 'amount',
      width: 120,
      sorter: true,
      render: (v) => formatOrderAmount(v),
    },
    {
      title: t('order.status'),
      dataIndex: 'status',
      width: 110,
      render: (status: string) => (
        <Tag color={getOrderStatusTagColor(status)}>{getOrderStatusText(t, status)}</Tag>
      ),
    },
    {
      title: t('order.created_at'),
      dataIndex: 'created_at',
      width: 180,
      sorter: true,
    },
    {
      title: t('order.remark'),
      dataIndex: 'remark',
      width: 200,
      ellipsis: true,
      render: (v) => v || '-',
    },
    {
      title: t('common.operation'),
      key: 'operation',
      width: 220,
      fixed: 'end',
      render: (_, row) => (
        <Space>
          {getButtonState('order.show').show && (
            <PermissionButton permission="order.show" type="link" onClick={() => void openDetail(row)}>
              {t('common.view')}
            </PermissionButton>
          )}
          {getButtonState('order.update').show && (
            <PermissionButton permission="order.update" type="link" onClick={() => void openEdit(row)}>
              {t('common.edit')}
            </PermissionButton>
          )}
          {getButtonState('order.destroy').show && (
            <PermissionButton permission="order.destroy" type="link" danger onClick={() => handleDelete(row)}>
              {t('common.delete')}
            </PermissionButton>
          )}
        </Space>
      ),
    },
  ]

  const orderInfo = orderDetail?.order

  return (
    <PageContainer
      title={t('menu.order')}
      extra={
        <Space wrap>
          {toolbar}
          <Upload {...uploadProps}>
            <PermissionButton permission="order.import" icon={<UploadOutlined />} loading={importing}>
              {t('common.import')}
            </PermissionButton>
          </Upload>
          <PermissionButton
            permission="order.export"
            type="primary"
            loading={exporting}
            onClick={() => void handleExport()}
          >
            {t('common.export')}
          </PermissionButton>
        </Space>
      }
    >
      <SearchForm
        fields={[
          { name: 'user_id', label: t('order.user_id') },
          { name: 'order_no', label: t('order.order_no') },
          {
            name: 'status',
            label: t('order.status'),
            type: 'select',
            options: statusOptions,
          },
          { name: 'min_amount', label: t('order.min_amount'), advanced: true },
          { name: 'max_amount', label: t('order.max_amount'), advanced: true },
          { name: 'start_time', label: t('order.start_time'), type: 'datetime', advanced: true },
          { name: 'end_time', label: t('order.end_time'), type: 'datetime', advanced: true },
        ]}
        values={searchForm}
        onChange={onSearchFormChange}
        onSearch={handleSearch}
        onReset={handleReset}
      />

      <Table<OrderRow>
        rowKey="id"
        loading={loading}
        columns={columns}
        dataSource={tableData}
        scroll={{ x: 1200 }}
        expandable={{
          expandedRowRender: (row) => (
            <div style={{ padding: '8px 12px' }}>
              <h4 style={{ margin: '0 0 8px' }}>{t('order.order_details')}</h4>
              <Table
                size="small"
                pagination={false}
                rowKey={(_, index) => String(index)}
                columns={detailColumns}
                dataSource={row.details || []}
              />
            </div>
          ),
        }}
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

      <OrderFormModal open={createOpen} onClose={() => setCreateOpen(false)} onSuccess={() => void refresh()} />

      <Modal
        open={editOpen}
        title={t('order.edit_order')}
        width={600}
        destroyOnHidden
        confirmLoading={editSubmitting}
        onCancel={closeEdit}
        onOk={() => void handleEditSubmit()}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
      >
        <Form form={editForm} layout="vertical">
          <Form.Item
            name="status"
            label={t('order.status')}
            rules={[{ required: true, message: t('order.status_required') }]}
          >
            <Select placeholder={t('order.update_status_tip')} options={statusOptions} />
          </Form.Item>
          <Form.Item name="remark" label={t('order.remark')}>
            <Input.TextArea rows={4} placeholder={t('order.remark_placeholder')} />
          </Form.Item>
        </Form>
      </Modal>

      <Drawer open={detailOpen} title={t('order.detail')} width="80%" onClose={() => setDetailOpen(false)}>
        {orderInfo ? (
          <>
            <Descriptions column={2} bordered size="small">
              <Descriptions.Item label={t('order.order_no')}>
                {String(getOrderDetailField(orderInfo, 'order_no'))}
              </Descriptions.Item>
              <Descriptions.Item label={t('order.user_id')}>
                {String(getOrderDetailField(orderInfo, 'user_id'))}
              </Descriptions.Item>
              <Descriptions.Item label={t('order.amount')}>
                {formatOrderAmount(getOrderDetailField(orderInfo, 'amount', 0))}
              </Descriptions.Item>
              <Descriptions.Item label={t('order.status')}>
                <Tag color={getOrderStatusTagColor(String(getOrderDetailField(orderInfo, 'status', '')))}>
                  {getOrderStatusText(t, String(getOrderDetailField(orderInfo, 'status', '')))}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label={t('order.created_at')}>
                {formatOrderTime(getOrderDetailField(orderInfo, 'created_at', ''))}
              </Descriptions.Item>
              <Descriptions.Item label={t('order.remark')}>
                {String(getOrderDetailField(orderInfo, 'remark') || '-')}
              </Descriptions.Item>
            </Descriptions>
            <Divider>{t('order.details')}</Divider>
            <Table
              size="small"
              pagination={false}
              rowKey={(_, index) => String(index)}
              columns={detailColumns}
              dataSource={orderDetail?.details || []}
            />
          </>
        ) : null}
      </Drawer>
    </PageContainer>
  )
}
