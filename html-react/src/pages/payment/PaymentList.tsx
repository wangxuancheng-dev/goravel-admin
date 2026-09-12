import { useMemo, useState } from 'react'
import { Descriptions, Drawer, Space, Spin, Table, Tag, Button } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useTranslation } from 'react-i18next'
import { exportPayments, getPaymentDetail, getPaymentList } from '@/api/payment'
import { useListPage } from '@/hooks/useListPage'
import { handlePaginatedTableChange } from '@/utils/tableChange'
import { useQueuedExport } from '@/hooks/useQueuedExport'
import { useOptions } from '@/hooks/useOptions'
import { usePermission } from '@/hooks/usePermission'
import { useUnhandledError } from '@/hooks/useUnhandledError'
import PageContainer from '@/components/PageContainer'
import SearchForm from '@/components/SearchForm'
import PermissionButton from '@/components/PermissionButton'
import { entityField } from '@/utils/normalize'
import {
  createPaymentInitialSearchForm,
  createPaymentStatusOptions,
  formatPaymentAmount,
  formatPaymentDateTime,
  getPaymentMethodName,
  getPaymentStatusColor,
  getPaymentStatusText,
  type PaymentSearchForm,
} from './payment.config'

interface PaymentRow {
  id: number | string
  payment_no: string
  order_no: string
  payment_method?: unknown
  user_id: string | number
  amount: number | string
  status: string
  pay_time?: string
  created_at?: string
}

