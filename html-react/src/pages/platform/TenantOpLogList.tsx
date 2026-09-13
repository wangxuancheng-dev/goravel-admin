import { useMemo } from 'react'
import { Table, Tag, Tooltip } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useTranslation } from 'react-i18next'
import { getPlatformTenantOpLogList } from '@/api/platform'
import { useListPage } from '@/hooks/useListPage'
import { handlePaginatedTableChange } from '@/utils/tableChange'
import { entityField } from '@/utils/normalize'
import PageContainer from '@/components/PageContainer'
import SearchForm from '@/components/SearchForm'

interface OpLogRow {
  id: number | string
  code?: string
  op?: string
  status?: string
  message?: string
  batch_id?: string
  operator_name?: string
  started_at?: string
  finished_at?: string
}

function statusColor(status?: string) {
  switch (status) {
    case 'success':
      return 'success'
    case 'failed':
      return 'error'
    case 'running':
      return 'warning'
    default:
      return 'default'
  }
}

function shortMsg(msg?: string) {
  const s = String(msg || '')
  if (!s) return '—'
  if (s.length <= 48) return s
  return `${s.slice(0, 46)}…`
}

export default function PlatformTenantOpLogList() {
  const { t } = useTranslation()
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
  } = useListPage<OpLogRow>({
    fetchApi: getPlatformTenantOpLogList,
    initialSearchForm: { code: '', op: '', status: '', batch_id: '', operator: '' },
    defaultSort: 'id:desc',
    normalizeRows: false,
    transformData: (row) => ({
      id: entityField(row, 'id', '')!,
      code: String(entityField(row, 'code', '') ?? ''),
      op: String(entityField(row, 'op', '') ?? ''),
      status: String(entityField(row, 'status', '') ?? ''),
      message: String(entityField(row, 'message', '') ?? ''),
      batch_id: String(entityField(row, 'batch_id', '') ?? ''),
      operator_name: String(entityField(row, 'operator_name', '') ?? ''),
      started_at: String(entityField(row, 'started_at', '') ?? ''),
      finished_at: String(entityField(row, 'finished_at', '') ?? ''),
    }),
  })

  const columns: ColumnsType<OpLogRow> = useMemo(
    () => [
      { title: t('table.id'), dataIndex: 'id', width: 70, sorter: true },
      { title: t('tenant.code'), dataIndex: 'code', width: 110 },
      { title: t('tenant_op_log.op'), dataIndex: 'op', width: 100 },
      {
        title: t('tenant_op_log.status'),
        dataIndex: 'status',
        width: 100,
        render: (v: string) => <Tag color={statusColor(v)}>{v || '—'}</Tag>,
      },
      { title: t('tenant_op_log.operator'), dataIndex: 'operator_name', width: 120 },
      { title: t('tenant_op_log.batch_id'), dataIndex: 'batch_id', ellipsis: true },
      {
        title: t('tenant_op_log.message'),
        dataIndex: 'message',
        ellipsis: true,
        render: (v: string) => (
          <Tooltip title={v || ''}>
            <span>{shortMsg(v)}</span>
          </Tooltip>
        ),
      },
      { title: t('tenant_op_log.started_at'), dataIndex: 'started_at', width: 170 },
      { title: t('tenant_op_log.finished_at'), dataIndex: 'finished_at', width: 170 },
    ],
    [t],
  )

  return (
    <PageContainer title={t('menu.tenant_op_log')}>
      <SearchForm
        fields={[
          { name: 'code', label: t('tenant.code'), type: 'input' },
          {
            name: 'op',
            label: t('tenant_op_log.op'),
            type: 'select',
            options: [
              { label: 'migrate', value: 'migrate' },
              { label: 'seed', value: 'seed' },
              { label: 'backup', value: 'backup' },
              { label: 'restore', value: 'restore' },
            ],
          },
          {
            name: 'status',
            label: t('tenant_op_log.status'),
            type: 'select',
            options: [
              { label: 'queued', value: 'queued' },
              { label: 'running', value: 'running' },
              { label: 'success', value: 'success' },
              { label: 'failed', value: 'failed' },
            ],
          },
          { name: 'batch_id', label: t('tenant_op_log.batch_id'), type: 'input', advanced: true },
          { name: 'operator', label: t('tenant_op_log.operator'), type: 'input', advanced: true },
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
        onChange={(pager, _f, sorter) => {
          handlePaginatedTableChange({ pager, sorter, pagination, loadData, handleSortChange })
        }}
      />
    </PageContainer>
  )
}
