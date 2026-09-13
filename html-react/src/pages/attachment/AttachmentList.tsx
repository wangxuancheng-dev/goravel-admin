import { useEffect, useRef, useState } from 'react'
import {
  App,
  Button,
  Image,
  Input,
  Modal,
  Progress,
  Select,
  Space,
  Switch,
  Table,
  Tag,
  Upload,
} from 'antd'
import type { ColumnsType } from 'antd/es/table'
import {
  DeleteOutlined,
  PictureOutlined,
  ScissorOutlined,
  SettingOutlined,
  UploadOutlined,
} from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import {
  batchDeleteAttachments,
  deleteAttachment,
  getAttachmentDownloadUrl,
  getAttachmentPreviewUrl,
  getAttachmentList,
  updateCategory,
  updateDisplayName,
  updateVisibility,
  uploadAttachment,
} from '@/api/attachment'
import { useListPage } from '@/hooks/useListPage'
import { handlePaginatedTableChange } from '@/utils/tableChange'
import { useCrudActions } from '@/hooks/useCrudActions'
import { usePermission } from '@/hooks/usePermission'
import { useUnhandledError } from '@/hooks/useUnhandledError'
import { useOptions } from '@/hooks/useOptions'
import { useAttachmentChunkUpload } from '@/hooks/useAttachmentChunkUpload'
import { useAttachmentImagePreview } from '@/hooks/useAttachmentImagePreview'
import { useColumnSetting } from '@/hooks/useColumnSetting'
import PageContainer from '@/components/PageContainer'
import ColumnSettingDialog from '@/components/ColumnSettingDialog'
import SearchForm from '@/components/SearchForm'
import PermissionButton from '@/components/PermissionButton'
import Storage from '@/utils/storage'
import i18n from '@/i18n'
import {
  attachmentPreviewKind,
  canInlinePreviewAttachment,
  type AttachmentPreviewKind,
} from '@/utils/attachmentUrl'
import AttachmentCategoryModal from './AttachmentCategoryModal'
import CropUploadModal from './CropUploadModal'
import {
  ATTACHMENT_LARGE_FILE_THRESHOLD,
  attachmentInitialSearchForm,
  formatAttachmentAdmin,
  formatAttachmentFileSize,
  getAttachmentFileTypeColor,
  getAttachmentFileTypeLabel,
  transformAttachmentRow,
  type AttachmentRow,
} from './attachment.config'

