import { useMemo } from 'react'
import { App, Button, Table, Tag, Tooltip } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useTranslation } from 'react-i18next'
import { getPlatformAlertDeliveryList, retryPlatformAlertDelivery } from '@/api/platform'
import { useListPage } from '@/hooks/useListPage'
import { handlePaginatedTableChange } from '@/utils/tableChange'
import { entityField } from '@/utils/normalize'
import { useUnhandledError } from '@/hooks/useUnhandledError'
import { isPlatformOwner } from '@/utils/platformRequest'
import PageContainer from '@/components/PageContainer'
import SearchForm from '@/components/SearchForm'

interface AlertRow {
  id: number | string
  channel?: string
  event?: string
  tenant_code?: string
  op?: string
  status?: string
  http_status?: number
  target_masked?: string
  error_message?: string
  attempt?: number
  created_at?: string
  delivered_at?: string
}

function statusColor(status?: string) {
  if (status === 'success') return 'success'
  if (status === 'failed') return 'error'
  return 'default'
}

export default function PlatformAlertDeliveryList() {
  const { t } = useTranslation()
  const { message } = App.useApp()
  const showError = useUnhandledError()
  const owner = isPlatformOwner()
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
  } = useListPage<AlertRow>({
    fetchApi: getPlatformAlertDeliveryList,
    initialSearchForm: { event: '', status: '', channel: '', tenant_code: '' },
    defaultSort: 'id:desc',
    normalizeRows: false,
    transformData: (row) => ({
      id: entityField(row, 'id', '')!,
      channel: String(entityField(row, 'channel', '') ?? ''),
      event: String(entityField(row, 'event', '') ?? ''),
      tenant_code: String(entityField(row, 'tenant_code', '') ?? ''),
      op: String(entityField(row, 'op', '') ?? ''),
      status: String(entityField(row, 'status', '') ?? ''),
      http_status: Number(entityField(row, 'http_status', 0) ?? 0),
      target_masked: String(entityField(row, 'target_masked', '') ?? ''),
      error_message: String(entityField(row, 'error_message', '') ?? ''),
      attempt: Number(entityField(row, 'attempt', 1) ?? 1),
      created_at: String(entityField(row, 'created_at', '') ?? ''),
      delivered_at: String(entityField(row, 'delivered_at', '') ?? ''),
    }),
  })

  const onRetry = async (row: AlertRow) => {
    try {
      await retryPlatformAlertDelivery(row.id)
      message.success(t('platform_alert.retry_success'))
      void loadData()
    } catch (e) {
      showError(e, t('common.operation_failed'))
    }
  }

  const columns: ColumnsType<AlertRow> = useMemo(
    () => [
      { title: t('table.id'), dataIndex: 'id', width: 70, sorter: true },
      { title: t('platform_alert.channel'), dataIndex: 'channel', width: 90 },
      { title: t('platform_alert.event'), dataIndex: 'event', width: 160 },
      { title: t('tenant.code'), dataIndex: 'tenant_code', width: 110 },
      { title: t('tenant_op_log.op'), dataIndex: 'op', width: 90 },
      {
        title: t('platform_alert.status'),
        dataIndex: 'status',
        width: 100,
        render: (v: string) => <Tag color={statusColor(v)}>{v || '—'}</Tag>,
      },
      { title: t('platform_alert.http_status'), dataIndex: 'http_status', width: 90 },
      { title: t('platform_alert.attempt'), dataIndex: 'attempt', width: 80 },
      {
        title: t('platform_alert.target'),
        dataIndex: 'target_masked',
        ellipsis: true,
      },
      {
        title: t('platform_alert.error'),
        dataIndex: 'error_message',
        ellipsis: true,
        render: (v: string) => (
          <Tooltip title={v || ''}>
            <span>{v ? (v.length > 40 ? `${v.slice(0, 38)}…` : v) : '—'}</span>
          </Tooltip>
        ),
      },
      { title: t('table.created_at'), dataIndex: 'created_at', width: 170 },
      {
        title: t('table.actions'),
        key: 'actions',
        width: 100,
        fixed: 'right',
        render: (_: unknown, row) =>
          owner && row.channel === 'webhook' ? (
            <Button type="link" size="small" onClick={() => void onRetry(row)}>
              {t('platform_alert.retry')}
            </Button>
          ) : (
            '—'
          ),
      },
    ],
    [t, owner],
  )

  return (
    <PageContainer title={t('menu.platform_alert')}>
      <SearchForm
        fields={[
          { name: 'tenant_code', label: t('tenant.code'), type: 'input' },
          {
            name: 'channel',
            label: t('platform_alert.channel'),
            type: 'select',
            options: [
              { label: 'webhook', value: 'webhook' },
              { label: 'mail', value: 'mail' },
            ],
          },
          {
            name: 'status',
            label: t('platform_alert.status'),
            type: 'select',
            options: [
              { label: 'success', value: 'success' },
              { label: 'failed', value: 'failed' },
              { label: 'pending', value: 'pending' },
            ],
          },
          { name: 'event', label: t('platform_alert.event'), type: 'input' },
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
        scroll={{ x: 1200 }}
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
    </PageContainer>
  )
}
