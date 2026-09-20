<template>
  <ListPage
    ref="listPageRef"
    page-class="admin"
    :title="$t('menu.admin')"
    :add-button-text="$t('admin.add_admin')"
    :add-button-disabled="getButtonState('admin.store').disabled"
    :search-form="searchForm"
    :search-fields="searchFields"
    :initial-search-values="adminInitialSearchForm"
    i18n-prefix="admin"
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
    <template #extra-buttons>
      <el-button
        type="success"
        :disabled="getButtonState('admin.export').disabled || isExporting"
        :loading="isExporting"
        @click="handleExport"
      >
        {{ $t('common.export') }}
      </el-button>
    </template>

    <template #status="{ row }">
      <el-switch
        :model-value="rowStatus(row) === 1"
        :disabled="isProtectedAdmin(row.id) || getButtonState('admin.update').disabled"
        @change="(val) => handleStatusChange(row, val)"
      />
    </template>

    <template #department="{ row }">
      {{ getDepartmentDisplayName(row.department) }}
    </template>

    <template #position="{ row }">
      {{ getPositionDisplayName(row.position) }}
    </template>

    <template #roles="{ row }">
      <template v-if="row.roles?.length">
        <el-tag
          v-for="role in getUniqueRoles(row.roles)"
          :key="role.id"
          style="margin-right: 5px; margin-bottom: 2px"
        >
          {{ role.name || role.Name }}
        </el-tag>
      </template>
      <span v-else>-</span>
    </template>

    <template #is_2fa_bound="{ row }">
      {{ row.is_2fa_bound ? $t('admin.google_auth_bound') : $t('admin.google_auth_not_bound') }}
    </template>

    <template #operation="{ row }">
      <TableActionButtons
        :row="row"
        :primary-actions="getPrimaryActions(row)"
        :more-actions="getMoreActions(row)"
        :get-button-state="getButtonState"
        @action="handleAction"
      />
    </template>

    <template #form>
      <AdminForm
        ref="adminFormRef"
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
import { useQueuedExport } from '@/composables/useQueuedExport'
import { ElMessage, ElMessageBox } from 'element-plus'
import ListPage from '@/components/ListPage.vue'
import TableActionButtons from '@/components/TableActionButtons.vue'
import AdminForm from './AdminForm.vue'
import { useStandardListPage } from '@/composables/useStandardListPage'
import { useColumnSetting } from '@/composables/useColumnSetting'
import { rowStatus } from '@/utils/listPageHelpers'
import { getField } from '@/utils/normalizeFormData'
import { getStatusOptions } from '@/utils/fieldOptions'
import {
  getAdminList,
  deleteAdmin,
  updateAdmin,
  exportAdmin,
  resetPassword,
  kickOutUser,
  unbindAdminGoogleAuth,
  resetAdminGoogleAuth
} from '@/api/admin'
import logger from '@/utils/logger'
import ErrorHandler from '@/utils/errorHandler'
import { promptSensitiveConfirm } from '@/utils/sensitiveConfirm'
import { compact, map as lodashMap } from 'lodash-es'
import {
  adminInitialSearchForm,
  createAdminSearchFields,
  createAdminTableColumns,
  isProtectedAdmin,
  getUniqueRoles,
  getDepartmentDisplayName,
  getPositionDisplayName
} from './admin.config'

const { t } = useI18n()
const listPageRef = ref(null)
const adminFormRef = ref(null)

const { isExporting, handleExport } = useQueuedExport({
  exportApi: exportAdmin,
  getParams: () => searchForm,
  requireExportId: false
})

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
  handleAdd,
  handleDelete: handleDeleteRow,
  getButtonState
} = useStandardListPage({
  fetchApi: getAdminList,
  initialSearchForm: adminInitialSearchForm,
  defaultSort: 'id:desc',
  deleteApi: deleteAdmin,
  normalizeRows: false,
  tableRef: computed(() => listPageRef.value?.tableRef?.tableRef),
  beforeDelete: (row) => {
    if (isProtectedAdmin(row.id)) {
      ElMessage.warning(t('admin.protected_cannot_delete'))
      return false
    }
  }
})

const searchFields = computed(() => createAdminSearchFields(t, getStatusOptions))

const allTableColumns = computed(() => createAdminTableColumns(t))

const {
  tableColumns,
  visibleColumns,
  allColumns,
  defaultVisibleColumns,
  columnOrder,
  fixedColumns,
  handleColumnSettingConfirm
} = useColumnSetting('admin', allTableColumns, {
  adjacentPairs: [['department', 'position']]
})

const handleEdit = async (row) => {
  editId.value = row.id
  dialogVisible.value = true
  await new Promise((resolve) => setTimeout(resolve, 100))

  const uniqueRoleIds = row.roles
    ? [...new Set(compact(lodashMap(row.roles, (r) => r.id)))]
    : []

  adminFormRef.value?.setFormData({
    id: row.id,
    username: getField(row, 'username', ''),
    password: '',
    nickname: getField(row, 'nickname', ''),
    email: getField(row, 'email', ''),
    phone: getField(row, 'phone', ''),
    department_id: getField(row, 'department_id', null),
    position_id: getField(row, 'position_id', 0),
    role_ids: uniqueRoleIds,
    status: getField(row, 'status', 1),
    is_super_admin: row.is_super_admin === true
  })
}

const handleFormSuccess = () => {
  loadData()
}

