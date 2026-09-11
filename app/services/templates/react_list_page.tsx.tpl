import { useState } from 'react'
import { Space<<if .HasListStatusSwitch>>, Switch<<end>>, Table } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { useTranslation } from 'react-i18next'
import {
  <<if .HasDelete>>delete<<.ModelName>>,<<end>>
  get<<.ModelName>>List,
  <<if and .HasEdit .HasListStatusSwitch>>update<<.ModelName>>,<<end>>
  <<if .HasExport>>export<<.ModelName>>,<<end>>
} from '@/api/<<.ModuleNameK>>'
import { useListPage } from '@/hooks/useListPage'
import { handlePaginatedTableChange } from '@/utils/tableChange'
import { useCrudActions } from '@/hooks/useCrudActions'
import { usePermission } from '@/hooks/usePermission'
<<if and .HasExport .ExportAsync>>
import { useQueuedExport } from '@/hooks/useQueuedExport'
<<end>>
<<if and .HasExport (not .ExportAsync)>>
import { App } from 'antd'
import { useNavigate } from 'react-router-dom'
<<end>>
import PageContainer from '@/components/PageContainer'
import SearchForm from '@/components/SearchForm'
import PermissionButton from '@/components/PermissionButton'
<<if .HasRichTextInList>>
import { extractTextFromMarkdown } from '@/utils/markdown'
<<end>>
import <<.ModelName>>FormModal from './<<.ModelName>>FormModal'
import {
  <<.ModuleNameCamel>>InitialSearchForm,
  build<<.ModelName>>ListParams,
  create<<.ModelName>>SearchFields,
  transform<<.ModelName>>Row,
  type <<.ModelName>>Row,
<<range .ListFields>>
<<- if and .ShowInList .Relation>>
  get<<.Relation.JsonName>>DisplayName,
<<- end>>
<<- end>>
} from './<<.ModuleNameCamel>>.config'

export default function <<.ModelName>>List() {
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
  } = useListPage<< "<" >><<.ModelName>>Row<< ">" >>({
    fetchApi: get<<.ModelName>>List,
    initialSearchForm: <<.ModuleNameCamel>>InitialSearchForm,
    normalizeRows: true,
    transformData: transform<<.ModelName>>Row,
    buildParams: build<<.ModelName>>ListParams,
  })

  const { toolbar, confirmDelete } = useCrudActions({
    <<if .HasCreate>>createPermission: '<<.ModuleName>>.store',<<end>>
    onRefresh: refresh,
    onCreate: () => {
      setEditId(null)
      setOpen(true)
    },
    <<if .HasDelete>>deleteApi: delete<<.ModelName>>,<<end>>
  })
<<if and .HasExport .ExportAsync>>
  const { exporting, handleExport } = useQueuedExport({
    exportApi: export<<.ModelName>>,
    getParams: () => searchForm,
    redirectPath: '/exports',
  })
<<end>>
<<if and .HasExport (not .ExportAsync)>>
  const { message } = App.useApp()
  const navigate = useNavigate()
  const [exporting, setExporting] = useState(false)
  const handleExport = async () => {
    if (exporting) return
    setExporting(true)
    try {
      const response = await export<<.ModelName>>(searchForm)
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
<<end>>

  const columns: ColumnsType<< "<" >><<.ModelName>>Row> = [
    { title: t('table.id'), dataIndex: 'id', width: 80, sorter: true },
<<range .ListFields>>
<<- if and .ShowInList (ne .Name "id") (ne .Name "created_at") (ne .Name "updated_at") (ne .Name "operation")>>
    <<if and (eq .Name "status") (eq .FormType "switch")>>
    {
      title: t('common.status'),
      dataIndex: 'status',
      width: 100,
      render: (status: number, row) => (
        <Switch
          checked={Number(status ?? 1) === 1}
          disabled={getButtonState('<<.ModuleName>>.update').disabled}
          onChange={(checked) => void handleStatusChange(row, checked)}
        />
      ),
    },
    <<else if .Relation>>
    {
      title: t('<<.Name>>', { defaultValue: '<<.Label>>' }),
      dataIndex: '<<.Name>>',
      render: (_, row) => get<<.Relation.JsonName>>DisplayName(row.<<.Relation.JsonName>>),
    },
    <<else if or (eq .FormType "editor") (eq .FormType "markdown")>>
    {
      title: t('<<.Name>>', { defaultValue: '<<.Label>>' }),
      dataIndex: '<<.Name>>',
      ellipsis: true,
      width: 220,
      render: (value: unknown) => extractTextFromMarkdown(String(value ?? '')).slice(0, 120) || '-',
    },
    <<else if eq .FormType "image-upload">>
    {
      title: t('<<.Name>>', { defaultValue: '<<.Label>>' }),
      dataIndex: '<<.Name>>',
      width: 100,
      render: (value: unknown) =>
        value ? <img src={String(value)} alt="" style={{ width: 48, height: 48, objectFit: 'cover', borderRadius: 4 }} /> : '-',
    },
    <<else>>
    { title: t('<<.Name>>', { defaultValue: '<<.Label>>' }), dataIndex: '<<.Name>>'<<if .Sortable>>, sorter: true<<end>> },
    <<end>>
<<- end>>
<<- end>>
    { title: t('table.updated_at'), dataIndex: 'updated_at', width: 180, sorter: true },
    { title: t('table.created_at'), dataIndex: 'created_at', width: 180, sorter: true },
    {
      title: t('common.operation'),
      key: 'operation',
      width: 160,
      fixed: 'end',
      render: (_, row) => (
        <Space>
          <<if .HasEdit>>
          {getButtonState('<<.ModuleName>>.update').show && (
            <PermissionButton
              permission="<<.ModuleName>>.update"
              type="link"
              onClick={() => {
                setEditId(row.id)
                setOpen(true)
              }}
            >
              {t('common.edit')}
            </PermissionButton>
          )}
          <<end>>
          <<if .HasDelete>>
          {getButtonState('<<.ModuleName>>.destroy').show && (
            <PermissionButton
              permission="<<.ModuleName>>.destroy"
              type="link"
              danger
              onClick={() => confirmDelete(row.id)}
            >
              {t('common.delete')}
            </PermissionButton>
          )}
          <<end>>
        </Space>
      ),
    },
  ]

  <<if and .HasEdit .HasListStatusSwitch>>
  const handleStatusChange = async (row: <<.ModelName>>Row, checked: boolean) => {
    await update<<.ModelName>>(row.id, { status: checked ? 1 : 0 })
    await refresh()
  }
  <<end>>

  return (
    <PageContainer
      title={t('menu.<<.ModuleName>>')}
      extra={
        <Space>
          {toolbar}
          <<if .HasExport>>
          <PermissionButton
            permission="<<.ModuleName>>.export"
            loading={exporting}
            onClick={() => void handleExport()}
          >
            {t('common.export')}
          </PermissionButton>
          <<end>>
        </Space>
      }
    >
      <SearchForm
        fields={create<<.ModelName>>SearchFields(t)}
        values={searchForm}
        onChange={onSearchFormChange}
        onSearch={handleSearch}
        onReset={handleReset}
      />
      <Table<< "<" >><<.ModelName>>Row<< ">" >>
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
      <<if or .HasCreate .HasEdit>>
      << "<" >><<.ModelName>>FormModal
        open={open}
        editId={editId}
        onClose={() => setOpen(false)}
        onSuccess={() => void refresh()}
      />
      <<end>>
    </PageContainer>
  )
}
