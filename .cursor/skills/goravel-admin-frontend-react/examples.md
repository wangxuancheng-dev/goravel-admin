## API module

```ts
import { createCRUDApi, extendApi } from '@/utils/apiFactory'
import { normalizeListResponse } from '@/utils/normalize'
import request from '@/utils/request'

const base = createCRUDApi('widgets')

const api = extendApi(base, {
  export: (params?: Record<string, unknown>) =>
    request({ url: '/widgets/export', method: 'post', data: params }),
})

export async function getWidgetList(params?: Record<string, unknown>) {
  // Required: normalizes both data.list and data.data from backend
  return normalizeListResponse(await api.list(params))
}

export const getWidgetDetail = api.detail
export const createWidget = api.create
export const updateWidget = api.update
export const deleteWidget = api.delete
export const exportWidget = api.export
```

## Simple page (SimpleCrudPage)

```tsx
import SimpleCrudPage from '@/components/SimpleCrudPage'
import { createWidget, deleteWidget, getWidgetList, updateWidget } from '@/api/widget'

export default function WidgetList() {
  const { t } = useTranslation()
  return (
    <SimpleCrudPage
      title={t('menu.widget')}
      permissions={{ store: 'widget.store', update: 'widget.update', destroy: 'widget.destroy' }}
      fetchApi={getWidgetList}
      createApi={createWidget}
      updateApi={updateWidget}
      deleteApi={deleteWidget}
      createTitle={t('widget.add')}
      editTitle={t('widget.edit')}
      initialSearchForm={{ name: '', status: '' }}
      searchFields={[
        { name: 'name', label: t('common.name') },
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
      formFields={[
        { name: 'name', label: t('common.name'), required: true },
        { name: 'status', label: t('common.status'), type: 'status' },
      ]}
      columns={[
        { title: t('table.id'), dataIndex: 'id', width: 80, sorter: true },
        { title: t('common.name'), dataIndex: 'name' },
        { title: t('table.created_at'), dataIndex: 'created_at', width: 180, sorter: true },
      ]}
    />
  )
}
```

## Complex page (List + FormModal + config)

`widget.config.ts`:

```ts
import { entityField } from '@/utils/normalize'

export const widgetInitialSearchForm = { name: '', status: '' }
export type WidgetSearchForm = typeof widgetInitialSearchForm

export interface WidgetRow {
  id: number | string
  name?: string
  status?: number
  created_at?: string
}

export function transformWidgetRow(row: Record<string, unknown>): WidgetRow {
  return {
    id: entityField(row, 'id', '')!,
    name: String(entityField(row, 'name', '') ?? ''),
    status: Number(entityField(row, 'status', 0) ?? 0),
    created_at: String(entityField(row, 'created_at', '') ?? ''),
  }
}
```

`WidgetList.tsx` (skeleton):

```tsx
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
} = useListPage<WidgetRow, WidgetSearchForm>({
  fetchApi: getWidgetList,
  initialSearchForm: widgetInitialSearchForm,
  transformData: transformWidgetRow,
})

const { toolbar, confirmDelete } = useCrudActions({
  createPermission: 'widget.store',
  onRefresh: refresh,
  onCreate: () => { setEditId(null); setOpen(true) },
  deleteApi: deleteWidget,
})

return (
  <PageContainer title={t('menu.widget')} extra={toolbar}>
    <SearchForm
      fields={...}
      values={searchForm}
      onChange={onSearchFormChange}
      onSearch={handleSearch}
      onReset={handleReset}
    />
    <Table ... />
    <WidgetFormModal open={open} editId={editId} onClose={() => setOpen(false)} onSuccess={refresh} />
  </PageContainer>
)
```

## Permissions checklist
- Create: `resource.store`
- Update: `resource.update`
- Delete: `resource.destroy`
- Export: `resource.export`
- Use `PermissionButton` / `getButtonState` consistently with Vue admin slugs.
