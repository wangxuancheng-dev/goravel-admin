import { useEffect, useMemo, useState } from 'react'
import { App, Descriptions, Drawer, Space, Spin, Table, Tag, Tooltip } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { EyeOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import {
  batchDeleteSystemLogs,
  cleanSystemLogs,
  deleteSystemLog,
  getSystemLogDetail,
  getSystemLogList,
  getSystemLogModuleOptions,
} from '@/api/log'
import { useListPage } from '@/hooks/useListPage'
import { handlePaginatedTableChange } from '@/utils/tableChange'
import { useCrudActions } from '@/hooks/useCrudActions'
import { usePermission } from '@/hooks/usePermission'
import { useUnhandledError } from '@/hooks/useUnhandledError'
import PageContainer from '@/components/PageContainer'
import SearchForm from '@/components/SearchForm'
import PermissionButton from '@/components/PermissionButton'
import LogCleanModal from './LogCleanModal'
import {
  createSystemLogSearchFields,
  extractLogDetail,
  formatSystemLogContext,
  formatSystemLogContextPreview,
  getSystemLogLevelColor,
  getSystemLogLevelLabel,
  getSystemLogModuleLabel,
  systemLogInitialSearchForm,
  transformSystemLogRow,
  type SystemLogRow,
} from './systemLog.config'

export default function SystemLogList() {
  const { t } = useTranslation()
  const { message, modal } = App.useApp()
  const showError = useUnhandledError()
  const { getButtonState } = usePermission()
  const [selectedRowKeys, setSelectedRowKeys] = useState<Array<string | number>>([])
  const [detailOpen, setDetailOpen] = useState(false)
  const [detailLoading, setDetailLoading] = useState(false)
  const [logDetail, setLogDetail] = useState<SystemLogRow | null>(null)
  const [moduleOptions, setModuleOptions] = useState<Array<{ label: string; value: string }>>([])
  const [cleanOpen, setCleanOpen] = useState(false)
  const [cleanLoading, setCleanLoading] = useState(false)

  const getModuleLabel = (module?: string) => getSystemLogModuleLabel(t, module)

  const {
    tableData,
    loading,
    pagination,
    searchForm,
    onSearchFormChange,
    loadData,
    handleSearch,
    handleReset,
    refresh,
    handleSortChange,
  } = useListPage<SystemLogRow>({
    fetchApi: getSystemLogList,
    initialSearchForm: { ...systemLogInitialSearchForm },
    defaultSort: 'id:desc',
    transformData: (row) => transformSystemLogRow(row as unknown as Record<string, unknown>),
    onSearch: () => setSelectedRowKeys([]),
    onReset: () => setSelectedRowKeys([]),
  })

  const { toolbar, confirmDelete } = useCrudActions({
    onRefresh: refresh,
    deleteApi: deleteSystemLog,
  })

  useEffect(() => {
    void (async () => {
      try {
        const res = await getSystemLogModuleOptions()
        const data = res.data as Record<string, unknown> | undefined
        const modules = Array.isArray(data?.modules) ? (data.modules as string[]) : []
        setModuleOptions(
          modules.map((module) => ({
            label: getModuleLabel(module),
            value: module,
          })),
        )
      } catch {
        setModuleOptions([])
      }
    })()
  }, [t])

  const handleView = async (row: SystemLogRow) => {
    setDetailOpen(true)
    setDetailLoading(true)
    setLogDetail(null)
    try {
      const res = await getSystemLogDetail(row.id)
      const raw = extractLogDetail<Record<string, unknown>>(res.data as Record<string, unknown>, [
        'system_log',
        'log',
      ])
      if (raw) {
        setLogDetail(transformSystemLogRow(raw))
      }
    } catch (error) {
      showError(error, t('common.operation_failed'))
      setDetailOpen(false)
    } finally {
      setDetailLoading(false)
    }
  }

  const handleBatchDelete = () => {
    if (!selectedRowKeys.length) return
    modal.confirm({
      title: t('common.delete_confirm'),
      content: t('log.batch_delete_confirm', { count: selectedRowKeys.length }),
      okType: 'danger',
      onOk: async () => {
        try {
          await batchDeleteSystemLogs(selectedRowKeys)
          message.success(t('common.delete_success'))
          setSelectedRowKeys([])
          await refresh()
        } catch (error) {
          showError(error, t('common.operation_failed'))
        }
      },
    })
  }

  const handleClean = async (days: number) => {
    setCleanLoading(true)
    try {
      await cleanSystemLogs({ days })
      message.success(t('log.clean_success'))
      setCleanOpen(false)
      setSelectedRowKeys([])
      await refresh()
    } catch (error) {
      showError(error, t('common.operation_failed'))
    } finally {
      setCleanLoading(false)
    }
  }

  const searchFields = useMemo(
    () => createSystemLogSearchFields(t, moduleOptions),
    [t, moduleOptions],
  )

  const columns: ColumnsType<SystemLogRow> = [
    { title: t('table.id'), dataIndex: 'id', width: 80, sorter: true },
    {
      title: t('log.level'),
      dataIndex: 'level',
      width: 100,
      render: (level: string) => (
        <Tag color={getSystemLogLevelColor(level)}>{getSystemLogLevelLabel(t, level)}</Tag>
      ),
    },
    {
      title: t('log.module'),
      dataIndex: 'module',
      width: 120,
      render: (module: string) => getModuleLabel(module),
    },
    {
      title: t('log.trace_id'),
      dataIndex: 'trace_id',
      width: 220,
      ellipsis: true,
      render: (traceId: string) => traceId || '-',
    },
    { title: t('log.message'), dataIndex: 'message', ellipsis: true },
    {
      title: t('log.context'),
      dataIndex: 'context',
      width: 200,
      ellipsis: true,
      render: (context: unknown) =>
        context ? (
          <Tooltip title={<pre style={{ margin: 0, maxWidth: 500, whiteSpace: 'pre-wrap' }}>{formatSystemLogContext(context)}</pre>}>
            <span style={{ cursor: 'pointer', color: '#1677ff' }}>{formatSystemLogContextPreview(context)}</span>
          </Tooltip>
        ) : (
          '-'
        ),
    },
    { title: t('log.time'), dataIndex: 'created_at', width: 180, sorter: true },
    {
      title: t('common.operation'),
      key: 'operation',
      width: 160,
      fixed: 'end',
      onCell: () => ({ style: { whiteSpace: 'nowrap' } }),
      render: (_, row) => (
        <Space size={4} wrap={false}>
          {getButtonState('system_log.show').show ? (
            <PermissionButton
              permission="system_log.show"
              type="link"
              icon={<EyeOutlined />}
              onClick={() => void handleView(row)}
            >
              {t('common.view')}
            </PermissionButton>
          ) : null}
          {getButtonState('system_log.destroy').show ? (
            <PermissionButton
              permission="system_log.destroy"
              type="link"
              danger
              onClick={() => confirmDelete(row.id, getModuleLabel(row.module))}
            >
              {t('common.delete')}
            </PermissionButton>
          ) : null}
        </Space>
      ),
    },
  ]

  return (
    <PageContainer
      title={t('menu.system_log')}
      extra={
        <Space>
          {selectedRowKeys.length > 0 && getButtonState('system_log.batch_delete').show ? (
            <PermissionButton permission="system_log.batch_delete" danger onClick={handleBatchDelete}>
              {t('common.delete_selected')} ({selectedRowKeys.length})
            </PermissionButton>
          ) : null}
          {getButtonState('system_log.clean').show ? (
            <PermissionButton permission="system_log.clean" onClick={() => setCleanOpen(true)}>
              {t('log.clean')}
            </PermissionButton>
          ) : null}
          {toolbar}
        </Space>
      }
    >
      <SearchForm
        fields={searchFields}
        values={searchForm}
        onChange={onSearchFormChange}
        onSearch={handleSearch}
        onReset={handleReset}
      />
      <Table<SystemLogRow>
        rowKey="id"
        loading={loading}
        columns={columns}
        dataSource={tableData}
        scroll={{ x: 1300 }}
        rowSelection={
          getButtonState('system_log.batch_delete').show
            ? {
                selectedRowKeys,
                onChange: (keys) => setSelectedRowKeys(keys as Array<string | number>),
              }
            : undefined
        }
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

      <Drawer open={detailOpen} title={t('log.detail')} width={1100} onClose={() => setDetailOpen(false)}>
        <Spin spinning={detailLoading}>
          {logDetail ? (
            <Descriptions bordered column={2} size="small">
              <Descriptions.Item label={t('table.id')}>{logDetail.id}</Descriptions.Item>
              <Descriptions.Item label={t('log.level')}>
                <Tag color={getSystemLogLevelColor(logDetail.level)}>
                  {getSystemLogLevelLabel(t, logDetail.level)}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label={t('log.module')}>{getModuleLabel(logDetail.module)}</Descriptions.Item>
              <Descriptions.Item label={t('log.trace_id')} span={2}>
                {logDetail.trace_id || '-'}
              </Descriptions.Item>
              <Descriptions.Item label={t('log.message')} span={2}>
                {logDetail.message}
              </Descriptions.Item>
              <Descriptions.Item label={t('log.context')} span={2}>
                {logDetail.context ? (
                  <pre style={{ margin: 0, maxHeight: 400, overflow: 'auto', fontSize: 12 }}>
                    {formatSystemLogContext(logDetail.context)}
                  </pre>
                ) : (
                  '-'
                )}
              </Descriptions.Item>
              <Descriptions.Item label={t('log.time')} span={2}>
                {logDetail.created_at}
              </Descriptions.Item>
            </Descriptions>
          ) : null}
        </Spin>
      </Drawer>

      <LogCleanModal
        open={cleanOpen}
        loading={cleanLoading}
        onClose={() => setCleanOpen(false)}
        onConfirm={handleClean}
      />
    </PageContainer>
  )
}
