import { useMemo, useState } from 'react'
import { App, Button, Space, Table, Tag } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useTranslation } from 'react-i18next'
import { deleteImport, getImportList } from '@/api/import'
import { useListPage } from '@/hooks/useListPage'
import { handlePaginatedTableChange } from '@/utils/tableChange'
import { useCrudActions } from '@/hooks/useCrudActions'
import { usePermission } from '@/hooks/usePermission'
import { useUnhandledError } from '@/hooks/useUnhandledError'
import SearchForm from '@/components/SearchForm'
import PermissionButton from '@/components/PermissionButton'
import { entityField } from '@/utils/normalize'
import { getApiBaseURL } from '@/utils/env'
import { buildAdminAuthHeaders } from '@/utils/authHeaders'

interface ImportRow {
  id: number | string
  type?: string
  status?: number
  total_rows?: number
  success_rows?: number
  failed_rows?: number
  error_msg?: string
  error_file_url?: string
  created_at?: string
  admin?: Record<string, unknown> | null
}

function transformImportRow(row: Record<string, unknown>): ImportRow {
  const admin = (row.admin || row.Admin) as Record<string, unknown> | null
  return {
    id: entityField(row, 'id', '')!,
    type: String(entityField(row, 'type', '') ?? ''),
    status: Number(entityField(row, 'status', 0) ?? 0),
    total_rows: Number(entityField(row, 'total_rows', 0) ?? 0),
    success_rows: Number(entityField(row, 'success_rows', 0) ?? 0),
    failed_rows: Number(entityField(row, 'failed_rows', 0) ?? 0),
    error_msg: String(entityField(row, 'error_msg', '') ?? ''),
    error_file_url: String(entityField(row, 'error_file_url', '') ?? ''),
    created_at: String(entityField(row, 'created_at', '') ?? ''),
    admin,
  }
}

export default function ImportList() {
  const { t } = useTranslation()
  const { message } = App.useApp()
  const showError = useUnhandledError()
  const { getButtonState } = usePermission()
  const [downloadingIds, setDownloadingIds] = useState<Set<string | number>>(new Set())

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
  } = useListPage<ImportRow>({
    fetchApi: getImportList,
    initialSearchForm: {
      type: '',
      status: '',
      start_time: '',
      end_time: '',
    },
    normalizeRows: false,
    transformData: (row) => transformImportRow(row as unknown as Record<string, unknown>),
  })

  const { confirmDelete } = useCrudActions({
    onRefresh: refresh,
    deleteApi: deleteImport,
  })

  const formatImportType = (type?: string) => {
    if (!type) return '-'
    const key = `export.types.${type}`
    const translated = t(key, { defaultValue: '__missing__' })
    return translated !== '__missing__' && translated !== key ? translated : type
  }

  const formatStatus = (status?: number) => {
    if (status === 1) return t('log.success')
    if (status === 0) return t('log.processing')
    return t('log.failed')
  }

  const getStatusColor = (status?: number) => {
    if (status === 1) return 'success'
    if (status === 0) return 'processing'
    return 'error'
  }

  const handleDownloadError = async (row: ImportRow) => {
    const importId = row.id
    if (downloadingIds.has(importId) || !row.error_file_url) return

    setDownloadingIds((prev) => new Set(prev).add(importId))
    try {
      let fullUrl = row.error_file_url
      if (fullUrl.startsWith('/')) {
        const apiBase = getApiBaseURL()
        const cleanUrl = fullUrl.replace(/^\/api\/admin/, '')
        fullUrl = `${apiBase.replace(/\/+$/, '')}${cleanUrl.startsWith('/') ? cleanUrl : `/${cleanUrl}`}`
      }

      const response = await fetch(fullUrl, {
        method: 'GET',
        headers: buildAdminAuthHeaders(),
      })
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }

      const blob = await response.blob()
      const downloadUrl = window.URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = downloadUrl
      link.download = `import_errors_${importId}.csv`
      link.style.display = 'none'
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(downloadUrl)
      message.success(t('export.download_success'))
    } catch (error) {
      showError(error, t('export.download_failed'))
    } finally {
      setDownloadingIds((prev) => {
        const next = new Set(prev)
        next.delete(importId)
        return next
      })
    }
  }

  const columns = useMemo<ColumnsType<ImportRow>>(
    () => [
      { title: t('table.id'), dataIndex: 'id', width: 80, sorter: true },
      { title: t('export.type'), dataIndex: 'type', width: 120, render: (v: string) => formatImportType(v) },
      {
        title: t('log.status'),
        dataIndex: 'status',
        width: 140,
        render: (status: number, row) => (
          <div>
            <Tag color={getStatusColor(status)}>{formatStatus(status)}</Tag>
            {status === 2 && row.error_msg ? (
              <div style={{ marginTop: 4, color: '#ff4d4f', fontSize: 12, wordBreak: 'break-all' }}>{row.error_msg}</div>
            ) : null}
          </div>
        ),
      },
      { title: t('import.total_rows'), dataIndex: 'total_rows', width: 100 },
      { title: t('import.success_rows'), dataIndex: 'success_rows', width: 100 },
      { title: t('import.failed_rows'), dataIndex: 'failed_rows', width: 100 },
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
        width: 200,
        fixed: 'end',
        render: (_, row) => (
          <Space>
            {row.error_file_url ? (
              <Button
                type="link"
                loading={downloadingIds.has(row.id)}
                onClick={() => void handleDownloadError(row)}
              >
                {t('import.download_error')}
              </Button>
            ) : null}
            {getButtonState('import.destroy').show ? (
              <PermissionButton
                permission="import.destroy"
                type="link"
                danger
                onClick={() => confirmDelete(row.id)}
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

  return (
    <div>
      <SearchForm
        fields={[
          {
            name: 'type',
            label: t('export.type'),
            type: 'select',
            options: [
              { label: t('export.types.orders'), value: 'orders' },
              { label: t('export.types.articles'), value: 'articles' },
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
      <Table<ImportRow>
        rowKey="id"
        loading={loading}
        columns={columns}
        dataSource={tableData}
        scroll={{ x: 1200 }}
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
    </div>
  )
}
