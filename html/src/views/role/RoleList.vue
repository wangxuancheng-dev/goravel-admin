<template>
  <ListPage
    ref="listPageRef"
    page-class="role"
    :title="$t('menu.role')"
    :add-button-text="$t('role.add_role')"
    :add-button-disabled="getButtonState('role.store').disabled"
    :search-form="searchForm"
    :search-fields="searchFields"
    :initial-search-values="roleInitialSearchForm"
    i18n-prefix="role"
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
      <el-switch
        :model-value="rowStatus(row) === 1"
        :disabled="isProtectedRole(row) || getButtonState('role.update').disabled"
        @change="(val) => handleStatusChange(row, val)"
      />
    </template>

    <template #operation="{ row }">
      <TableActionButtons
        :row="row"
        :primary-actions="getPrimaryActions(row)"
        :get-button-state="getButtonState"
      />
    </template>

    <template #form>
      <RoleForm
        ref="roleFormRef"
        v-model="dialogVisible"
        :edit-id="editId"
        @success="handleFormSuccess"
      />
    </template>
  </ListPage>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import ListPage from '@/components/ListPage.vue'
import TableActionButtons from '@/components/TableActionButtons.vue'
import RoleForm from './RoleForm.vue'
import { useStandardListPage } from '@/composables/useStandardListPage'
import { useColumnSetting } from '@/composables/useColumnSetting'
import { rowStatus } from '@/utils/listPageHelpers'
import { getRoleList, deleteRole, updateRole } from '@/api/role'
import {
  roleInitialSearchForm,
  createRoleSearchFields,
  createRoleTableColumns,
  isProtectedRole
} from './role.config'

const { t } = useI18n()
const listPageRef = ref(null)
const roleFormRef = ref(null)

const allTableColumns = computed(() => createRoleTableColumns(t))

const {
  tableColumns,
  visibleColumns,
  allColumns,
  defaultVisibleColumns,
  columnOrder,
  fixedColumns,
  handleColumnSettingConfirm
} = useColumnSetting('role', allTableColumns)

const searchFields = computed(() => createRoleSearchFields(t))

const {
  pagination,
  tableData,
  loading,
  searchForm,
  dialogVisible,
  editId,
  loadData,
  handleSearch,
  handleReset,
  handleSortChange,
  handleEdit,
  handleFormSuccess,
  handleDelete: handleDeleteRow,
  getButtonState
} = useStandardListPage({
  fetchApi: getRoleList,
  initialSearchForm: roleInitialSearchForm,
  defaultSort: 'id:desc',
  deleteApi: deleteRole,
  requireSensitiveConfirm: true,
  normalizeRows: false,
  tableRef: computed(() => listPageRef.value?.tableRef?.tableRef),
  beforeDelete: (row) => {
    if (isProtectedRole(row)) {
      ElMessage.warning(t('role.protected_cannot_delete'))
      return false
    }
  }
})

const handleAdd = () => {
  if (dialogVisible.value) {
    dialogVisible.value = false
    setTimeout(() => {
      editId.value = null
      dialogVisible.value = true
    }, 200)
  } else {
    editId.value = null
    dialogVisible.value = true
  }
}

const handleStatusChange = async (row, newStatus) => {
  if (isProtectedRole(row) && !newStatus) {
    ElMessage.warning(t('role.protected_cannot_disable'))
    loadData()
    return
  }

  try {
    const statusValue = newStatus ? 1 : 0
    await updateRole(row.id, { status: statusValue })
    ElMessage.success(newStatus ? t('role.enable_success') : t('role.disable_success'))
    const role = tableData.value.find((item) => item.id === row.id)
    if (role) {
      role.status = statusValue
    }
  } catch (error) {
    loadData()
    if (!error.__handled) {
      ElMessage.error(error.message || t('common.operation_failed'))
    }
  }
}

const handleDelete = (row) => handleDeleteRow(row, loadData)

const getPrimaryActions = (row) => [
  {
    key: 'edit',
    label: t('common.edit'),
    type: 'primary',
    permission: 'role.update',
    handler: handleEdit
  },
  {
    key: 'delete',
    label: t('common.delete'),
    type: 'danger',
    permission: 'role.destroy',
    show: () => !isProtectedRole(row),
    handler: handleDelete
  }
]
</script>
