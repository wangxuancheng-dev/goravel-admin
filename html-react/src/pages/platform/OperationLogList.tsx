import { useMemo, useState } from 'react'
import { Button, Descriptions, Modal, Table, Tag } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useTranslation } from 'react-i18next'
import { getPlatformOperationLogDetail, getPlatformOperationLogList } from '@/api/platform'
import { useListPage } from '@/hooks/useListPage'
import { handlePaginatedTableChange } from '@/utils/tableChange'
import { entityField } from '@/utils/normalize'
import PageContainer from '@/components/PageContainer'
import SearchForm from '@/components/SearchForm'

interface OperationLogRow {
  id: number | string
  username?: string
  method?: string
  path?: string
  title?: string
  ip?: string
  status?: number | string
  duration?: number
  user_agent?: string
  request?: string
  created_at?: string
}

function formatRequest(raw?: string) {
  if (!raw) return ''
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    return raw
  }
}

export default function PlatformOperationLogList() {
  const { t } = useTranslation()
  const [detailOpen, setDetailOpen] = useState(false)
  const [detail, setDetail] = useState<OperationLogRow | null>(null)
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
  } = useListPage<OperationLogRow>({
    fetchApi: getPlatformOperationLogList,
    initialSearchForm: { username: '', method: '', path: '', status: '' },
    defaultSort: 'id:desc',
    normalizeRows: false,
    transformData: (row) => ({
      id: entityField(row, 'id', '')!,
      username: String(entityField(row, 'username', '') ?? ''),
      method: String(entityField(row, 'method', '') ?? ''),
      path: String(entityField(row, 'path', '') ?? ''),
      title: String(entityField(row, 'title', '') ?? ''),
      ip: String(entityField(row, 'ip', '') ?? ''),
      status: entityField(row, 'status', 0) as number | string,
      duration: Number(entityField(row, 'duration', 0) ?? 0),
      user_agent: String(entityField(row, 'user_agent', '') ?? ''),
      request: String(entityField(row, 'request', '') ?? ''),
      created_at: String(entityField(row, 'created_at', '') ?? ''),
    }),
  })

  const columns: ColumnsType<OperationLogRow> = useMemo(
    () => [
      { title: t('table.id'), dataIndex: 'id', width: 70, sorter: true },
      { title: t('log.admin'), dataIndex: 'username', width: 120 },
      { title: t('log.method'), dataIndex: 'method', width: 90 },
      { title: t('log.title'), dataIndex: 'title', ellipsis: true },
      { title: t('log.path'), dataIndex: 'path', ellipsis: true },
      { title: t('log.ip'), dataIndex: 'ip', width: 130 },
      {
        title: t('table.status'),
        dataIndex: 'status',
        width: 90,
        render: (v) => (
          <Tag color={Number(v) === 1 ? 'success' : 'error'}>
            {Number(v) === 1 ? t('log.success') : t('log.failed')}
          </Tag>
        ),
      },
      { title: 'ms', dataIndex: 'duration', width: 80 },
      { title: t('log.operation_time'), dataIndex: 'created_at', width: 170 },
      {
        title: t('table.operation'),
        key: 'action',
        width: 90,
        fixed: 'right',
        render: (_, row) => (
          <Button
            type="link"
            onClick={async () => {
              try {
                const res = await getPlatformOperationLogDetail(row.id)
                const data = (res as { data?: { operation_log?: OperationLogRow } & OperationLogRow })?.data
                setDetail(data?.operation_log || data || row)
                setDetailOpen(true)
              } catch {
                setDetail(row)
                setDetailOpen(true)
              }
            }}
          >
            {t('common.view')}
          </Button>
        ),
      },
    ],
    [t],
  )

  return (
    <PageContainer title={t('menu.operation_log')}>
      <SearchForm
        fields={[
          { name: 'username', label: t('log.admin'), type: 'input' },
          {
            name: 'method',
            label: t('log.method'),
            type: 'select',
            options: [
              { label: 'POST', value: 'POST' },
              { label: 'PUT', value: 'PUT' },
              { label: 'PATCH', value: 'PATCH' },
              { label: 'DELETE', value: 'DELETE' },
            ],
          },
          { name: 'path', label: t('log.path'), type: 'input' },
          {
            name: 'status',
            label: t('table.status'),
            type: 'select',
            options: [
              { label: t('log.success'), value: '1' },
              { label: t('log.failed'), value: '0' },
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
        scroll={{ x: 1200 }}
        onChange={(pager, _f, sorter) => {
          handlePaginatedTableChange({ pager, sorter, pagination, loadData, handleSortChange })
        }}
      />
      <Modal
        title={t('log.detail')}
        open={detailOpen}
        onCancel={() => setDetailOpen(false)}
        footer={null}
        width={900}
        destroyOnClose
      >
        {detail ? (
          <Descriptions bordered size="small" column={2}>
            <Descriptions.Item label={t('table.id')}>{detail.id}</Descriptions.Item>
            <Descriptions.Item label={t('log.admin')}>{detail.username || '-'}</Descriptions.Item>
            <Descriptions.Item label={t('log.method')}>{detail.method || '-'}</Descriptions.Item>
            <Descriptions.Item label={t('log.title')}>{detail.title || '-'}</Descriptions.Item>
            <Descriptions.Item label={t('log.path')} span={2}>
              {detail.path || '-'}
            </Descriptions.Item>
            <Descriptions.Item label={t('log.ip')}>{detail.ip || '-'}</Descriptions.Item>
            <Descriptions.Item label={t('table.status')}>
              <Tag color={Number(detail.status) === 1 ? 'success' : 'error'}>
                {Number(detail.status) === 1 ? t('log.success') : t('log.failed')}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label={t('log.operation_time')}>{detail.created_at || '-'}</Descriptions.Item>
            <Descriptions.Item label="Duration">{detail.duration ?? '-'} ms</Descriptions.Item>
            <Descriptions.Item label={t('log.user_agent')} span={2}>
              {detail.user_agent || '-'}
            </Descriptions.Item>
            <Descriptions.Item label={t('log.request')} span={2}>
              <pre style={{ margin: 0, whiteSpace: 'pre-wrap', maxHeight: 280, overflow: 'auto' }}>
                {formatRequest(detail.request) || '-'}
              </pre>
            </Descriptions.Item>
          </Descriptions>
        ) : null}
      </Modal>
    </PageContainer>
  )
}
