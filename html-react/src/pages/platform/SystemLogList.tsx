import { useEffect, useMemo, useState } from 'react'
import { Alert, Button, Descriptions, Modal, Space, Table, Tag, Tooltip } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useTranslation } from 'react-i18next'
import { useSearchParams } from 'react-router-dom'
import {
  getPlatformSystemLogDetail,
  getPlatformSystemLogList,
  getPlatformTenantList,
} from '@/api/platform'
import { useListPage } from '@/hooks/useListPage'
import { handlePaginatedTableChange } from '@/utils/tableChange'
import PageContainer from '@/components/PageContainer'
import SearchForm from '@/components/SearchForm'
import {
  formatSystemLogContext,
  formatSystemLogContextPreview,
  getSystemLogLevelColor,
  getSystemLogLevelLabel,
  systemLogInitialSearchForm,
  transformSystemLogRow,
  type SystemLogRow,
} from '@/pages/log/systemLog.config'

export default function PlatformSystemLogList() {
  const { t } = useTranslation()
  const [searchParams] = useSearchParams()
  const [tenantOptions, setTenantOptions] = useState<Array<{ label: string; value: string }>>([])
  const [detailOpen, setDetailOpen] = useState(false)
  const [detail, setDetail] = useState<SystemLogRow | null>(null)

  const initialSearchForm = {
    tenant_code: searchParams.get('code') || '',
    ...systemLogInitialSearchForm,
  }

  const {
    tableData,
    loading,
    pagination,
    searchForm,
    setSearchForm,
    onSearchFormChange,
    loadData,
    handleSearch,
    handleReset,
    handleSortChange,
  } = useListPage<SystemLogRow>({
    fetchApi: getPlatformSystemLogList,
    initialSearchForm,
    defaultSort: 'id:desc',
    normalizeRows: false,
    transformData: (row) => transformSystemLogRow(row as unknown as Record<string, unknown>),
  })

  useEffect(() => {
    void (async () => {
      try {
        const res = await getPlatformTenantList({ page: 1, page_size: 200, order_by: 'code:asc' })
        const list = ((res as { data?: { list?: Array<{ code?: string }> } })?.data?.list || [])
        setTenantOptions(list.map((row) => ({ label: row.code || '', value: row.code || '' })).filter((o) => o.value))
      } catch {
        /* ignore */
      }
    })()
  }, [])

  useEffect(() => {
    const code = searchParams.get('code') || ''
    if (code === String(searchForm.tenant_code || '')) return
    setSearchForm({ ...searchForm, tenant_code: code })
    const timer = window.setTimeout(() => {
      void loadData({ currentPage: 1 })
    }, 0)
    return () => window.clearTimeout(timer)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchParams])

  const scopeHint = searchForm.tenant_code
    ? t('platform.system_log_scope_tenant', { code: String(searchForm.tenant_code) })
    : t('platform.system_log_scope_platform')

  const searchFields = useMemo(
    () => [
      {
        name: 'tenant_code',
        label: t('tenant.code'),
        type: 'select' as const,
        allowClear: true,
        options: [
          { label: t('platform.system_log_platform_opt'), value: '' },
          ...tenantOptions,
        ],
      },
      {
        name: 'level',
        label: t('log.level'),
        type: 'select' as const,
        options: [
          { label: 'error', value: 'error' },
          { label: 'warning', value: 'warning' },
          { label: 'info', value: 'info' },
          { label: 'debug', value: 'debug' },
        ],
      },
      { name: 'module', label: t('log.module'), type: 'input' as const },
      { name: 'trace_id', label: t('log.trace_id'), type: 'input' as const },
      { name: 'message', label: t('log.message'), type: 'input' as const },
    ],
    [t, tenantOptions],
  )

  const columns: ColumnsType<SystemLogRow> = useMemo(
    () => [
      { title: t('table.id'), dataIndex: 'id', width: 70, sorter: true },
      {
        title: t('log.level'),
        dataIndex: 'level',
        width: 100,
        render: (v: string) => <Tag color={getSystemLogLevelColor(v)}>{getSystemLogLevelLabel(t, v)}</Tag>,
      },
      { title: t('log.module'), dataIndex: 'module', width: 140 },
      { title: t('log.trace_id'), dataIndex: 'trace_id', ellipsis: true },
      { title: t('log.message'), dataIndex: 'message', ellipsis: true },
      {
        title: t('log.context'),
        dataIndex: 'context',
        ellipsis: true,
        render: (v: SystemLogRow['context']) =>
          v ? (
            <Tooltip title={formatSystemLogContext(v)}>
              <span>{formatSystemLogContextPreview(v)}</span>
            </Tooltip>
          ) : (
            '-'
          ),
      },
      { title: t('log.time'), dataIndex: 'created_at', width: 170 },
      {
        title: t('table.operation'),
        key: 'operation',
        width: 90,
        fixed: 'right',
        render: (_, row) => (
          <Button
            type="link"
            onClick={() => {
              void (async () => {
                try {
                  const res = await getPlatformSystemLogDetail(row.id, {
                    tenant_code: searchForm.tenant_code || undefined,
                  })
                  const raw = (res as { data?: { log?: Record<string, unknown> } })?.data?.log || row
                  setDetail(transformSystemLogRow(raw as Record<string, unknown>))
                  setDetailOpen(true)
                } catch {
                  /* ignore */
                }
              })()
            }}
          >
            {t('common.view')}
          </Button>
        ),
      },
    ],
    [t, searchForm.tenant_code],
  )

  return (
    <PageContainer title={t('menu.system_log')}>
      <Alert type="info" showIcon style={{ marginBottom: 12 }} message={scopeHint} />
      <SearchForm
        fields={searchFields}
        values={searchForm}
        onChange={onSearchFormChange}
        onSearch={() => void handleSearch()}
        onReset={() => void handleReset()}
      />
      <Space style={{ marginBottom: 12 }}>
        <Button onClick={() => void loadData()}>{t('common.refresh')}</Button>
      </Space>
      <Table
        rowKey="id"
        loading={loading}
        columns={columns}
        dataSource={tableData}
        pagination={pagination}
        scroll={{ x: 1100 }}
        onChange={handlePaginatedTableChange(handleSortChange, loadData)}
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
            <Descriptions.Item label={t('log.level')}>
              <Tag color={getSystemLogLevelColor(detail.level)}>{getSystemLogLevelLabel(t, detail.level)}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label={t('log.module')}>{detail.module || '-'}</Descriptions.Item>
            <Descriptions.Item label={t('log.trace_id')}>{detail.trace_id || '-'}</Descriptions.Item>
            <Descriptions.Item label={t('log.message')} span={2}>
              {detail.message}
            </Descriptions.Item>
            <Descriptions.Item label={t('log.context')} span={2}>
              {detail.context ? (
                <pre style={{ margin: 0, whiteSpace: 'pre-wrap', maxHeight: 280, overflow: 'auto' }}>
                  {formatSystemLogContext(detail.context)}
                </pre>
              ) : (
                '-'
              )}
            </Descriptions.Item>
            <Descriptions.Item label={t('log.time')} span={2}>
              {detail.created_at || '-'}
            </Descriptions.Item>
          </Descriptions>
        ) : null}
      </Modal>
    </PageContainer>
  )
}
