import { useMemo, useState } from 'react'
import { App, Button, Space, Table, Tag } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useTranslation } from 'react-i18next'
import { batchDeleteExports, deleteExport, getExportList } from '@/api/export'
import { useListPage } from '@/hooks/useListPage'
import { handlePaginatedTableChange } from '@/utils/tableChange'
import { useCrudActions } from '@/hooks/useCrudActions'
import { usePermission } from '@/hooks/usePermission'
import { useUnhandledError } from '@/hooks/useUnhandledError'
import PageContainer from '@/components/PageContainer'
import SearchForm from '@/components/SearchForm'
import PermissionButton from '@/components/PermissionButton'
import { entityField } from '@/utils/normalize'
import { getApiBaseURL } from '@/utils/env'
import { buildAdminAuthHeaders } from '@/utils/authHeaders'

interface ExportRow {
  id: number | string
  type?: string
  filename?: string
  disk?: string
  path?: string
  extension?: string
  size?: number
  status?: number
  error_msg?: string
  created_at?: string
  file_url?: string
  admin?: Record<string, unknown> | null
  name?: string
}

function transformExportRow(row: Record<string, unknown>): ExportRow {
  const admin = (row.admin || row.Admin) as Record<string, unknown> | null
  return {
    id: entityField(row, 'id', '')!,
    type: String(entityField(row, 'type', '') ?? ''),
    filename: String(entityField(row, 'filename', '') ?? ''),
    disk: String(entityField(row, 'disk', '') ?? ''),
    path: String(entityField(row, 'path', '') ?? ''),
    extension: String(entityField(row, 'extension', '') ?? ''),
    size: Number(entityField(row, 'size', 0) ?? 0),
    status: Number(entityField(row, 'status', 0) ?? 0),
    error_msg: String(entityField(row, 'error_msg', '') ?? ''),
    created_at: String(entityField(row, 'created_at', '') ?? ''),
    file_url: String(entityField(row, 'file_url', '') ?? ''),
    admin,
    name: String(entityField(row, 'filename', '') ?? ''),
  }
}

