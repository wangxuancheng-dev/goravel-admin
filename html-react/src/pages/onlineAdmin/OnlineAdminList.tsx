import { useState } from 'react'
import { Avatar, Button, Space, Table } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useTranslation } from 'react-i18next'
import { SettingOutlined } from '@ant-design/icons'
import { App } from 'antd'
import {
  batchKickOutOnlineAdmins,
  getOnlineAdminList,
  kickOutOnlineAdmin,
} from '@/api/onlineAdmin'
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
import { entityField } from '@/utils/normalize'
import { promptSensitiveConfirm } from '@/utils/sensitiveConfirm'

interface OnlineAdminRow {
  id: number | string
  username?: string
  nickname?: string
  avatar?: string
  browser?: string
  ip?: string
  os?: string
  session_id?: string
  last_active?: string
  name?: string
}

function formatOnlineTime(time?: string) {
  if (!time) return '-'
  const date = new Date(time)
  if (Number.isNaN(date.getTime())) return time
  return date.toLocaleString().replace(/\//g, '-')
}

export default function OnlineAdminList() {
  const { t } = useTranslation()
  const { modal, message } = App.useApp()
  const showError = useUnhandledError()
  const { getButtonState } = usePermission()
  const [selectedRowKeys, setSelectedRowKeys] = useState<Array<string | number>>([])

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
  } = useListPage<OnlineAdminRow>({
    fetchApi: getOnlineAdminList,
    initialSearchForm: { username: '', ip: '', browser: '', os: '' },
    fieldMapping: { last_active: 'last_used_at' },
    defaultSort: 'last_used_at:desc',
    normalizeRows: false,
    transformData: (row) => {
      const record = row
      return {
        id: entityField(record, 'id', '')!,
        username: String(entityField(record, 'username', '') ?? ''),
        nickname: String(entityField(record, 'nickname', '') ?? ''),
        avatar: String(entityField(record, 'avatar', '') ?? ''),
        browser: String(entityField(record, 'browser', '') ?? ''),
        ip: String(entityField(record, 'ip', '') ?? ''),
        os: String(entityField(record, 'os', '') ?? ''),
        session_id: String(entityField(record, 'session_id', '') ?? ''),
        last_active: String(
          entityField(record, 'last_active', '') ?? entityField(record, 'last_used_at', '') ?? '',
        ),
        name: String(entityField(record, 'username', '') ?? ''),
      }
    },
  })

  const { toolbar } = useCrudActions({ onRefresh: refresh })

  const handleKickOut = (row: OnlineAdminRow) => {
    void (async () => {
      try {
        const confirmCode = await promptSensitiveConfirm(modal, t, {
          title: t('online_admin.kick_out_confirm', { username: row.username }),
        })
        await kickOutOnlineAdmin(row.id, { confirm_code: confirmCode })
        message.success(t('online_admin.kick_out_success'))
        await refresh()
      } catch (error) {
        if ((error as Error)?.message === 'cancel') return
        showError(error, t('online_admin.kick_out_failed'))
      }
    })()
  }

  const handleBatchKickOut = () => {
    if (!selectedRowKeys.length) return
    void (async () => {
      try {
        const confirmCode = await promptSensitiveConfirm(modal, t, {
          title: t('online_admin.batch_kick_out_confirm', { count: selectedRowKeys.length }),
        })
        await batchKickOutOnlineAdmins(selectedRowKeys, { confirm_code: confirmCode })
        message.success(t('online_admin.batch_kick_out_success'))
        setSelectedRowKeys([])
        await refresh()
      } catch (error) {
        if ((error as Error)?.message === 'cancel') return
        showError(error, t('online_admin.batch_kick_out_failed'))
      }
    })()
  }

  const columns: ColumnsType<OnlineAdminRow> = [
    { title: t('online_admin.username'), dataIndex: 'username', width: 120 },
    { title: t('online_admin.nickname'), dataIndex: 'nickname', width: 120 },
    {
      title: t('online_admin.avatar'),
      dataIndex: 'avatar',
      width: 80,
      render: (_, row) => (
        <Avatar src={row.avatar || undefined} size={32}>
          {row.nickname?.charAt(0) || row.username?.charAt(0) || 'U'}
        </Avatar>
      ),
    },
    { title: t('online_admin.browser'), dataIndex: 'browser', width: 150 },
    { title: t('online_admin.ip'), dataIndex: 'ip', width: 150 },
    { title: t('online_admin.os'), dataIndex: 'os', width: 150 },
    { title: t('online_admin.session_id'), dataIndex: 'session_id', width: 200, ellipsis: true },
    {
      title: t('online_admin.last_active'),
      dataIndex: 'last_active',
      width: 180,
      sorter: true,
      render: (time: string) => formatOnlineTime(time),
    },
    {
      title: t('common.operation'),
      key: 'operation',
      width: 120,
      fixed: 'end',
      render: (_, row) =>
        getButtonState('admin.kick_out').show ? (
          <PermissionButton permission="admin.kick_out" type="link" danger onClick={() => handleKickOut(row)}>
            {t('online_admin.kick_out')}
          </PermissionButton>
        ) : null,
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
  } = useColumnSetting('online_admin', columns)

  return (
    <PageContainer
      title={t('menu.online_admin')}
      extra={
        <Space>
          {selectedRowKeys.length > 0 && getButtonState('admin.kick_out').show ? (
            <Button danger onClick={handleBatchKickOut}>
              {t('online_admin.batch_kick_out')} ({selectedRowKeys.length})
            </Button>
          ) : null}
          {toolbar}
          <Button icon={<SettingOutlined />} onClick={openColumnSetting}>
            {t('common.column_setting')}
          </Button>
        </Space>
      }
    >
      <SearchForm
        fields={[
          { name: 'username', label: t('online_admin.username') },
          { name: 'ip', label: t('online_admin.ip') },
          { name: 'browser', label: t('online_admin.browser') },
          { name: 'os', label: t('online_admin.os') },
        ]}
        values={searchForm}
        onChange={onSearchFormChange}
        onSearch={handleSearch}
        onReset={handleReset}
      />
      <Table<OnlineAdminRow>
        rowKey="id"
        loading={loading}
        columns={filteredColumns}
        dataSource={tableData}
        scroll={{ x: 1200 }}
        rowSelection={{
          selectedRowKeys,
          onChange: (keys) => setSelectedRowKeys(keys as Array<string | number>),
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
