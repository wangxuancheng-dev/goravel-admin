import { useState } from 'react'
import { Button, Modal, Space, Table, Tag } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { PlusOutlined } from '@ant-design/icons'
import dayjs from 'dayjs'
import { useTranslation } from 'react-i18next'
import { getNotificationList } from '@/api/notification'
import { useListPage } from '@/hooks/useListPage'
import { handlePaginatedTableChange } from '@/utils/tableChange'
import { useCrudActions } from '@/hooks/useCrudActions'
import { useNotificationStore } from '@/stores/notification'
import { useUserStore } from '@/stores/user'
import PageContainer from '@/components/PageContainer'
import SearchForm from '@/components/SearchForm'
import PermissionButton from '@/components/PermissionButton'
import MarkdownContent from '@/components/MarkdownContent'
import NotificationFormModal from './NotificationFormModal'
import {
  buildNotificationParams,
  getNotificationTypeLabel,
  notificationInitialSearchForm,
  transformNotificationRow,
  type NotificationRow,
} from './notification.config'
import { extractTextFromMarkdown } from '@/utils/markdown'

export default function NotificationList() {
  const { t } = useTranslation()
  const adminInfo = useUserStore((s) => s.adminInfo)
  const unreadCount = useNotificationStore((s) => s.unreadCount)
  const markAsRead = useNotificationStore((s) => s.markAsRead)
  const markAllRead = useNotificationStore((s) => s.markAllRead)
  const refreshBell = useNotificationStore((s) => s.refresh)
  const [formOpen, setFormOpen] = useState(false)
  const [viewOpen, setViewOpen] = useState(false)
  const [current, setCurrent] = useState<NotificationRow | null>(null)

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
  } = useListPage<NotificationRow, typeof notificationInitialSearchForm>({
    fetchApi: getNotificationList,
    initialSearchForm: notificationInitialSearchForm,
    defaultSort: 'id:desc',
    transformData: (row) => transformNotificationRow(row as unknown as Record<string, unknown>),
    buildParams: buildNotificationParams,
    onLoadSuccess: (_rows, res) => {
      const unread = (res?.data as { unread_count?: number } | undefined)?.unread_count
      if (unread !== undefined) {
        useNotificationStore.setState({ unreadCount: unread })
      }
    },
  })

  const { toolbar } = useCrudActions({
    onRefresh: async () => {
      await refresh()
      await refreshBell()
    },
  })

  const senderLabel = (row: NotificationRow) => {
    if (row.type === 'message' && String(row.sender_id) === String(adminInfo?.id)) {
      const receiver = row.receiver
      return (
        <span>
          {t('notification.sent_to')}:{' '}
          {receiver ? receiver.nickname || receiver.username : '-'}
        </span>
      )
    }
    if (row.sender) return row.sender.nickname || row.sender.username
    return t('notification.system')
  }

  const handleView = async (row: NotificationRow) => {
    setCurrent(row)
    setViewOpen(true)
    if (!row.is_read) {
      await markAsRead(row.id)
      await refresh()
    }
  }

  const columns: ColumnsType<NotificationRow> = [
    { title: t('table.id'), dataIndex: 'id', width: 80 },
    { title: t('notification.table.title'), dataIndex: 'title', width: 180, ellipsis: true },
    {
      title: t('notification.table.content'),
      dataIndex: 'content',
      ellipsis: true,
      render: (content: string) => extractTextFromMarkdown(content).slice(0, 120),
    },
    {
      title: t('notification.table.type'),
      dataIndex: 'type',
      width: 100,
      render: (type: string) => <Tag>{getNotificationTypeLabel(t, type)}</Tag>,
    },
    {
      title: t('notification.table.sender'),
      key: 'sender',
      width: 160,
      ellipsis: true,
      render: (_, row) => senderLabel(row),
    },
    {
      title: t('notification.table.status'),
      dataIndex: 'is_read',
      width: 90,
      render: (read: boolean) =>
        read ? (
          <Tag>{t('notification.read')}</Tag>
        ) : (
          <Tag color="processing">{t('notification.unread')}</Tag>
        ),
    },
    {
      title: t('notification.table.created_at'),
      dataIndex: 'created_at',
      width: 170,
      sorter: true,
      render: (v: string) => (v ? dayjs(v).format('YYYY-MM-DD HH:mm:ss') : '-'),
    },
    {
      title: t('common.operation'),
      key: 'operation',
      width: 160,
      fixed: 'end',
      render: (_, row) => (
        <Space>
          <Button type="link" onClick={() => void handleView(row)}>
            {t('common.view')}
          </Button>
          {!row.is_read && (
            <Button
              type="link"
              onClick={async () => {
                await markAsRead(row.id)
                await refresh()
              }}
            >
              {t('notification.mark_read')}
            </Button>
          )}
        </Space>
      ),
    },
  ]

  return (
    <PageContainer
      title={t('notification.center')}
      extra={
        <Space>
          {toolbar}
          <PermissionButton
            permission="notification.store"
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => setFormOpen(true)}
          >
            {t('notification.create')}
          </PermissionButton>
          <Button disabled={unreadCount === 0} onClick={() => void markAllRead().then(() => refresh())}>
            {t('notification.mark_all')}
          </Button>
        </Space>
      }
    >
      <SearchForm
        fields={[
          {
            name: 'type',
            label: t('notification.table.type'),
            type: 'select',
            options: [
              { label: t('common.all'), value: '' },
              { label: t('notification.types.announcement'), value: 'announcement' },
              { label: t('notification.types.notice'), value: 'notice' },
              { label: t('notification.types.message'), value: 'message' },
            ],
          },
          {
            name: 'is_read',
            label: t('notification.table.status'),
            type: 'select',
            options: [
              { label: t('common.all'), value: '' },
              { label: t('notification.unread'), value: 'false' },
              { label: t('notification.read'), value: 'true' },
            ],
          },
        ]}
        values={searchForm}
        onChange={onSearchFormChange}
        onSearch={handleSearch}
        onReset={handleReset}
      />
      <Table<NotificationRow>
        rowKey="id"
        loading={loading}
        columns={columns}
        dataSource={tableData}
        scroll={{ x: 1100 }}
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

      <NotificationFormModal
        open={formOpen}
        onClose={() => setFormOpen(false)}
        onSuccess={() => void refresh()}
      />

      <Modal
        open={viewOpen}
        title={t('notification.detail')}
        width={800}
        onCancel={() => setViewOpen(false)}
        footer={
          <Button onClick={() => setViewOpen(false)}>{t('common.close')}</Button>
        }
      >
        {current && (
          <>
            <h3 style={{ marginTop: 0 }}>{current.title}</h3>
            <Space wrap style={{ marginBottom: 16, color: 'var(--ant-color-text-secondary)' }}>
              <Tag>{getNotificationTypeLabel(t, current.type)}</Tag>
              <span>
                {current.created_at ? dayjs(current.created_at).format('YYYY-MM-DD HH:mm:ss') : ''}
              </span>
              <span>
                {t('notification.table.sender')}: {senderLabel(current)}
              </span>
            </Space>
              <MarkdownContent content={current.content} />
          </>
        )}
      </Modal>
    </PageContainer>
  )
}