export default function AttachmentList() {
  const { t } = useTranslation()
  const { message, modal } = App.useApp()
  const showError = useUnhandledError()
  const { getButtonState } = usePermission()
  const [uploading, setUploading] = useState(false)
  const [downloadingIds, setDownloadingIds] = useState<Set<string | number>>(new Set())
  const [selectedRowKeys, setSelectedRowKeys] = useState<Array<string | number>>([])
  const [categoryModalOpen, setCategoryModalOpen] = useState(false)
  const [cropModalOpen, setCropModalOpen] = useState(false)
  const [previewOpen, setPreviewOpen] = useState(false)
  const [previewLoading, setPreviewLoading] = useState(false)
  const [previewRow, setPreviewRow] = useState<AttachmentRow | null>(null)
  const [previewKind, setPreviewKind] = useState<AttachmentPreviewKind>('')
  const [previewBlobUrl, setPreviewBlobUrl] = useState('')
  const [previewText, setPreviewText] = useState('')
  const previewBlobUrlRef = useRef('')
  const [, bump] = useState(0)
  const forceRender = () => bump((n) => n + 1)
  const { selectOptions: categoryOptions, reload: reloadCategories } = useOptions('attachment_category')
  const { loadImageAsBlob, getImageUrl, getImageLoadingState } = useAttachmentImagePreview()

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
  } = useListPage<AttachmentRow>({
    fetchApi: getAttachmentList,
    initialSearchForm: { ...attachmentInitialSearchForm },
    defaultSort: 'id:desc',
    normalizeRows: false,
    transformData: (row) => transformAttachmentRow(row as unknown as Record<string, unknown>),
    onLoadSuccess: (rows) => {
      rows.forEach((row) => {
        if (row.file_type === 'image') {
          void loadImageAsBlob(row)
        }
      })
    },
  })

  const chunk = useAttachmentChunkUpload(refresh)

  const { toolbar, confirmDelete } = useCrudActions({
    onRefresh: refresh,
    deleteApi: deleteAttachment,
  })

  useEffect(() => {
    void reloadCategories()
  }, [reloadCategories])

  useEffect(() => {
    return () => {
      if (previewBlobUrlRef.current) {
        window.URL.revokeObjectURL(previewBlobUrlRef.current)
        previewBlobUrlRef.current = ''
      }
    }
  }, [])

  const handleUploadFile = async (file: File) => {
    const maxSize = 100 * 1024 * 1024
    if (file.size > maxSize) {
      message.error(t('attachment.file_too_large'))
      return
    }
    if (file.size > ATTACHMENT_LARGE_FILE_THRESHOLD) {
      void chunk.startChunkUpload(file)
      return
    }
    setUploading(true)
    try {
      await uploadAttachment(file)
      message.success(t('attachment.upload_success'))
      await refresh()
    } catch (error) {
      showError(error, t('attachment.upload_failed'))
    } finally {
      setUploading(false)
    }
  }

  const handleUpdateDisplayName = async (row: AttachmentRow, value: string) => {
    if (!row.id) return
    const next = value || ''
    if (next === (row.display_name || '')) return
    try {
      await updateDisplayName(row.id, next)
      row.display_name = next
      forceRender()
      message.success(t('attachment.update_success'))
    } catch (error) {
      showError(error, t('attachment.update_failed'))
      await refresh()
    }
  }

  const handleUpdateCategory = async (row: AttachmentRow, categoryId: string | number) => {
    if (!row.id || !categoryId) return
    const prev = row.category_id
    row.category_id = categoryId
    forceRender()
    try {
      await updateCategory(row.id, categoryId)
      message.success(t('attachment.update_success'))
    } catch (error) {
      row.category_id = prev
      forceRender()
      showError(error, t('attachment.update_failed'))
      await refresh()
    }
  }

  const handleUpdateVisibility = async (row: AttachmentRow, checked: boolean) => {
    if (!row.id) return
    const prev = row.is_public
    row.is_public = checked ? 1 : 0
    forceRender()
    try {
      const res = await updateVisibility(row.id, checked)
      const data = res?.data as { file_url?: string } | undefined
      if (data?.file_url) {
        row.file_url = data.file_url
        if (row.file_type === 'image') {
          void loadImageAsBlob(row)
        }
      }
      message.success(t('attachment.update_success'))
    } catch (error) {
      row.is_public = prev
      forceRender()
      showError(error, t('attachment.update_failed'))
      await refresh()
    }
  }

  const revokePreviewBlob = () => {
    if (previewBlobUrlRef.current) {
      window.URL.revokeObjectURL(previewBlobUrlRef.current)
      previewBlobUrlRef.current = ''
    }
    setPreviewBlobUrl('')
    setPreviewText('')
    setPreviewKind('')
    setPreviewRow(null)
  }

  const closePreview = () => {
    setPreviewOpen(false)
  }

  const handlePreview = async (row: AttachmentRow) => {
    if (!canInlinePreviewAttachment(row)) {
      message.warning(t('attachment.preview_unsupported'))
      return
    }
    const kind = attachmentPreviewKind(row)
    if (!kind) {
      message.warning(t('attachment.preview_unsupported'))
      return
    }

    revokePreviewBlob()
    setPreviewRow(row)
    setPreviewKind(kind)
    setPreviewOpen(true)
    setPreviewLoading(true)

    try {
      const previewUrl = getAttachmentPreviewUrl(row.id)
      const token = String(Storage.getItem('token', '') ?? '').trim()
      const currentLocale = i18n.language || Storage.getItem('language', 'zh-CN') || 'zh-CN'
      const acceptLanguage = currentLocale === 'en-US' ? 'en-US' : 'zh-CN'

      const response = await fetch(previewUrl, {
        method: 'GET',
        headers: {
          Authorization: `Bearer ${token}`,
          'Accept-Language': acceptLanguage,
        },
      })

      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }

      const contentType = response.headers.get('content-type') || ''
      if (contentType.includes('text/html') && kind !== 'text') {
        throw new Error('invalid preview content')
      }

      const blob = await response.blob()
      if (kind === 'text') {
        setPreviewText(await blob.text())
      } else {
        const objectUrl = window.URL.createObjectURL(blob)
        previewBlobUrlRef.current = objectUrl
        setPreviewBlobUrl(objectUrl)
      }
    } catch (error) {
      setPreviewOpen(false)
      revokePreviewBlob()
      showError(error, t('attachment.preview_failed'))
    } finally {
      setPreviewLoading(false)
    }
  }

  const handleDownload = async (row: AttachmentRow) => {
    const attachmentId = row.id
    if (downloadingIds.has(attachmentId)) return

    setDownloadingIds((prev) => new Set(prev).add(attachmentId))

    try {
      const downloadUrl = getAttachmentDownloadUrl(attachmentId)
      const token = String(Storage.getItem('token', '') ?? '').trim()
      const currentLocale = i18n.language || Storage.getItem('language', 'zh-CN') || 'zh-CN'
      const acceptLanguage = currentLocale === 'en-US' ? 'en-US' : 'zh-CN'

      const response = await fetch(downloadUrl, {
        method: 'GET',
        headers: {
          Authorization: `Bearer ${token}`,
          'Accept-Language': acceptLanguage,
        },
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
        message.error(t('attachment.download_failed'))
        return
      }

      const blob = await response.blob()
      const contentDisposition = response.headers.get('content-disposition') || ''
      let filename = row.filename || 'attachment'
      const filenameMatch = contentDisposition.match(/filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/)
      if (filenameMatch?.[1]) {
        filename = filenameMatch[1].replace(/['"]/g, '')
        try {
          filename = decodeURIComponent(filename)
        } catch {
          // keep original
        }
      }

      const objectUrl = window.URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = objectUrl
      link.download = filename
      link.style.display = 'none'
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(objectUrl)
      message.success(t('attachment.download_success'))
    } catch (error) {
      showError(error, t('attachment.download_failed'))
    } finally {
      setTimeout(() => {
        setDownloadingIds((prev) => {
          const next = new Set(prev)
          next.delete(attachmentId)
          return next
        })
      }, 2000)
    }
  }

  const handleBatchDelete = () => {
    if (!selectedRowKeys.length) return
    modal.confirm({
      title: t('common.confirm'),
      content: t('attachment.batch_delete_confirm', { count: selectedRowKeys.length }),
      okType: 'danger',
      onOk: async () => {
        try {
          await batchDeleteAttachments(selectedRowKeys)
          message.success(t('common.delete_success'))
          setSelectedRowKeys([])
          await refresh()
        } catch (error) {
          showError(error, t('common.operation_failed'))
        }
      },
    })
  }

  const columns: ColumnsType<AttachmentRow> = [
    { title: t('table.id'), dataIndex: 'id', width: 80, sorter: true },
    {
      title: t('attachment.filename'),
      dataIndex: 'filename',
      width: 220,
      render: (_, row) => {
        const previewUrl = getImageUrl(row)
        const loadingState = getImageLoadingState(row)
        return (
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            {row.file_type === 'image' ? (
              previewUrl ? (
                <Image
                  src={previewUrl}
                  width={50}
                  height={50}
                  style={{ objectFit: 'cover', borderRadius: 4, border: '1px solid #f0f0f0' }}
                  preview={{ src: previewUrl }}
                />
              ) : loadingState === 'error' ? (
                <div
                  style={{
                    width: 50,
                    height: 50,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    background: '#fef0f0',
                    borderRadius: 4,
                    border: '1px solid #fde2e2',
                    color: '#f56c6c',
                  }}
                >
                  <PictureOutlined />
                </div>
              ) : (
                <div
                  style={{
                    width: 50,
                    height: 50,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    background: '#fafafa',
                    borderRadius: 4,
                    border: '1px solid #f0f0f0',
                    color: '#999',
                  }}
                >
                  ...
                </div>
              )
            ) : null}
            <span style={{ wordBreak: 'break-all' }}>{row.filename}</span>
          </div>
        )
      },
    },
    {
      title: t('attachment.display_name'),
      dataIndex: 'display_name',
      width: 180,
      render: (_, row) => (
        <Input
          key={`${row.id}-${row.display_name}`}
          size="small"
          defaultValue={row.display_name}
          placeholder={t('attachment.display_name_placeholder')}
          onBlur={(e) => void handleUpdateDisplayName(row, e.target.value)}
          onPressEnter={(e) => {
            ;(e.target as HTMLInputElement).blur()
          }}
        />
      ),
    },
    {
      title: t('attachment.category'),
      dataIndex: 'category_id',
      width: 160,
      render: (_, row) => (
        <Select
          size="small"
          style={{ width: 140 }}
          value={row.category_id ?? undefined}
          options={categoryOptions}
          disabled={getButtonState('attachment.update_category').disabled}
          onChange={(value) => void handleUpdateCategory(row, value)}
        />
      ),
    },
    {
      title: t('attachment.visibility'),
      dataIndex: 'is_public',
      width: 110,
      render: (_, row) => (
        <Switch
          size="small"
          checked={Number(row.is_public) === 1}
          checkedChildren={t('attachment.visibility_public_short')}
          unCheckedChildren={t('attachment.visibility_private_short')}
          disabled={getButtonState('attachment.update_visibility').disabled}
          onChange={(checked) => void handleUpdateVisibility(row, checked)}
        />
      ),
    },
    {
      title: t('attachment.file_type'),
      dataIndex: 'file_type',
      width: 100,
      render: (v: string) => (
        <Tag color={getAttachmentFileTypeColor(v)}>{getAttachmentFileTypeLabel(t, v)}</Tag>
      ),
    },
    {
      title: t('attachment.disk'),
      dataIndex: 'disk',
      width: 100,
      render: (v: string) => <Tag>{v || '-'}</Tag>,
    },
    { title: t('attachment.extension'), dataIndex: 'extension', width: 90 },
    {
      title: t('attachment.size'),
      dataIndex: 'size',
      width: 120,
      render: (v: number) => formatAttachmentFileSize(v),
    },
    { title: t('attachment.mime_type'), dataIndex: 'mime_type', width: 140, ellipsis: true },
    {
      title: t('log.admin'),
      dataIndex: 'admin',
      width: 120,
      render: (_, row) => formatAttachmentAdmin(row),
    },
    { title: t('table.created_at'), dataIndex: 'created_at', width: 180, sorter: true },
    {
      title: t('common.operation'),
      key: 'operation',
      width: 220,
      fixed: 'end',
      render: (_, row) => (
        <Space>
          {canInlinePreviewAttachment(row) ? (
            <Button
              type="link"
              loading={previewLoading && previewRow?.id === row.id}
              onClick={() => void handlePreview(row)}
            >
              {t('attachment.preview')}
            </Button>
          ) : null}
          <Button
            type="link"
            loading={downloadingIds.has(row.id)}
            onClick={() => void handleDownload(row)}
          >
            {t('common.download')}
          </Button>
          {getButtonState('attachment.destroy').show ? (
            <PermissionButton
              permission="attachment.destroy"
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
  } = useColumnSetting('attachment', columns)

  return (
    <PageContainer
      title={t('menu.attachment')}
      extra={
        <Space wrap>
          {getButtonState('attachment.upload').show ? (
            <Upload
              showUploadList={false}
              beforeUpload={(file) => {
                void handleUploadFile(file as File)
                return false
              }}
            >
              <PermissionButton
                permission="attachment.upload"
                type="primary"
                icon={<UploadOutlined />}
                loading={uploading}
              >
                {t('attachment.upload')}
              </PermissionButton>
            </Upload>
          ) : null}
          {getButtonState('attachment.upload').show ? (
            <PermissionButton
              permission="attachment.upload"
              icon={<ScissorOutlined />}
              onClick={() => setCropModalOpen(true)}
            >
              {t('attachment.crop_upload')}
            </PermissionButton>
          ) : null}
          {getButtonState('attachment.chunk').show ? (
            <PermissionButton
              permission="attachment.chunk"
              icon={<UploadOutlined />}
              onClick={chunk.handleLargeFileUpload}
            >
              {t('attachment.large_file_upload')}
            </PermissionButton>
          ) : null}
          {getButtonState('attachment_category.index').show ? (
            <PermissionButton
              permission="attachment_category.index"
              onClick={() => setCategoryModalOpen(true)}
            >
              {t('attachment.category_manage')}
            </PermissionButton>
          ) : null}
          {selectedRowKeys.length > 0 && getButtonState('attachment.destroy').show ? (
            <Button danger icon={<DeleteOutlined />} onClick={handleBatchDelete}>
              {t('common.delete_selected')} ({selectedRowKeys.length})
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
          { name: 'filename', label: t('attachment.filename') },
          { name: 'display_name', label: t('attachment.display_name') },
          {
            name: 'category_id',
            label: t('attachment.category'),
            type: 'select',
            options: categoryOptions,
          },
          {
            name: 'is_public',
            label: t('attachment.visibility'),
            type: 'select',
            advanced: true,
            options: [
              { label: t('attachment.visibility_public'), value: '1' },
              { label: t('attachment.visibility_private'), value: '0' },
            ],
          },
          {
            name: 'file_type',
            label: t('attachment.file_type'),
            type: 'select',
            advanced: true,
            options: [
              { label: t('attachment.file_type_image'), value: 'image' },
              { label: t('attachment.file_type_video'), value: 'video' },
              { label: t('attachment.file_type_document'), value: 'document' },
              { label: t('attachment.file_type_other'), value: 'other' },
            ],
          },
          { name: 'extension', label: t('attachment.extension'), advanced: true },
          { name: 'start_time', label: t('log.start_time'), type: 'datetime', advanced: true },
          { name: 'end_time', label: t('log.end_time'), type: 'datetime', advanced: true },
        ]}
        values={searchForm}
        onChange={onSearchFormChange}
        onSearch={handleSearch}
        onReset={handleReset}
      />
      <Table<AttachmentRow>
        rowKey="id"
        loading={loading}
        columns={filteredColumns}
        dataSource={tableData}
        scroll={{ x: 2000 }}
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

      <AttachmentCategoryModal
        open={categoryModalOpen}
        onClose={() => setCategoryModalOpen(false)}
        onChanged={() => {
          void reloadCategories()
          void refresh()
        }}
      />

      <CropUploadModal
        open={cropModalOpen}
        onClose={() => setCropModalOpen(false)}
        onSuccess={() => void refresh()}
      />

      <Modal
        open={previewOpen}
        title={previewRow?.display_name || previewRow?.filename || t('attachment.preview')}
        width={860}
        footer={null}
        destroyOnHidden
        onCancel={closePreview}
        afterOpenChange={(open) => {
          if (!open) {
            setPreviewLoading(false)
            revokePreviewBlob()
          }
        }}
      >
        <div style={{ minHeight: 200, textAlign: 'center' }}>
          {previewLoading ? (
            <div style={{ padding: 48, color: '#999' }}>...</div>
          ) : previewKind === 'image' && previewBlobUrl ? (
            <img
              src={previewBlobUrl}
              alt={previewRow?.filename || ''}
              style={{ maxWidth: '100%', maxHeight: '70vh' }}
            />
          ) : previewKind === 'video' && previewBlobUrl ? (
            <video src={previewBlobUrl} controls style={{ maxWidth: '100%', maxHeight: '70vh' }} />
          ) : previewKind === 'pdf' && previewBlobUrl ? (
            <iframe
              src={previewBlobUrl}
              title={previewRow?.filename || 'pdf'}
              style={{ width: '100%', height: '70vh', border: 'none' }}
            />
          ) : previewKind === 'text' ? (
            <pre
              style={{
                maxHeight: '70vh',
                overflow: 'auto',
                textAlign: 'left',
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-word',
                margin: 0,
                padding: 12,
                background: '#f5f7fa',
                borderRadius: 4,
                fontSize: 13,
                lineHeight: 1.5,
              }}
            >
              {previewText}
            </pre>
          ) : (
            <div style={{ padding: 48, color: '#999' }}>{t('attachment.preview_unsupported')}</div>
          )}
        </div>
      </Modal>
      <Modal
        open={chunk.visible}
        title={t('attachment.chunk_upload')}
        width={600}
        closable={false}
        maskClosable={false}
        keyboard={false}
        footer={
          chunk.status === 'success' ? (
            <Button type="primary" onClick={() => void chunk.handleClose()}>
              {t('common.confirm')}
            </Button>
          ) : chunk.status === 'exception' ? (
            <Space>
              <Button onClick={chunk.handleCancel}>{t('common.cancel')}</Button>
              <Button type="primary" onClick={chunk.handleRetry}>
                {t('common.retry')}
              </Button>
            </Space>
          ) : (
            <Button onClick={chunk.handleCancel}>{t('common.cancel')}</Button>
          )
        }
      >
        {chunk.file ? (
          <div>
            <p>
              <strong>{t('attachment.filename')}:</strong> {chunk.file.name}
            </p>
            <p>
              <strong>{t('attachment.size')}:</strong> {formatAttachmentFileSize(chunk.file.size)}
            </p>
            <Progress
              percent={Math.round(chunk.progress)}
              status={
                chunk.status === 'success'
                  ? 'success'
                  : chunk.status === 'exception'
                    ? 'exception'
                    : 'active'
              }
            />
            <div style={{ marginTop: 12, textAlign: 'center', color: '#666' }}>
              {chunk.status === 'success'
                ? t('attachment.upload_success')
                : chunk.status === 'exception'
                  ? t('attachment.upload_failed')
                  : `${t('attachment.uploading')}: ${Math.round(chunk.progress)}%`}
            </div>
          </div>
        ) : null}
      </Modal>
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
