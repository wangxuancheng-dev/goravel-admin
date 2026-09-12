<template>
  <ListPage
    ref="listPageRef"
    page-class="permission"
    :title="$t('menu.permission')"
    :add-button-text="$t('permission.add_permission')"
    :add-button-disabled="getButtonState('permission.store').disabled"
    :search-form="searchForm"
    :search-fields="searchFields"
    :initial-search-values="permissionInitialSearchForm"
    i18n-prefix="permission"
    :table-data="tableData"
    :loading="loading"
    :table-columns="tableColumns"
    :table-key="`table-${tableColumns.length}`"
    :pagination="pagination"
    show-toolbar
    show-column-setting
    :visible-columns="visibleColumns"
    :all-columns="allColumns"
    :default-visible-columns="defaultVisibleColumns"
    :column-order="columnOrder"
    :fixed-columns="fixedColumns"
    :on-column-setting-confirm="handleColumnSettingConfirm"
    @add="handleAdd"
    @search="handleSearch"
    @reset="handleReset"
    @refresh="loadData"
    @page-change="loadData"
    @sort-change="handleSortChange"
  >
    <template #status="{ row }">
      <el-tag :type="rowStatus(row) === 1 ? 'success' : 'danger'">
        {{ rowStatus(row) === 1 ? $t('common.enabled') : $t('common.disabled') }}
      </el-tag>
    </template>

    <template #menu="{ row }">
      <span>{{ getMenuDisplayTitle(row.menu) }}</span>
    </template>

    <template #operation="{ row }">
      <TableActionButtons
        :row="row"
        :primary-actions="operationActions"
        :get-button-state="getButtonState"
      />
    </template>

    <template #form>
      <PermissionForm
        v-model="dialogVisible"
        :edit-id="editId"
        :menu-tree-data="menuTreeData"
        @success="handleFormSuccess"
      />
    </template>
  </ListPage>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ListPage from '@/components/ListPage.vue'
import TableActionButtons from '@/components/TableActionButtons.vue'
import PermissionForm from './PermissionForm.vue'
import { useStandardListPage } from '@/composables/useStandardListPage'
import { useColumnSetting } from '@/composables/useColumnSetting'
import { createCrudActions, rowStatus } from '@/utils/listPageHelpers'
import { getMenuTitle as getMenuTitleUtil } from '@/utils/menuTranslation'
import { getPermissionList, deletePermission } from '@/api/permission'
import { getMenuTree } from '@/api/menu'
import {
  permissionInitialSearchForm,
  createPermissionSearchFields,
  createPermissionTableColumns
} from './permission.config'

const { t, te } = useI18n()
const listPageRef = ref(null)
const menuTreeData = ref([])

const allTableColumns = computed(() => createPermissionTableColumns(t))

const {
  tableColumns,
  visibleColumns,
  allColumns,
  defaultVisibleColumns,
  columnOrder,
  fixedColumns,
  handleColumnSettingConfirm
} = useColumnSetting('permission', allTableColumns)

const searchFields = computed(() => createPermissionSearchFields(t))

const {
  pagination,
  tableData,
  loading,
  searchForm,
  loadData,
  handleSearch,
  handleReset,
  handleSortChange,
  initDefaultSort,
  dialogVisible,
  editId,
  handleAdd,
  handleEdit,
  handleFormSuccess,
  handleDelete,
  getButtonState
} = useStandardListPage({
  fetchApi: getPermissionList,
  initialSearchForm: permissionInitialSearchForm,
  defaultSort: 'id:desc',
  deleteApi: deletePermission,
  requireSensitiveConfirm: true,
  normalizeRows: false,
  immediate: false,
  tableRef: computed(() => listPageRef.value?.tableRef?.tableRef)
})

const operationActions = computed(() =>
  createCrudActions(t, 'permission', {
    onEdit: handleEdit,
    onDelete: handleDelete
  })
)

const getMenuTitle = (menu) => {
  if (!menu || typeof menu !== 'object') return '-'
  return getMenuTitleUtil(t, te, menu) || '-'
}

const convertMenuToTreeData = (menus) =>
  menus.map((menu) => {
    const menuId = menu.id
    const title = getMenuTitle(menu)
    const path = menu.path || ''
    const node = {
      value: menuId,
      label: path ? `${title} (${path})` : title,
      title,
      path
    }
    if (menu.children?.length) {
      node.children = convertMenuToTreeData(menu.children)
    }
    return node
  })

const loadMenuList = async () => {
  try {
    const { data } = await getMenuTree()
    menuTreeData.value = convertMenuToTreeData(data.menus || [])
  } catch (error) {
    console.error('Load menu list failed:', error)
  }
}

const getMenuDisplayTitle = (menu) => {
  if (!menu || typeof menu !== 'object') return '-'
  return getMenuTitle(menu)
}

onMounted(() => {
  initDefaultSort()
  loadMenuList()
  loadData()
})
</script>
