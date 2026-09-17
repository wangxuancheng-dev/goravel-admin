import { useMemo, useState } from 'react'
import { Button, Descriptions, Modal, Table, Tag } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useTranslation } from 'react-i18next'
import { getPlatformLoginLogDetail, getPlatformLoginLogList } from '@/api/platform'
import { useListPage } from '@/hooks/useListPage'
import { handlePaginatedTableChange } from '@/utils/tableChange'
import { entityField } from '@/utils/normalize'
import PageContainer from '@/components/PageContainer'
import SearchForm from '@/components/SearchForm'

interface LoginLogRow {
  id: number | string
  username?: string
  ip?: string
  location?: string
  status?: number | string
  message?: string
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

export default function PlatformLoginLogList() {
  const { t } = useTranslation()
  const [detailOpen, setDetailOpen] = useState(false)
  const [detail, setDetail] = useState<LoginLogRow | null>(null)
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
  } = useListPage<LoginLogRow>({
    fetchApi: getPlatformLoginLogList,
    initialSearchForm: { username: '', ip: '', status: '' },
    defaultSort: 'id:desc',
    normalizeRows: false,
    transformData: (row) => ({
      id: entityField(row, 'id', '')!,
      username: String(entityField(row, 'username', '') ?? ''),
      ip: String(entityField(row, 'ip', '') ?? ''),
      location: String(entityField(row, 'location', '') ?? ''),
      status: entityField(row, 'status', 0) as number | string,
      message: String(entityField(row, 'message', '') ?? ''),
      user_agent: String(entityField(row, 'user_agent', '') ?? ''),
      request: String(entityField(row, 'request', '') ?? ''),
      created_at: String(entityField(row, 'created_at', '') ?? ''),
    }),
  })

  const translateMessage = (messageKey?: string) => {
    if (!messageKey) return '-'
    const key = `log.${messageKey}`
    const translated = t(key)
    return translated !== key ? translated : messageKey
  }

  const columns: ColumnsType<LoginLogRow> = useMemo(
    () => [
      { title: t('table.id'), dataIndex: 'id', width: 70, sorter: true },
      { title: t('log.admin'), dataIndex: 'username', width: 140 },
      { title: t('log.ip'), dataIndex: 'ip', width: 140 },
      { title: t('log.location'), dataIndex: 'location', width: 140 },
      {
        title: t('table.status'),
        dataIndex: 'status',
        width: 100,
        render: (v) => (
          <Tag color={Number(v) === 1 ? 'success' : 'error'}>
            {Number(v) === 1 ? t('log.success') : t('log.failed')}
          </Tag>
        ),
      },
      {
        title: t('log.message'),
        dataIndex: 'message',
        ellipsis: true,
        render: (v: string) => translateMessage(v),
      },
      { title: t('log.login_time'), dataIndex: 'created_at', width: 170 },
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
                const res = await getPlatformLoginLogDetail(row.id)
                const data = (res as { data?: { login_log?: LoginLogRow } & LoginLogRow })?.data
                setDetail(data?.login_log || data || row)
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
    <PageContainer title={t('menu.login_log')}>
      <SearchForm
        fields={[
          { name: 'username', label: t('log.admin'), type: 'input' },
          { name: 'ip', label: t('log.ip'), type: 'input' },
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
        scroll={{ x: 1100 }}
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
            <Descriptions.Item label={t('log.ip')}>{detail.ip || '-'}</Descriptions.Item>
            <Descriptions.Item label={t('log.location')}>{detail.location || '-'}</Descriptions.Item>
            <Descriptions.Item label={t('table.status')}>
              <Tag color={Number(detail.status) === 1 ? 'success' : 'error'}>
                {Number(detail.status) === 1 ? t('log.success') : t('log.failed')}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label={t('log.login_time')}>{detail.created_at || '-'}</Descriptions.Item>
            <Descriptions.Item label={t('log.user_agent')} span={2}>
              {detail.user_agent || '-'}
            </Descriptions.Item>
            <Descriptions.Item label={t('log.message')} span={2}>
              {translateMessage(detail.message)}
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
