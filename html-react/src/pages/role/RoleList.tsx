import { useState } from 'react'
import { Button, Space, Table } from 'antd'
import type { ColumnsType } from 'antd/es/table'
import { SettingOutlined } from '@ant-design/icons'
import { useTranslation } from 'react-i18next'
import { deleteRole, getRoleList } from '@/api/role'
import { useListPage } from '@/hooks/useListPage'
import { handlePaginatedTableChange } from '@/utils/tableChange'
import { useCrudActions } from '@/hooks/useCrudActions'
import { usePermission } from '@/hooks/usePermission'
import { useColumnSetting } from '@/hooks/useColumnSetting'
import PageContainer from '@/components/PageContainer'
import ColumnSettingDialog from '@/components/ColumnSettingDialog'
import SearchForm from '@/components/SearchForm'
import StatusTag from '@/components/StatusTag'
import PermissionButton from '@/components/PermissionButton'
import RoleFormModal from './RoleFormModal'
import { roleInitialSearchForm, transformRoleRow, type RoleRow } from './role.config'

export default function RoleList() {
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
  } = useListPage<RoleRow, typeof roleInitialSearchForm>({
    fetchApi: getRoleList,
    initialSearchForm: roleInitialSearchForm,
    normalizeRows: true,
    transformData: transformRoleRow,
  })

  const { toolbar, confirmDelete } = useCrudActions({
    createPermission: 'role.store',
    onRefresh: refresh,
    onCreate: () => {
      setEditId(null)
      setOpen(true)
    },
    deleteApi: deleteRole,
    requireSensitiveConfirm: true,
  })

  const columns: ColumnsType<RoleRow> = [
    { title: t('table.id'), dataIndex: 'id', width: 80, sorter: true },
    { title: t('role.name'), dataIndex: 'name' },
    { title: t('role.slug'), dataIndex: 'slug' },
    { title: t('common.description'), dataIndex: 'description', ellipsis: true },
    { title: t('common.sort'), dataIndex: 'sort', width: 80, sorter: true },
    {
      title: t('common.status'),
      dataIndex: 'status',
      width: 100,
      render: (status: number) => <StatusTag status={status} />,
    },
    { title: t('table.created_at'), dataIndex: 'created_at', width: 180, sorter: true },
    {
      title: t('common.operation'),
      key: 'operation',
      width: 160,
      fixed: 'end',
      render: (_, row) => (
        <Space>
          {getButtonState('role.update').show && (
            <PermissionButton
              permission="role.update"
              type="link"
              onClick={() => {
                setEditId(row.id)
                setOpen(true)
              }}
            >
              {t('common.edit')}
            </PermissionButton>
          )}
          {getButtonState('role.destroy').show && row.slug !== 'super-admin' && (
            <PermissionButton
              permission="role.destroy"
              type="link"
              danger
              onClick={() => confirmDelete(row.id, row.name)}
            >
              {t('common.delete')}
            </PermissionButton>
          )}
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
  } = useColumnSetting('role', columns)

  return (
    <PageContainer
      title={t('menu.role_management')}
      extra={
        <Space>
          {toolbar}
          <Button icon={<SettingOutlined />} onClick={openColumnSetting}>
            {t('common.column_setting')}
          </Button>
        </Space>
      }
    >
      <SearchForm
        fields={[
          { name: 'name', label: t('role.name') },
          {
            name: 'status',
            label: t('common.status'),
            type: 'select',
            options: [
              { label: t('common.enabled'), value: 1 },
              { label: t('common.disabled'), value: 0 },
            ],
          },
        ]}
        values={searchForm}
        onChange={onSearchFormChange}
        onSearch={handleSearch}
        onReset={handleReset}
      />
      <Table<RoleRow>
        rowKey="id"
        loading={loading}
        columns={filteredColumns}
        dataSource={tableData}
        scroll={{ x: 1000 }}
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
      <RoleFormModal
        open={open}
        editId={editId}
        onClose={() => setOpen(false)}
        onSuccess={() => void refresh()}
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
