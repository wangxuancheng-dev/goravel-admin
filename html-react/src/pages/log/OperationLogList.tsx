import { useCallback, useEffect, useMemo, useState } from 'react'
import {
  App,
  Button,
  Descriptions,
  Drawer,
  Popover,
  Space,
  Spin,
  Table,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { CopyOutlined, EyeOutlined, SettingOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import {
  batchDeleteOperationLogs,
  cleanOperationLogs,
  deleteOperationLog,
  getOperationLogDetail,
  getOperationLogList,
  getOperationLogTitleOptions,
} from '@/api/log'
import { useListPage } from '@/hooks/useListPage'
import { handlePaginatedTableChange } from '@/utils/tableChange'
import { useCrudActions } from '@/hooks/useCrudActions'
import { usePermission } from '@/hooks/usePermission'
import { useUnhandledError } from '@/hooks/useUnhandledError'
import { useColumnSetting } from '@/hooks/useColumnSetting'
import PageContainer from '@/components/PageContainer'
import ColumnSettingDialog from '@/components/ColumnSettingDialog'
import SearchForm from '@/components/SearchForm'
import PermissionButton from '@/components/PermissionButton'
import { getOperationTitle } from '@/utils/operationTitle'
import { formatDateTime } from '@/utils/dateUtils'
import {
  OPERATION_LOG_MAX_TIME_RANGE_MONTHS,
  validateTimeRange,
} from '@/utils/timeRangeValidator'
import LogCleanModal from './LogCleanModal'
import {
  createOperationLogInitialSearchForm,
  createOperationLogSearchFields,
  extractLogDetail,
  formatChangeValue,
  formatRequestParamsFull,
  getRequestPreview,
  hasRequestParams,
  transformOperationLogRow,
  type OperationLogRow,
} from './operationLog.config'

export default function OperationLogList() {
  const { t, i18n } = useTranslation()
  const { message, modal } = App.useApp()
  const showError = useUnhandledError()
  const { getButtonState } = usePermission()
  const [selectedRowKeys, setSelectedRowKeys] = useState<Array<string | number>>([])
  const [detailOpen, setDetailOpen] = useState(false)
  const [detailLoading, setDetailLoading] = useState(false)
  const [logDetail, setLogDetail] = useState<OperationLogRow | null>(null)
  const [titleOptions, setTitleOptions] = useState<string[]>([])
  const [cleanOpen, setCleanOpen] = useState(false)
  const [cleanLoading, setCleanLoading] = useState(false)

  const translateOperationTitle = useCallback(
    (title: string) => getOperationTitle(t, title),
    [t],
  )

  const {
    tableData,
    loading,
    pagination,
    searchForm,
    onSearchFormChange,
    loadData,
    handleSearch: baseHandleSearch,
    handleReset,
    handleSortChange,
    refresh,
  } = useListPage<OperationLogRow>({
    fetchApi: getOperationLogList,
    initialSearchForm: createOperationLogInitialSearchForm(),
    defaultSort: 'id:desc',
    transformData: (row) => transformOperationLogRow(row as unknown as Record<string, unknown>),
    onSearch: () => setSelectedRowKeys([]),
    onReset: () => setSelectedRowKeys([]),
  })

  const { toolbar, confirmDelete } = useCrudActions({
    onRefresh: refresh,
    deleteApi: deleteOperationLog,
  })

  useEffect(() => {
    void (async () => {
      try {
        const res = await getOperationLogTitleOptions()
        const data = res.data as Record<string, unknown> | undefined
        const titles = Array.isArray(data?.titles) ? (data.titles as string[]) : []
        const mergedSet = new Set<string>()
        titles.forEach((title) => {
          if (title && typeof title === 'string') {
            const trimmed = title.trim()
            if (trimmed && trimmed !== 'operation.unknown' && !trimmed.startsWith('operation.')) {
              mergedSet.add(trimmed)
            }
          }
        })
        const sorted = Array.from(mergedSet).sort((a, b) =>
          translateOperationTitle(a).localeCompare(translateOperationTitle(b), i18n.language || 'zh-CN'),
        )
        setTitleOptions(sorted)
      } catch {
        setTitleOptions([])
      }
    })()
  }, [i18n.language, translateOperationTitle])

  const getCurrentTimeString = () => formatDateTime(new Date())

  const validateTimeRangeForSearch = () => {
    if (!searchForm.start_time) return true
    const endTime = String(searchForm.end_time || getCurrentTimeString())
    const validation = validateTimeRange(
      String(searchForm.start_time),
      endTime,
      OPERATION_LOG_MAX_TIME_RANGE_MONTHS,
    )
    if (!validation.valid) {
      const errorMessage = validation.errorKey
        ? validation.errorParams
          ? t(`log.${validation.errorKey}`, validation.errorParams)
          : t(`log.${validation.errorKey}`)
        : validation.error
      message.warning(errorMessage)
      return false
    }
    return true
  }

  const handleSearch = () => {
    if (!validateTimeRangeForSearch()) return
    setSelectedRowKeys([])
    baseHandleSearch()
  }

  const copyText = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text)
      message.success(t('common.copy_success'))
    } catch {
      message.error(t('common.copy_failed'))
    }
  }

  const handleView = async (row: OperationLogRow) => {
    setDetailOpen(true)
    setDetailLoading(true)
    setLogDetail(null)
    try {
      const res = await getOperationLogDetail(row.id)
      const raw = extractLogDetail<Record<string, unknown>>(res.data as Record<string, unknown>, [
        'operation_log',
        'log',
      ])
      if (raw) {
        setLogDetail(transformOperationLogRow(raw))
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
          await batchDeleteOperationLogs(selectedRowKeys)
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
      await cleanOperationLogs({ days })
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
    () => createOperationLogSearchFields(t, titleOptions, translateOperationTitle),
    [t, titleOptions, translateOperationTitle],
  )

  const columns: ColumnsType<OperationLogRow> = [
    { title: t('table.id'), dataIndex: 'id', width: 80, sorter: true },
    {
      title: t('log.admin'),
      dataIndex: ['admin', 'username'],
      width: 120,
      render: (_, row) => row.admin?.username || '-',
    },
    {
      title: t('log.title'),
      dataIndex: 'title',
      width: 200,
      render: (title: string) => translateOperationTitle(title),
    },
    { title: t('log.method'), dataIndex: 'method', width: 100 },
    { title: t('log.path'), dataIndex: 'path', ellipsis: true },
    { title: t('log.ip'), dataIndex: 'ip', width: 140 },
    {
      title: t('log.request_params'),
      dataIndex: 'request',
      width: 250,
      ellipsis: true,
      render: (_, row) => {
        const request = row.request ?? row.params
        if (!hasRequestParams(request)) return '-'
        const fullText = formatRequestParamsFull(request)
        return (
          <Popover
            title={t('log.request_params')}
            content={
              <div style={{ maxWidth: 600 }}>
                <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 8 }}>
                  <Button type="link" size="small" icon={<CopyOutlined />} onClick={() => void copyText(fullText)}>
                    {t('common.copy')}
                  </Button>
                </div>
                <pre style={{ margin: 0, maxHeight: 400, overflow: 'auto', fontSize: 12 }}>{fullText}</pre>
              </div>
            }
          >
            <span style={{ cursor: 'pointer', fontFamily: 'monospace', fontSize: 12 }}>
              {getRequestPreview(request)}
            </span>
          </Popover>
        )
      },
    },
    { title: t('log.operation_time'), dataIndex: 'created_at', width: 180, sorter: true },
    {
      title: t('common.operation'),
      key: 'operation',
      width: 160,
      fixed: 'end',
      onCell: () => ({ style: { whiteSpace: 'nowrap' } }),
      render: (_, row) => (
        <Space size={4} wrap={false}>
          {getButtonState('operation_log.show').show ? (
            <PermissionButton
              permission="operation_log.show"
              type="link"
              icon={<EyeOutlined />}
              onClick={() => void handleView(row)}
            >
              {t('common.view')}
            </PermissionButton>
          ) : null}
          {getButtonState('operation_log.destroy').show ? (
            <PermissionButton
              permission="operation_log.destroy"
              type="link"
              danger
              onClick={() => confirmDelete(row.id, translateOperationTitle(row.title || ''))}
            >
              {t('common.delete')}
            </PermissionButton>
          ) : null}
        </Space>
      ),
    },
  ]

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
  } = useColumnSetting('operation_log', columns)

  const detailRequest = logDetail?.params ?? logDetail?.request
  const detailRequestText = formatRequestParamsFull(detailRequest)

  return (
    <PageContainer
      title={t('menu.operation_log')}
      extra={
        <Space>
          {selectedRowKeys.length > 0 && getButtonState('operation_log.batch_delete').show ? (
            <PermissionButton permission="operation_log.batch_delete" danger onClick={handleBatchDelete}>
              {t('common.delete_selected')} ({selectedRowKeys.length})
            </PermissionButton>
          ) : null}
          {getButtonState('operation_log.clean').show ? (
            <PermissionButton permission="operation_log.clean" onClick={() => setCleanOpen(true)}>
              {t('log.clean')}
            </PermissionButton>
          ) : null}
          {toolbar}
          <Button icon={<SettingOutlined />} onClick={openColumnSetting}>
            {t('common.column_setting')}
          </Button>
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
      <Table<OperationLogRow>
        rowKey="id"
        loading={loading}
        columns={filteredColumns}
        dataSource={tableData}
        scroll={{ x: 1400 }}
        rowSelection={
          getButtonState('operation_log.batch_delete').show
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

      <Drawer
        open={detailOpen}
        title={t('log.detail')}
        width={1100}
        onClose={() => setDetailOpen(false)}
      >
        <Spin spinning={detailLoading}>
          {logDetail ? (
            <Descriptions bordered column={2} size="small">
              <Descriptions.Item label={t('table.id')}>{logDetail.id}</Descriptions.Item>
              <Descriptions.Item label={t('log.admin')}>{logDetail.admin?.username || '-'}</Descriptions.Item>
              <Descriptions.Item label={t('log.method')}>{logDetail.method}</Descriptions.Item>
              <Descriptions.Item label={t('log.path')}>{logDetail.path}</Descriptions.Item>
              <Descriptions.Item label={t('log.ip')}>{logDetail.ip}</Descriptions.Item>
              <Descriptions.Item label={t('log.status_code')}>{logDetail.status_code}</Descriptions.Item>
              <Descriptions.Item label={t('log.operation_time')} span={2}>
                {logDetail.created_at}
              </Descriptions.Item>
              <Descriptions.Item label={t('log.request_params')} span={2}>
                <div>
                  <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 8 }}>
                    <Button type="link" size="small" icon={<CopyOutlined />} onClick={() => void copyText(detailRequestText)}>
                      {t('common.copy')}
                    </Button>
                  </div>
                  <pre style={{ margin: 0, maxHeight: 400, overflow: 'auto', fontSize: 12 }}>{detailRequestText}</pre>
                </div>
              </Descriptions.Item>
              {logDetail.changes?.length ? (
                <Descriptions.Item label={t('log.changes')} span={2}>
                  <Table
                    size="small"
                    bordered
                    pagination={false}
                    rowKey={(row, index) => `${row.field}-${index}`}
                    dataSource={logDetail.changes}
                    columns={[
                      { title: t('log.changes_field'), dataIndex: 'field', width: 180 },
                      {
                        title: t('log.changes_old'),
                        dataIndex: 'old',
                        render: (value: unknown) => (
                          <span style={{ color: '#ff4d4f', fontFamily: 'monospace', fontSize: 12, whiteSpace: 'pre-wrap' }}>
                            {formatChangeValue(value)}
                          </span>
                        ),
                      },
                      {
                        title: t('log.changes_new'),
                        dataIndex: 'new',
                        render: (value: unknown) => (
                          <span style={{ color: '#52c41a', fontFamily: 'monospace', fontSize: 12, whiteSpace: 'pre-wrap' }}>
                            {formatChangeValue(value)}
                          </span>
                        ),
                      },
                    ]}
                  />
                </Descriptions.Item>
              ) : null}
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
      <ColumnSettingDialog
        open={columnSettingOpen}
        onClose={closeColumnSetting}
        allColumns={allColumns}
        visibleColumns={visibleColumns}
        columnOrder={columnOrder}
        fixedColumns={fixedColumns}
        onConfirm={handleColumnSettingConfirm}
      />
    </PageContainer>
  )
}