const handleStatusChange = async (row, newStatus) => {
  if (isProtectedAdmin(row.id) && !newStatus) {
    ElMessage.warning(t('admin.protected_cannot_disable'))
    loadData()
    return
  }

  try {
    const statusValue = newStatus ? 1 : 0
    await updateAdmin(row.id, { status: statusValue })
    ElMessage.success(newStatus ? t('admin.enable_success') : t('admin.disable_success'))
    const admin = tableData.value.find((a) => a.id === row.id)
    if (admin) {
      admin.status = statusValue
    }
  } catch (error) {
    logger.error('Status change error:', error)
    loadData()
    if (!error.__handled) {
      ElMessage.error(error.response?.data?.message || error.message || t('common.operation_failed'))
    }
  }
}

const handleDelete = (row) => handleDeleteRow(row, loadData)

const handleResetPassword = async (row) => {
  try {
    const { value: password } = await ElMessageBox.prompt(
      t('admin.new_password'),
      t('admin.reset_password'),
      {
        confirmButtonText: t('common.confirm'),
        cancelButtonText: t('common.cancel'),
        inputType: 'password'
      }
    )
    const confirmCode = await promptSensitiveConfirm(t)
    await resetPassword(row.id, { password, confirm_code: confirmCode })
    ElMessage.success(t('admin.reset_password_success'))
  } catch (error) {
    if (error !== 'cancel') {
      logger.error('Reset password error:', error)
      ErrorHandler.handle(error, { silent: true })
    }
  }
}

const handleKickOut = async (row) => {
  try {
    await ElMessageBox.confirm(
      t('admin.kick_out_confirm', { username: row.username }),
      t('form.tip'),
      {
        confirmButtonText: t('common.confirm'),
        cancelButtonText: t('common.cancel'),
        type: 'warning'
      }
    )
    await kickOutUser(row.id)
    ElMessage.success(t('admin.kick_out_success'))
  } catch (error) {
    if (error !== 'cancel') {
      logger.error('Kick out error:', error)
      ErrorHandler.handle(error, { silent: true })
    }
  }
}

const getPrimaryActions = (row) => [
  {
    key: 'edit',
    label: t('common.edit'),
    type: 'primary',
    permission: 'admin.update',
    handler: handleEdit
  },
  {
    key: 'delete',
    label: t('common.delete'),
    type: 'danger',
    permission: 'admin.destroy',
    show: () => !isProtectedAdmin(row.id),
    handler: handleDelete
  }
]

const getMoreActions = (row) => [
  {
    key: 'resetPassword',
    command: 'resetPassword',
    label: t('admin.reset_password'),
    permission: 'admin.password',
    handler: handleResetPassword
  },
  {
    key: 'kickOut',
    command: 'kickOut',
    label: t('admin.kick_out'),
    permission: 'admin.kick_out',
    handler: handleKickOut
  },
  {
    key: 'unbindGoogleAuth',
    command: 'unbindGoogleAuth',
    label: t('admin.unbind_google_auth'),
    permission: 'admin.unbind_google_auth',
    divided: true,
    show: () => row.is_2fa_bound && !isProtectedAdmin(row.id),
    handler: handleUnbindGoogleAuth
  },
  {
    key: 'resetGoogleAuth',
    command: 'resetGoogleAuth',
    label: t('admin.reset_google_auth'),
    permission: 'admin.reset_google_auth',
    show: () => row.is_2fa_bound && !isProtectedAdmin(row.id),
    handler: handleResetGoogleAuth
  }
]

const handleAction = (command, row) => {
  switch (command) {
    case 'edit':
      handleEdit(row)
      break
    case 'delete':
      handleDelete(row)
      break
    case 'resetPassword':
      handleResetPassword(row)
      break
    case 'kickOut':
      handleKickOut(row)
      break
    case 'unbindGoogleAuth':
      handleUnbindGoogleAuth(row)
      break
    case 'resetGoogleAuth':
      handleResetGoogleAuth(row)
      break
  }
}

const handleUnbindGoogleAuth = async (row) => {
  try {
    const { value: code } = await ElMessageBox.prompt(
      t('admin.unbind_google_auth_confirm', { username: row.username }),
      t('admin.unbind_google_auth'),
      {
        confirmButtonText: t('common.confirm'),
        cancelButtonText: t('common.cancel'),
        inputPlaceholder: t('profile.enter_6_digit_code'),
        inputType: 'text',
        inputPattern: /^\d{6}$/,
        inputErrorMessage: t('profile.google_code_format')
      }
    )
    await unbindAdminGoogleAuth(row.id, { code })
    ElMessage.success(t('admin.unbind_google_auth_success'))
    loadData()
  } catch (error) {
    if (error !== 'cancel') {
      logger.error('Unbind google auth error:', error)
      if (!error?.__handled) {
        ElMessage.error(
          error.response?.data?.message || error.translatedMessage || error.message || t('common.operation_failed')
        )
      }
    }
  }
}

const handleResetGoogleAuth = async (row) => {
  try {
    await ElMessageBox.confirm(
      t('admin.reset_google_auth_confirm', { username: row.username }),
      t('admin.reset_google_auth'),
      {
        confirmButtonText: t('common.confirm'),
        cancelButtonText: t('common.cancel'),
        type: 'warning'
      }
    )
    await resetAdminGoogleAuth(row.id)
    ElMessage.success(t('admin.reset_google_auth_success'))
    loadData()
  } catch (error) {
    if (error !== 'cancel') {
      logger.error('Reset google auth error:', error)
      if (!error?.__handled) {
        ElMessage.error(
          error.response?.data?.message || error.translatedMessage || error.message || t('common.operation_failed')
        )
      }
    }
  }
}
</script>
