import { useState } from 'react'
import { Space, Switch, Table, App, Upload } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useTranslation } from 'react-i18next'
import {
  deleteArticle,
  getArticleList,
  updateArticle,
  exportArticle,
  importArticle,
} from '@/api/article'
import { useListPage } from '@/hooks/useListPage'
import { handlePaginatedTableChange } from '@/utils/tableChange'
import { useCrudActions } from '@/hooks/useCrudActions'
import { usePermission } from '@/hooks/usePermission'

import { useQueuedExport } from '@/hooks/useQueuedExport'

import { UploadOutlined } from '@ant-design/icons'
import type { UploadProps } from 'antd/es/upload'

import PageContainer from '@/components/PageContainer'
import SearchForm from '@/components/SearchForm'
import PermissionButton from '@/components/PermissionButton'

import { extractTextFromMarkdown } from '@/utils/markdown'

import ArticleFormModal from './ArticleFormModal'
import {
  articleInitialSearchForm,
  buildArticleListParams,
  createArticleSearchFields,
  transformArticleRow,
  type ArticleRow,

  getadminDisplayName,
} from './article.config'

export default function ArticleList() {
  const { t } = useTranslation()
  const { getButtonState } = usePermission()
  const [open, setOpen] = useState(false)
  const [editId, setEditId] = useState<string | number | null>(null)

  const { message } = App.useApp()

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
  } = useListPage<ArticleRow>({
    fetchApi: getArticleList,
    initialSearchForm: articleInitialSearchForm,
    normalizeRows: true,
    transformData: transformArticleRow,
    buildParams: buildArticleListParams,
  })

  const { toolbar, confirmDelete } = useCrudActions({
    createPermission: 'article.store',
    onRefresh: refresh,
    onCreate: () => {
      setEditId(null)
      setOpen(true)
    },
    deleteApi: deleteArticle,
  })

  const { exporting, handleExport } = useQueuedExport({
    exportApi: exportArticle,
    getParams: () => searchForm,
    redirectPath: '/exports',
  })

  const [importing, setImporting] = useState(false)
  const uploadProps: UploadProps = {
    accept: '.csv',
    showUploadList: false,
    beforeUpload: (file) => {
      if (!file.name.toLowerCase().endsWith('.csv')) {
        message.error(t('common.invalid_file_type'))
        return false
      }
      setImporting(true)
      void importArticle(file as File)
        .then((res) => {
          const payload = (res.data || {}) as Record<string, unknown>
          const result = ((payload.data || payload) as Record<string, unknown>) || {}
          if (result.async && result.import_id) {
            message.success(t('export.task_submitted'))
            return
          }
          const successCount = Number(result.success_count || 0)
          const failedCount = Number(result.failed_count || 0)
          const errors = Array.isArray(result.errors) ? (result.errors as string[]) : []
          if (successCount > 0) {
            message.success(t('common.import_success'))
            if (failedCount > 0 && errors.length) {
              message.warning(errors.slice(0, 10).join('\n'))
            }
            void refresh()
          } else {
            message.warning(t('common.import_no_data'))
            if (errors.length) {
              message.error(errors.slice(0, 10).join('\n'))
            }
          }
        })
        .catch(() => message.error(t('common.operation_failed')))
        .finally(() => setImporting(false))
      return false
    },
  }

  const columns: ColumnsType<ArticleRow> = [
    { title: t('table.id'), dataIndex: 'id', width: 80, sorter: true },

    {
      title: t('admin_id', { defaultValue: '管理员ID' }),
      dataIndex: 'admin_id',
      render: (_, row) => getadminDisplayName(row.admin),
    },
    
    { title: t('title', { defaultValue: '标题' }), dataIndex: 'title' },
    
    {
      title: t('content', { defaultValue: '内容' }),
      dataIndex: 'content',
      ellipsis: true,
      width: 220,
      render: (value: unknown) => extractTextFromMarkdown(String(value ?? '')).slice(0, 120) || '-',
    },
    
    {
      title: t('common.status'),
      dataIndex: 'status',
      width: 100,
      render: (status: number, row) => (
        <Switch
          checked={Number(status ?? 1) === 1}
          disabled={getButtonState('article.update').disabled}
          onChange={(checked) => void handleStatusChange(row, checked)}
        />
      ),
    },
    
    { title: t('table.updated_at'), dataIndex: 'updated_at', width: 180, sorter: true },
    { title: t('table.created_at'), dataIndex: 'created_at', width: 180, sorter: true },
    {
      title: t('common.operation'),
      key: 'operation',
      width: 160,
      fixed: 'end',
      render: (_, row) => (
        <Space>
          
          {getButtonState('article.update').show && (
            <PermissionButton
              permission="article.update"
              type="link"
              onClick={() => {
                setEditId(row.id)
                setOpen(true)
              }}
            >
              {t('common.edit')}
            </PermissionButton>
          )}
          
          {getButtonState('article.destroy').show && (
            <PermissionButton
              permission="article.destroy"
              type="link"
              danger
              onClick={() => confirmDelete(row.id)}
            >
              {t('common.delete')}
            </PermissionButton>
          )}
          
        </Space>
      ),
    },
  ]

  const handleStatusChange = async (row: ArticleRow, checked: boolean) => {
    await updateArticle(row.id, { status: checked ? 1 : 0 })
    await refresh()
  }
  
  return (
    <PageContainer
      title={t('menu.article')}
      extra={
        <Space>
          {toolbar}
          
          <Upload {...uploadProps}>
            <PermissionButton permission="article.import" icon={<UploadOutlined />} loading={importing}>
              {t('common.import')}
            </PermissionButton>
          </Upload>
          
          <PermissionButton
            permission="article.export"
            loading={exporting}
            onClick={() => void handleExport()}
          >
            {t('common.export')}
          </PermissionButton>
          
        </Space>
      }
    >
      <SearchForm
        fields={createArticleSearchFields(t)}
        values={searchForm}
        onChange={onSearchFormChange}
        onSearch={handleSearch}
        onReset={handleReset}
      />
      <Table<ArticleRow>
        rowKey="id"
        loading={loading}
        columns={columns}
        dataSource={tableData}
        scroll={{ x: 960 }}
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
      
      <ArticleFormModal
        open={open}
        editId={editId}
        onClose={() => setOpen(false)}
        onSuccess={() => void refresh()}
      />
      
    </PageContainer>
  )
}
