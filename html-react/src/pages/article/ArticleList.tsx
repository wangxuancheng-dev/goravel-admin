import { useState } from 'react'
import { Space, Table } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useTranslation } from 'react-i18next'
import {
  deleteArticle,
  getArticleList,
  
  exportArticle,
} from '@/api/article'
import { useListPage } from '@/hooks/useListPage'
import { handlePaginatedTableChange } from '@/utils/tableChange'
import { useCrudActions } from '@/hooks/useCrudActions'
import { usePermission } from '@/hooks/usePermission'

import { useQueuedExport } from '@/hooks/useQueuedExport'

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
    
    { title: t('status', { defaultValue: '0:未发布 1:发布' }), dataIndex: 'status' },
    
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

  return (
    <PageContainer
      title={t('menu.article')}
      extra={
        <Space>
          {toolbar}
          
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
