import { useTranslation } from 'react-i18next'
import SimpleCrudPage from '@/components/SimpleCrudPage'
import {
  createPermission,
  deletePermission,
  getPermissionList,
  updatePermission,
} from '@/api/permission'
import StatusTag from '@/components/StatusTag'

interface PermissionRow {
  id: number | string
  name?: string
  slug?: string
  description?: string
  http_method?: string
  http_path?: string
  status?: number
  created_at?: string
}

export default function PermissionList() {
  const { t } = useTranslation()

  return (
    <SimpleCrudPage<PermissionRow>
      title={t('menu.permission_management')}
      permissions={{
        store: 'permission.store',
        update: 'permission.update',
        destroy: 'permission.destroy',
      }}
      columnSettingKey="permission"
      fetchApi={getPermissionList}
      createApi={createPermission}
      updateApi={updatePermission}
      deleteApi={deletePermission}
      requireSensitiveConfirmOnDelete
      createTitle={t('permission.add')}
      editTitle={t('permission.edit')}
      initialSearchForm={{ name: '', slug: '' }}
      searchFields={[
        { name: 'name', label: t('permission.name') },
        { name: 'slug', label: t('permission.slug') },
      ]}
      formFields={[
        { name: 'name', label: t('permission.name'), required: true },
        { name: 'slug', label: t('permission.slug'), required: true },
        { name: 'http_method', label: t('common.method') },
        { name: 'http_path', label: t('common.path') },
        { name: 'description', label: t('common.description'), type: 'textarea' },
        { name: 'status', label: t('common.status'), type: 'status' },
      ]}
      columns={[
        { title: t('table.id'), dataIndex: 'id', width: 80, sorter: true },
        { title: t('permission.name'), dataIndex: 'name' },
        { title: t('permission.slug'), dataIndex: 'slug' },
        { title: t('common.method'), dataIndex: 'http_method', width: 100 },
        { title: t('common.path'), dataIndex: 'http_path', ellipsis: true },
        {
          title: t('common.status'),
          dataIndex: 'status',
          width: 100,
          render: (status: number) => <StatusTag status={status} />,
        },
        { title: t('table.created_at'), dataIndex: 'created_at', width: 180, sorter: true },
      ]}
    />
  )
}
