import { useState } from 'react'
import { Space, Table, App } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useTranslation } from 'react-i18next'
import {
  deleteQuote,
  getQuoteList,
  
  exportQuote,
  
} from '@/api/quote'
import { useListPage } from '@/hooks/useListPage'
import { handlePaginatedTableChange } from '@/utils/tableChange'
import { useCrudActions } from '@/hooks/useCrudActions'
import { usePermission } from '@/hooks/usePermission'

import { useNavigate } from 'react-router-dom'

import PageContainer from '@/components/PageContainer'
import SearchForm from '@/components/SearchForm'
import PermissionButton from '@/components/PermissionButton'

import QuoteFormModal from './QuoteFormModal'
import {
  quoteInitialSearchForm,
  buildQuoteListParams,
  createQuoteSearchFields,
  transformQuoteRow,
  type QuoteRow,

} from './quote.config'

export default function QuoteList() {
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
  } = useListPage<QuoteRow>({
    fetchApi: getQuoteList,
    initialSearchForm: quoteInitialSearchForm,
    transformData: transformQuoteRow,
    buildParams: buildQuoteListParams,
  })

  const { toolbar, confirmDelete } = useCrudActions({
    createPermission: 'quote.store',
    onRefresh: refresh,
    onCreate: () => {
      setEditId(null)
      setOpen(true)
    },
    deleteApi: deleteQuote,
  })

  const navigate = useNavigate()
  const [exporting, setExporting] = useState(false)
  const handleExport = async () => {
    if (exporting) return
    setExporting(true)
    try {
      const response = await exportQuote(searchForm)
      const data = (response.data || {}) as { file_url?: string; export_id?: number | string }
      if (data.file_url) {
        window.open(data.file_url, '_blank')
        message.success(t('export.success'))
      } else {
        message.success(t('export.success'))
        navigate('/exports')
      }
    } catch (error) {
      const err = error as { response?: { status?: number }; __handled?: boolean }
      if (err.response?.status === 429) {
        message.warning(t('common.already_queued'))
      } else if (!err.__handled) {
        message.error(t('export.failed'))
      }
    } finally {
      setExporting(false)
    }
  }

  const columns: ColumnsType<QuoteRow> = [
    { title: t('table.id'), dataIndex: 'id', width: 80, sorter: true },

    { title: t('quote_no', { defaultValue: 'quote number' }), dataIndex: 'quote_no' },
    
    { title: t('customer_name', { defaultValue: 'customer name' }), dataIndex: 'customer_name' },
    
    { title: t('status', { defaultValue: '0 draft 1 confirmed' }), dataIndex: 'status' },
    
    { title: t('remark', { defaultValue: 'remark' }), dataIndex: 'remark' },
    
    { title: t('table.updated_at'), dataIndex: 'updated_at', width: 180, sorter: true },
    { title: t('table.created_at'), dataIndex: 'created_at', width: 180, sorter: true },
    {
      title: t('common.operation'),
      key: 'operation',
      width: 160,
      fixed: 'end',
      render: (_, row) => (
        <Space>
          
          {getButtonState('quote.update').show && (
            <PermissionButton
              permission="quote.update"
              type="link"
              onClick={() => {
                setEditId(row.id)
                setOpen(true)
              }}
            >
              {t('common.edit')}
            </PermissionButton>
          )}
          
          {getButtonState('quote.destroy').show && (
            <PermissionButton
              permission="quote.destroy"
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
      title={t('menu.quote')}
      extra={
        <Space>
          {toolbar}
          
          <PermissionButton
            permission="quote.export"
            loading={exporting}
            onClick={() => void handleExport()}
          >
            {t('common.export')}
          </PermissionButton>
          
        </Space>
      }
    >
      <SearchForm
        fields={createQuoteSearchFields(t)}
        values={searchForm}
        onChange={onSearchFormChange}
        onSearch={handleSearch}
        onReset={handleReset}
      />
      <Table<QuoteRow>
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
      
      <QuoteFormModal
        open={open}
        editId={editId}
        onClose={() => setOpen(false)}
        onSuccess={() => void refresh()}
      />
      
    </PageContainer>
  )
}
