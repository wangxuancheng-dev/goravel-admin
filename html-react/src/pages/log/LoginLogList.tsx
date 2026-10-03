import { useMemo, useState } from 'react'
import { App, Button, Descriptions, Drawer, Space, Spin, Table, Tag } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { EyeOutlined, SettingOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import {
  batchDeleteLoginLogs,
  cleanLoginLogs,
  deleteLoginLog,
  getLoginLogDetail,
  getLoginLogList,
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
import LogCleanModal from './LogCleanModal'
import {
  createLoginLogSearchFields,
  extractLogDetail,
  formatRequestPreview,
  loginLogInitialSearchForm,
  transformLoginLogRow,
  translateLoginMessage,
  type LoginLogRow,
} from './loginLog.config'

export default function LoginLogList() {
  const { t } = useTranslation()
  const { message, modal } = App.useApp()
  const showError = useUnhandledError()
  const { getButtonState } = usePermission()
  const [selectedRowKeys, setSelectedRowKeys] = useState<Array<string | number>>([])
  const [detailOpen, setDetailOpen] = useState(false)
  const [detailLoading, setDetailLoading] = useState(false)
  const [logDetail, setLogDetail] = useState<LoginLogRow | null>(null)
  const [cleanOpen, setCleanOpen] = useState(false)
  const [cleanLoading, setCleanLoading] = useState(false)

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
  } = useListPage<LoginLogRow>({
    fetchApi: getLoginLogList,
    initialSearchForm: { ...loginLogInitialSearchForm },
    defaultSort: 'id:desc',
    transformData: (row) => transformLoginLogRow(row as unknown as Record<string, unknown>),
    onSearch: () => setSelectedRowKeys([]),
    onReset: () => setSelectedRowKeys([]),
  })

  const { toolbar, confirmDelete } = useCrudActions({
    onRefresh: refresh,
    deleteApi: deleteLoginLog,
  })

  const handleView = async (row: LoginLogRow) => {
    setDetailOpen(true)
    setDetailLoading(true)
    setLogDetail(null)
    try {
      const res = await getLoginLogDetail(row.id)
      const raw = extractLogDetail<Record<string, unknown>>(res.data as Record<string, unknown>, [
        'login_log',
        'log',
      ])
      if (raw) {
        setLogDetail(transformLoginLogRow(raw))
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
          await batchDeleteLoginLogs(selectedRowKeys)
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
      await cleanLoginLogs({ days })
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

  const searchFields = useMemo(() => createLoginLogSearchFields(t), [t])

  const columns: ColumnsType<LoginLogRow> = [
    { title: t('table.id'), dataIndex: 'id', width: 80, sorter: true },
    {
      title: t('log.admin'),
      dataIndex: ['admin', 'username'],
      width: 120,
      render: (_, row) => row.admin?.username || '-',
    },
    { title: t('log.ip'), dataIndex: 'ip', width: 140 },
    { title: t('log.location'), dataIndex: 'location', width: 160, ellipsis: true },
    { title: t('log.user_agent'), dataIndex: 'user_agent', ellipsis: true },
    {
      title: t('common.status'),
      dataIndex: 'status',
      width: 100,
      render: (status: number) => (
        <Tag color={(status ?? 1) === 1 ? 'success' : 'error'}>
          {(status ?? 1) === 1 ? t('log.success') : t('log.failed')}
        </Tag>
      ),
    },
    {
      title: t('log.message'),
      dataIndex: 'message',
      width: 140,
      render: (messageKey: string) => translateLoginMessage(t, messageKey),
    },
    { title: t('log.login_time'), dataIndex: 'created_at', width: 180, sorter: true },
    {
      title: t('common.operation'),
      key: 'operation',
      width: 160,
      fixed: 'end',
      onCell: () => ({ style: { whiteSpace: 'nowrap' } }),
      render: (_, row) => (
        <Space size={4} wrap={false}>
          {getButtonState('login_log.show').show ? (
            <PermissionButton
              permission="login_log.show"
              type="link"
              icon={<EyeOutlined />}
              onClick={() => void handleView(row)}
            >
              {t('common.view')}
            </PermissionButton>
          ) : null}
          {getButtonState('login_log.destroy').show ? (
            <PermissionButton
              permission="login_log.destroy"
              type="link"
              danger
              onClick={() => confirmDelete(row.id, row.admin?.username || row.name)}
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
  } = useColumnSetting('login_log', columns)

  return (
    <PageContainer
      title={t('menu.login_log')}
      extra={
        <Space>
          {selectedRowKeys.length > 0 && getButtonState('login_log.batch_delete').show ? (
            <PermissionButton permission="login_log.batch_delete" danger onClick={handleBatchDelete}>
              {t('common.delete_selected')} ({selectedRowKeys.length})
            </PermissionButton>
          ) : null}
          {getButtonState('login_log.clean').show ? (
            <PermissionButton permission="login_log.clean" onClick={() => setCleanOpen(true)}>
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
      <Table<LoginLogRow>
        rowKey="id"
        loading={loading}
        columns={filteredColumns}
        dataSource={tableData}
        scroll={{ x: 1300 }}
        rowSelection={
          getButtonState('login_log.batch_delete').show
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
              <Descriptions.Item label={t('log.admin')}>{logDetail.admin?.username || '-'}</Descriptions.Item>
              <Descriptions.Item label={t('log.ip')}>{logDetail.ip}</Descriptions.Item>
              <Descriptions.Item label={t('log.location')}>{logDetail.location || '-'}</Descriptions.Item>
              <Descriptions.Item label={t('common.status')}>
                <Tag color={logDetail.status === 1 ? 'success' : 'error'}>
                  {logDetail.status === 1 ? t('log.success') : t('log.failed')}
                </Tag>
              </Descriptions.Item>
              <Descriptions.Item label={t('log.user_agent')}>{logDetail.user_agent || '-'}</Descriptions.Item>
              <Descriptions.Item label={t('log.login_time')}>{logDetail.created_at}</Descriptions.Item>
              <Descriptions.Item label={t('log.message')} span={2}>
                {translateLoginMessage(t, logDetail.message)}
              </Descriptions.Item>
              <Descriptions.Item label={t('log.request')} span={2}>
                {logDetail.request ? (
                  <pre style={{ margin: 0, maxHeight: 300, overflow: 'auto', fontSize: 12 }}>
                    {formatRequestPreview(logDetail.request)}
                  </pre>
                ) : (
                  '-'
                )}
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