function formatExportSize(size?: number) {
  const value = Number(size || 0)
  if (!value) return '-'
  if (value < 1024) return `${value} B`
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(2)} KB`
  if (value < 1024 * 1024 * 1024) return `${(value / 1024 / 1024).toFixed(2)} MB`
  return `${(value / 1024 / 1024 / 1024).toFixed(2)} GB`
}

export default function ExportList({ embedded = false }: { embedded?: boolean } = {}) {
  const { t } = useTranslation()
  const { message, modal } = App.useApp()
  const showError = useUnhandledError()
  const { getButtonState } = usePermission()
  const [downloadingIds, setDownloadingIds] = useState<Set<string | number>>(new Set())
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
  } = useListPage<ExportRow>({
    fetchApi: getExportList,
    initialSearchForm: {
      type: '',
      filename: '',
      disk: '',
      status: '',
      start_time: '',
      end_time: '',
    },
    normalizeRows: false,
    transformData: (row) => transformExportRow(row as unknown as Record<string, unknown>),
  })

  const { toolbar, confirmDelete } = useCrudActions({
    onRefresh: refresh,
    deleteApi: deleteExport,
  })

  const formatExportType = (type?: string) => {
    if (!type) return '-'
    const key = `export.types.${type}`
    const translated = t(key, { defaultValue: '__missing__' })
    return translated !== '__missing__' && translated !== key ? translated : type
  }

  const formatExportStatus = (status?: number) => {
    if (status === 1) return t('log.success')
    if (status === 0) return t('log.processing')
    return t('log.failed')
  }

  const getExportStatusColor = (status?: number) => {
    if (status === 1) return 'success'
    if (status === 0) return 'processing'
    return 'error'
  }

  const isExportCompleted = (row: ExportRow) => Number(row.status ?? 0) === 1

  const handleDownload = async (row: ExportRow) => {
    const exportId = row.id
    if (downloadingIds.has(exportId)) return

    const url = row.file_url
    if (!url) {
      message.error(t('export.download_failed'))
      return
    }

    setDownloadingIds((prev) => new Set(prev).add(exportId))

    try {
      let fullUrl = url
      if (url.startsWith('/')) {
        const apiBase = getApiBaseURL()
        const cleanUrl = url.replace(/^\/api\/admin/, '')
        fullUrl = `${apiBase.replace(/\/+$/, '')}${cleanUrl.startsWith('/') ? cleanUrl : `/${cleanUrl}`}`
      }

      const response = await fetch(fullUrl, {
        method: 'GET',
        headers: buildAdminAuthHeaders(),
      })

      if (!response.ok) {
        if (response.status === 401) {
          message.error(t('error.unauthorized'))
        } else {
          throw new Error(`HTTP error! status: ${response.status}`)
        }
        return
      }

      const contentType = response.headers.get('content-type') || ''
      if (contentType.includes('text/html')) {
        message.error(t('export.download_failed'))
        return
      }

      const blob = await response.blob()
      const contentDisposition = response.headers.get('content-disposition') || ''
      let filename = row.filename || 'export.csv'
      const filenameMatch = contentDisposition.match(/filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/)
      if (filenameMatch?.[1]) {
        filename = filenameMatch[1].replace(/['"]/g, '')
        try {
          filename = decodeURIComponent(filename)
        } catch {
          // keep original
        }
      }

      const downloadUrl = window.URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = downloadUrl
      link.download = filename
      link.style.display = 'none'
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(downloadUrl)
      message.success(t('export.download_success'))
    } catch (error) {
      showError(error, t('export.download_failed'))
    } finally {
      setTimeout(() => {
        setDownloadingIds((prev) => {
          const next = new Set(prev)
          next.delete(exportId)
          return next
        })
      }, 2000)
    }
  }

  const handleBatchDelete = () => {
    if (!selectedRowKeys.length) return
    modal.confirm({
      title: t('common.delete_confirm'),
      content: t('log.batch_delete_confirm', { count: selectedRowKeys.length, defaultValue: `确定删除选中的 ${selectedRowKeys.length} 条记录吗？` }),
      okType: 'danger',
      onOk: async () => {
        try {
          await batchDeleteExports(selectedRowKeys)
          message.success(t('common.delete_success'))
          setSelectedRowKeys([])
          await refresh()
        } catch (error) {
          showError(error, t('common.operation_failed'))
        }
      },
    })
  }

  const columns = useMemo<ColumnsType<ExportRow>>(
    () => [
      { title: t('table.id'), dataIndex: 'id', width: 80, sorter: true },
      { title: t('export.type'), dataIndex: 'type', width: 120, render: (v: string) => formatExportType(v) },
      { title: t('export.filename'), dataIndex: 'filename', ellipsis: true },
      { title: t('export.disk'), dataIndex: 'disk', width: 120 },
      { title: t('export.path'), dataIndex: 'path', ellipsis: true },
      { title: t('export.extension'), dataIndex: 'extension', width: 100 },
      { title: t('export.size'), dataIndex: 'size', width: 120, render: (v: number) => formatExportSize(v) },
      {
        title: t('log.status'),
        dataIndex: 'status',
        width: 150,
        render: (status: number, row) => (
          <div>
            <Tag color={getExportStatusColor(status)}>{formatExportStatus(status)}</Tag>
            {status === 2 && row.error_msg ? (
              <div style={{ marginTop: 4, color: '#ff4d4f', fontSize: 12, wordBreak: 'break-all' }}>{row.error_msg}</div>
            ) : null}
          </div>
        ),
      },
      {
        title: t('log.admin'),
        dataIndex: 'admin',
        width: 140,
        render: (admin: Record<string, unknown> | null) =>
          String(entityField(admin || {}, 'username', '-') ?? '-'),
      },
      { title: t('table.created_at'), dataIndex: 'created_at', width: 180, sorter: true },
      {
        title: t('common.operation'),
        key: 'operation',
        width: 160,
        fixed: 'end',
        render: (_, row) => (
          <Space>
            <Button
              type="link"
              disabled={downloadingIds.has(row.id) || !isExportCompleted(row)}
              loading={downloadingIds.has(row.id)}
              onClick={() => void handleDownload(row)}
            >
              {t('common.view')}
            </Button>
            {getButtonState('export.destroy').show ? (
              <PermissionButton
                permission="export.destroy"
                type="link"
                danger
                onClick={() => confirmDelete(row.id, row.filename)}
              >
                {t('common.delete')}
              </PermissionButton>
            ) : null}
          </Space>
        ),
      },
    ],
    [confirmDelete, downloadingIds, getButtonState, t],
  )

  const content = (
    <>
      <SearchForm
        fields={[
          {
            name: 'type',
            label: t('export.type'),
            type: 'select',
            options: [
              { label: t('export.types.orders'), value: 'orders' },
              { label: t('export.types.payments'), value: 'payments' },
              { label: t('export.types.admins'), value: 'admins' },
              { label: t('export.types.users'), value: 'users' },
              { label: t('export.types.articles'), value: 'articles' },
              { label: t('export.types.operation_logs_archive'), value: 'operation_logs_archive' },
            ],
          },
          { name: 'filename', label: t('export.filename') },
          {
            name: 'disk',
            label: t('export.disk'),
            type: 'select',
            options: [
              { label: 'local', value: 'local' },
              { label: 'public', value: 'public' },
              { label: 's3', value: 's3' },
              { label: 'oss', value: 'oss' },
              { label: 'cos', value: 'cos' },
              { label: 'qiniu', value: 'qiniu' },
              { label: 'minio', value: 'minio' },
            ],
          },
          {
            name: 'status',
            label: t('log.status'),
            type: 'select',
            options: [
              { label: t('log.processing'), value: '0' },
              { label: t('log.success'), value: '1' },
              { label: t('log.failed'), value: '2' },
            ],
          },
          { name: 'start_time', label: t('log.start_time') },
          { name: 'end_time', label: t('log.end_time') },
        ]}
        values={searchForm}
        onChange={onSearchFormChange}
        onSearch={handleSearch}
        onReset={handleReset}
      />
      <Table<ExportRow>
        rowKey="id"
        loading={loading}
        columns={columns}
        dataSource={tableData}
        scroll={{ x: 1400 }}
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
    </>
  )

  if (embedded) {
    return (
      <div>
        {selectedRowKeys.length > 0 && getButtonState('export.destroy').show ? (
          <div style={{ marginBottom: 16 }}>
            <Button danger onClick={handleBatchDelete}>
              {t('common.delete_selected', { defaultValue: '删除选中' })} ({selectedRowKeys.length})
            </Button>
          </div>
        ) : null}
        {content}
      </div>
    )
  }

  return (
    <PageContainer
      title={t('menu.export')}
      extra={
        <Space>
          {selectedRowKeys.length > 0 && getButtonState('export.destroy').show ? (
            <Button danger onClick={handleBatchDelete}>
              {t('common.delete_selected', { defaultValue: '删除选中' })} ({selectedRowKeys.length})
            </Button>
          ) : null}
          {toolbar}
        </Space>
      }
    >
      {content}
    </PageContainer>
  )
}