export default function PaymentList() {
  const { t } = useTranslation()
  const showError = useUnhandledError()
  const { getButtonState } = usePermission()
  const { selectOptions: paymentMethodOptions } = useOptions('payment_method')
  const [detailOpen, setDetailOpen] = useState(false)
  const [detailLoading, setDetailLoading] = useState(false)
  const [paymentDetail, setPaymentDetail] = useState<Record<string, unknown> | null>(null)

  const initialSearchForm = useMemo(() => createPaymentInitialSearchForm(), [])

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
  } = useListPage<PaymentRow, PaymentSearchForm>({
    fetchApi: getPaymentList,
    initialSearchForm,
    defaultSort: 'created_at:desc',
    normalizeRows: true,
    transformData: (row) => {
      const record = row
      return {
        id: entityField(record, 'id', '')!,
        payment_no: String(entityField(record, 'payment_no', '') ?? ''),
        order_no: String(entityField(record, 'order_no', '') ?? ''),
        payment_method: entityField(record, 'payment_method'),
        user_id: entityField(record, 'user_id', '')!,
        amount: (entityField(record, 'amount', 0) as number | string) ?? 0,
        status: String(entityField(record, 'status', '') ?? ''),
        pay_time: String(entityField(record, 'pay_time', '') ?? ''),
        created_at: String(entityField(record, 'created_at', '') ?? ''),
      }
    },
  })

  const { exporting, handleExport } = useQueuedExport({
    exportApi: exportPayments,
    getParams: () => ({ ...searchForm, order_by: 'created_at:desc' }),
    queuedMessageKey: 'common.queued',
  })

  const openDetail = async (row: PaymentRow) => {
    setDetailOpen(true)
    setDetailLoading(true)
    setPaymentDetail(null)
    try {
      const res = await getPaymentDetail(row.payment_no)
      const data = res.data as Record<string, unknown> | undefined
      const detail = (data?.data || data || {}) as Record<string, unknown>
      setPaymentDetail(detail)
    } catch (error) {
      showError(error, t('common.query_failed'))
      setDetailOpen(false)
    } finally {
      setDetailLoading(false)
    }
  }

  const columns: ColumnsType<PaymentRow> = [
    { title: t('table.id'), dataIndex: 'id', width: 80, sorter: true },
    { title: t('payment.payment_no'), dataIndex: 'payment_no', minWidth: 300, sorter: true },
    { title: t('payment.order_no'), dataIndex: 'order_no', minWidth: 180, sorter: true },
    {
      title: t('payment.payment_method'),
      dataIndex: 'payment_method',
      width: 150,
      render: (method) => getPaymentMethodName(method),
    },
    { title: t('payment.user_id'), dataIndex: 'user_id', width: 100, sorter: true },
    {
      title: t('payment.amount'),
      dataIndex: 'amount',
      width: 120,
      align: 'right',
      sorter: true,
      render: (amount) => formatPaymentAmount(amount),
    },
    {
      title: t('payment.status'),
      dataIndex: 'status',
      width: 100,
      sorter: true,
      render: (status) => (
        <Tag color={getPaymentStatusColor(status)}>{getPaymentStatusText(t, status)}</Tag>
      ),
    },
    { title: t('payment.pay_time'), dataIndex: 'pay_time', width: 180, sorter: true },
    { title: t('table.created_at'), dataIndex: 'created_at', width: 180, sorter: true },
    {
      title: t('common.operation'),
      key: 'operation',
      width: 100,
      fixed: 'end',
      render: (_, row) =>
        getButtonState('payment.show').show ? (
          <PermissionButton permission="payment.show" type="link" onClick={() => void openDetail(row)}>
            {t('common.view')}
          </PermissionButton>
        ) : null,
    },
  ]

  return (
    <PageContainer
      title={t('menu.payment')}
      extra={
        <Space>
          <PermissionButton permission="payment.export" loading={exporting} onClick={() => void handleExport()}>
            {t('common.export')}
          </PermissionButton>
          <Button onClick={() => void refresh()}>{t('common.refresh')}</Button>
        </Space>
      }
    >
      <SearchForm
        fields={[
          {
            name: 'payment_no',
            label: t('payment.payment_no'),
            placeholder: t('payment.payment_no_placeholder'),
          },
          {
            name: 'order_no',
            label: t('payment.order_no'),
            placeholder: t('payment.order_no_placeholder'),
          },
          {
            name: 'payment_method_id',
            label: t('payment.payment_method_id'),
            type: 'select',
            placeholder: t('payment.payment_method_id_placeholder'),
            options: paymentMethodOptions,
          },
          {
            name: 'user_id',
            label: t('payment.user_id'),
            placeholder: t('payment.user_id_placeholder'),
            advanced: true,
          },
          {
            name: 'status',
            label: t('payment.status'),
            type: 'select',
            placeholder: t('payment.status_placeholder'),
            options: createPaymentStatusOptions(t),
          },
          {
            name: 'time_field',
            label: t('payment.time_field'),
            type: 'select',
            placeholder: t('payment.time_field_placeholder'),
            options: [
              { label: t('payment.time_field_created_at'), value: 'created_at' },
              { label: t('payment.time_field_pay_time'), value: 'pay_time' },
            ],
          },
          {
            name: 'start_time',
            label: t('payment.start_time'),
            type: 'datetime',
            placeholder: t('payment.start_time_placeholder'),
            advanced: true,
          },
          {
            name: 'end_time',
            label: t('payment.end_time'),
            type: 'datetime',
            placeholder: t('payment.end_time_placeholder'),
            advanced: true,
          },
        ]}
        values={searchForm}
        onChange={onSearchFormChange}
        onSearch={handleSearch}
        onReset={handleReset}
      />
      <Table<PaymentRow>
        rowKey="id"
        loading={loading}
        columns={columns}
        dataSource={tableData}
        scroll={{ x: 1400 }}
        pagination={{
          current: pagination.page,
          pageSize: pagination.pageSize,
          total: pagination.total,
          showSizeChanger: true,
          showTotal: (total) => (total >= 100000 ? t('common.total', { total: '100000+' }) : t('common.total', { total })),
        }}
        onChange={(pager, _f, sorter) =>
          handlePaginatedTableChange({ pager, sorter, pagination, loadData, handleSortChange })
        }
      />
      <Drawer
        open={detailOpen}
        title={t('payment.payment_detail')}
        width={800}
        onClose={() => setDetailOpen(false)}
      >
        <Spin spinning={detailLoading}>
          {paymentDetail ? (
            <Descriptions bordered column={2} size="small">
              <Descriptions.Item label={t('payment.payment_no')}>
                {String(entityField(paymentDetail, 'payment_no', '-') ?? '-')}
              </Descriptions.Item>
              <Descriptions.Item label={t('payment.order_no')}>
                {String(entityField(paymentDetail, 'order_no', '-') ?? '-')}
              </Descriptions.Item>
              <Descriptions.Item label={t('payment.payment_method')}>
                {getPaymentMethodName(entityField(paymentDetail, 'payment_method'))}
              </Descriptions.Item>
              <Descriptions.Item label={t('payment.user_id')}>
                {String(entityField(paymentDetail, 'user_id', '-') ?? '-')}
              </Descriptions.Item>
              <Descriptions.Item label={t('payment.amount')}>
                {formatPaymentAmount(entityField(paymentDetail, 'amount'))}
              </Descriptions.Item>
              <Descriptions.Item label={t('payment.status')}>
                <Tag color={getPaymentStatusColor(String(entityField(paymentDetail, 'status', '')))}>
                  {getPaymentStatusText(t, String(entityField(paymentDetail, 'status', '')))}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label={t('payment.third_party_no')}>
                {String(entityField(paymentDetail, 'third_party_no', '-') || '-')}
              </Descriptions.Item>
              <Descriptions.Item label={t('payment.pay_time')}>
                {String(entityField(paymentDetail, 'pay_time', '-') || '-')}
              </Descriptions.Item>
              {entityField(paymentDetail, 'fail_reason') ? (
                <Descriptions.Item label={t('payment.fail_reason')} span={2}>
                  {String(entityField(paymentDetail, 'fail_reason', ''))}
                </Descriptions.Item>
              ) : null}
              {entityField(paymentDetail, 'remark') ? (
                <Descriptions.Item label={t('payment.remark')} span={2}>
                  {String(entityField(paymentDetail, 'remark', ''))}
                </Descriptions.Item>
              ) : null}
              <Descriptions.Item label={t('table.created_at')}>
                {formatPaymentDateTime(entityField(paymentDetail, 'created_at'))}
              </Descriptions.Item>
              <Descriptions.Item label={t('table.updated_at')}>
                {formatPaymentDateTime(entityField(paymentDetail, 'updated_at'))}
              </Descriptions.Item>
            </Descriptions>
          ) : null}
        </Spin>
      </Drawer>
    </PageContainer>
  )
}
