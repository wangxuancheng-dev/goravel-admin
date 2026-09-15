import { useEffect, useState } from 'react'
import { Button, Space<<if .HasListStatusSwitch>>, Switch<<end>>, Table } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { SettingOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import {
  <<if .HasDelete>>delete<<.ModelName>>,<<end>>
  get<<.ModelName>>List,
  <<if and .HasEdit .HasListStatusSwitch>>update<<.ModelName>>,<<end>>
} from '@/api/<<.ModuleNameK>>'
import { useCrudActions } from '@/hooks/useCrudActions'
import { usePermission } from '@/hooks/usePermission'
import { useUnhandledError } from '@/hooks/useUnhandledError'
import { useColumnSetting } from '@/hooks/useColumnSetting'
import PageContainer from '@/components/PageContainer'
import ColumnSettingDialog from '@/components/ColumnSettingDialog'
import SearchForm from '@/components/SearchForm'
import StatusTag from '@/components/StatusTag'
import PermissionButton from '@/components/PermissionButton'
import <<.ModelName>>FormModal from './<<.ModelName>>FormModal'
import {
  <<.ModuleNameCamel>>InitialSearchForm,
  build<<.ModelName>>ListParams,
  collect<<.ModelName>>Ids,
  create<<.ModelName>>SearchFields,
  map<<.ModelName>>Rows,
  normalize<<.ModelName>>TreeList,
  type <<.ModelName>>Row,
<<range .ListFields>>
<<- if and .ShowInList .Relation>>
  get<<.Relation.JsonName>>DisplayName,
<<- end>>
<<- end>>
} from './<<.ModuleNameCamel>>.config'

export default function <<.ModelName>>List() {
  const { t } = useTranslation()
  const showError = useUnhandledError()
  const { getButtonState } = usePermission()
  const [data, setData] = useState<< "<" >><<.ModelName>>Row[]>([])
  const [loading, setLoading] = useState(false)
  const [open, setOpen] = useState(false)
  const [editId, setEditId] = useState<string | number | null>(null)
  const [expandedKeys, setExpandedKeys] = useState<Array<string | number>>([])
  const [searchForm, setSearchForm] = useState(<<.ModuleNameCamel>>InitialSearchForm)

  const load = async (params: Record<string, unknown> = searchForm) => {
    setLoading(true)
    try {
      const res = await get<<.ModelName>>List(build<<.ModelName>>ListParams(params as typeof <<.ModuleNameCamel>>InitialSearchForm, {}))
      const list = (res.data?.list ?? res.data?.data ?? res.data ?? []) as unknown[]
      const rows = normalize<<.ModelName>>TreeList(Array.isArray(list) ? list : [])
      setData(rows)
      setExpandedKeys(collect<<.ModelName>>Ids(rows))
    } catch (error) {
      showError(error, t('common.query_failed'))
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const { toolbar, confirmDelete } = useCrudActions({
    <<if .HasCreate>>createPermission: '<<.ModuleName>>.store',<<end>>
    onRefresh: () => void load(),
    onCreate: () => {
      setEditId(null)
      setOpen(true)
    },
    <<if .HasDelete>>deleteApi: delete<<.ModelName>>,<<end>>
  })

  const allExpanded = expandedKeys.length > 0

  <<if and .HasEdit .HasListStatusSwitch>>
  const handleStatusChange = async (row: <<.ModelName>>Row, checked: boolean) => {
    await update<<.ModelName>>(row.id, { status: checked ? 1 : 0 })
    await load()
  }
  <<end>>

  const columns: ColumnsType<< "<" >><<.ModelName>>Row> = [
<<range .ListFields>>
<<- if and .ShowInList (ne .Name "id") (ne .Name "created_at") (ne .Name "updated_at") (ne .Name "operation")>>
      <<if and (eq .Name "status") (eq .FormType "switch") $.HasEdit>>
      {
        title: t('common.status'),
        dataIndex: 'status',
        width: 100,
        render: (status: number, row) => (
          <Switch
            checked={Number(status ?? 1) === 1}
            disabled={getButtonState('<<$.ModuleName>>.update').disabled}
            onChange={(checked) => void handleStatusChange(row, checked)}
          />
        ),
      },
      <<else if and (eq .Name "status") (ne .FormType "switch")>>
      {
        title: t('common.status'),
        dataIndex: 'status',
        width: 100,
        render: (status: number) => <StatusTag status={status} />,
      },
      <<else if .Relation>>
      {
        title: t('<<.Name>>', { defaultValue: '<<.Label>>' }),
        dataIndex: '<<.Name>>',
        render: (_, row) => get<<.Relation.JsonName>>DisplayName(row.<<.Relation.JsonName>>),
      },
      <<else>>
      { title: t('<<.Name>>', { defaultValue: '<<.Label>>' }), dataIndex: '<<.Name>>'<<if .Sortable>>, sorter: true<<end>> },
      <<end>>
<<- end>>
<<- end>>
      { title: t('table.updated_at'), dataIndex: 'updated_at', width: 180 },
      { title: t('table.created_at'), dataIndex: 'created_at', width: 180 },
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
                onClick={() => confirmDelete(row.id, String(row.<<.TreeLabelField>> ?? row.id))}
              >
                {t('common.delete')}
              </PermissionButton>
            )}
            <<end>>
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
  } = useColumnSetting('<<.ModuleName>>', columns)

  return (
    <PageContainer
      title={t('menu.<<.ModuleName>>')}
      extra={
        <Space>
          <Button onClick={() => setExpandedKeys(allExpanded ? [] : collect<<.ModelName>>Ids(data))}>
            {allExpanded ? t('common.collapse') : t('common.expand')}
          </Button>
          {toolbar}
          <Button icon={<SettingOutlined />} onClick={openColumnSetting}>
            {t('common.column_setting')}
          </Button>
        </Space>
      }
    >
      <SearchForm
        fields={create<<.ModelName>>SearchFields(t)}
        values={searchForm}
        onChange={setSearchForm}
        onSearch={() => void load(searchForm)}
        onReset={() => {
          setSearchForm(<<.ModuleNameCamel>>InitialSearchForm)
          void load(<<.ModuleNameCamel>>InitialSearchForm)
        }}
      />
      <Table<< "<" >><<.ModelName>>Row<< ">" >>
        rowKey="id"
        loading={loading}
        columns={filteredColumns}
        dataSource={data}
        pagination={false}
        scroll={{ x: 960 }}
        expandable={{
          expandedRowKeys: expandedKeys,
          onExpandedRowsChange: (keys) => setExpandedKeys(keys as Array<string | number>),
        }}
      />
      <<if or .HasCreate .HasEdit>>
      << "<" >><<.ModelName>>FormModal
        open={open}
        editId={editId}
        treeData={data}
        onClose={() => setOpen(false)}
        onSuccess={() => void load()}
      />
      <<end>>
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
