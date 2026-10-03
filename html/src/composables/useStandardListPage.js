import { onMounted } from 'vue'
import { useListPage } from './useListPage'
import { useCrud } from './useCrud'
import { usePermission } from './usePermission'

/**
 * Standard paginated CRUD list page composable.
 * Combines useListPage + useCrud + usePermission with common handlers.
 */
export function useStandardListPage(options = {}) {
  const {
    fetchApi,
    initialSearchForm = {},
    defaultSort = 'id:desc',
    fieldMapping = {},
    tableRef = null,
    deleteApi = null,
    batchDeleteApi = null,
    normalizeRows = false,
    immediate = true,
    buildParams = null,
    onLoadSuccess = null,
    transformData = null,
    onSearch = null,
    onReset = null,
    beforeDelete = null,
    ...crudOptions
  } = options

  const { getButtonState } = usePermission()

  const crud = useCrud({
    deleteApi: deleteApi || undefined,
    batchDeleteApi: batchDeleteApi || undefined,
    beforeDelete,
    ...crudOptions
  })

  const list = useListPage({
    fetchApi,
    initialSearchForm,
    defaultSort,
    fieldMapping,
    tableRef,
    normalizeRows,
    buildParams,
    onLoadSuccess,
    transformData,
    onSearch,
    onReset
  })

  const handleFormSuccess = () => {
    crud.handleFormSuccess(list.loadData)
  }

  const handleDelete = (row) => {
    if (deleteApi) {
      crud.handleDelete(row, list.loadData)
    }
  }

  if (immediate) {
    onMounted(() => {
      list.initDefaultSort()
      list.loadData()
    })
  }

  return {
    ...list,
    dialogVisible: crud.dialogVisible,
    editId: crud.editId,
    handleAdd: crud.handleAdd,
    handleEdit: crud.handleEdit,
    handleClose: crud.handleClose,
    handleBatchDelete: crud.handleBatchDelete,
    getButtonState,
    handleFormSuccess,
    handleDelete
  }
}
