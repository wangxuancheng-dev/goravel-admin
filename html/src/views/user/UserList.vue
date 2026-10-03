<template>
  <ListPage
    ref="listPageRef"
    page-class="user"
    :title="$t('menu.user')"
    :add-button-text="$t('user.add_user')"
    :add-button-disabled="getButtonState('user.store').disabled"
    :search-form="searchForm"
    :search-fields="searchFields"
    :initial-search-values="userInitialSearchForm"
    i18n-prefix="user"
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
        :disabled="getButtonState('user.export').disabled || isExporting"
        :loading="isExporting"
        @click="handleExport"
      >
        <el-icon><Download /></el-icon>
        {{ $t('common.export') }}
      </el-button>
    </template>

    <template #status="{ row }">
      <el-switch
        :model-value="rowStatus(row) === 1"
        :disabled="getButtonState('user.update').disabled"
        @change="(val) => handleStatusChange(row, val)"
      />
    </template>

    <template #balance="{ row }">
      <span style="color: var(--el-color-primary); font-weight: bold;">
        {{ formatBalance(row.balance || 0, row.currency) }}
      </span>
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
      <UserForm
        ref="userFormRef"
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
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Download } from '@element-plus/icons-vue'
import ListPage from '@/components/ListPage.vue'
import TableActionButtons from '@/components/TableActionButtons.vue'
import UserForm from './UserForm.vue'
import { useStandardListPage } from '@/composables/useStandardListPage'
import { useColumnSetting } from '@/composables/useColumnSetting'
import { rowStatus } from '@/utils/listPageHelpers'
import { getStatusOptions } from '@/utils/fieldOptions'
import {
  getUserList,
  deleteUser,
  updateUser,
  resetPassword,
  updateBalance,
  exportUsers
} from '@/api/user'
import logger from '@/utils/logger'
import ErrorHandler from '@/utils/errorHandler'
import {
  userInitialSearchForm,
  createUserSearchFields,
  createUserTableColumns,
  formatBalance
} from './user.config'

const { t } = useI18n()
const router = useRouter()
const listPageRef = ref(null)
const userFormRef = ref(null)
const isExporting = ref(false)

const allTableColumns = computed(() => createUserTableColumns(t))

const {
  tableColumns,
  visibleColumns,
  allColumns,
  defaultVisibleColumns,
  columnOrder,
  fixedColumns,
  handleColumnSettingConfirm
} = useColumnSetting('user', allTableColumns)

const searchFields = computed(() => createUserSearchFields(t, getStatusOptions))

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
  handleEdit,
  handleClose,
  handleDelete: handleDeleteRow,
  getButtonState
} = useStandardListPage({
  fetchApi: getUserList,
  initialSearchForm: userInitialSearchForm,
  defaultSort: 'id:desc',
  deleteApi: deleteUser,
  tableRef: computed(() => listPageRef.value?.tableRef?.tableRef)
})

const getPrimaryActions = () => [
  { key: 'edit', label: t('common.edit'), permission: 'user.update' },
  { key: 'updateBalance', label: t('user.update_balance'), permission: 'user.update_balance', type: 'warning' }
]

const getMoreActions = () => [
  { key: 'resetPassword', command: 'resetPassword', label: t('user.reset_password'), permission: 'user.password' },
  { key: 'balanceLogs', command: 'balanceLogs', label: t('user.balance_logs'), permission: 'user_balance_log.index', divided: true },
  { key: 'delete', command: 'delete', label: t('common.delete'), permission: 'user.destroy', type: 'danger' }
]

const handleAction = async (command, row) => {
  switch (command) {
    case 'edit':
      handleEdit(row)
      break
    case 'updateBalance':
      await handleUpdateBalance(row)
      break
    case 'resetPassword':
      await handleResetPassword(row)
      break
    case 'balanceLogs':
      handleBalanceLogs(row)
      break
    case 'delete':
      await handleDelete(row)
      break
    default:
      logger.warn('Unknown action command:', command)
  }
}

const handleUpdateBalance = async (row) => {
  try {
    const { value: amountStr } = await ElMessageBox.prompt(
      t('user.update_balance_prompt'),
      t('user.update_balance'),
      {
        confirmButtonText: t('common.confirm'),
        cancelButtonText: t('common.cancel'),
        inputType: 'number',
        inputValidator: (value) => (!value || Number(value) === 0 ? t('user.amount_required') : true)
      }
    )

    const amount = Number(amountStr)
    if (amount === 0) {
      ElMessage.error(t('user.amount_cannot_be_zero'))
      return
    }

    const typeOptions = [
      { value: 'income', label: t('user.balance_income') },
      { value: 'expense', label: t('user.balance_expense') },
      { value: 'refund', label: t('user.balance_refund') }
    ]

    const { value: typeIndex } = await ElMessageBox.prompt(
      `${t('user.select_change_type_prompt')}\n\n${typeOptions.map((opt, idx) => `${idx + 1}. ${opt.label}`).join('\n')}`,
      t('user.select_change_type'),
      {
        confirmButtonText: t('common.confirm'),
        cancelButtonText: t('common.cancel'),
        inputType: 'number',
        inputPlaceholder: '1-3',
        inputValidator: (value) => {
          const num = Number(value)
          return !value || num < 1 || num > 3 ? t('user.invalid_selection') : true
        }
      }
    )

    const selectedIndex = Number(typeIndex) - 1
    if (selectedIndex < 0 || selectedIndex >= typeOptions.length) return

    await ElMessageBox.confirm(
      `${t('user.balance_update_confirm')}\n${t('user.amount')}: ${Math.abs(amount).toFixed(2)}\n${t('user.change_type')}: ${typeOptions[selectedIndex].label}`,
      t('common.confirm'),
      { type: 'warning' }
    )

    await updateBalance(row.id, {
      amount: Math.abs(amount),
      type: typeOptions[selectedIndex].value,
      source: 'manual',
      description: t('user.manual_balance_adjustment')
    })
    ElMessage.success(t('user.balance_update_success'))
    loadData()
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') {
      ErrorHandler.handle(error)
    }
  }
}

const handleResetPassword = async (row) => {
  try {
    const { value: password } = await ElMessageBox.prompt(t('user.new_password'), t('user.reset_password'), {
      confirmButtonText: t('common.confirm'),
      cancelButtonText: t('common.cancel'),
      inputType: 'password',
      inputValidator: (value) => (!value || value.length < 6 ? t('user.password_min_length') : true)
    })
    await resetPassword(row.id, { password })
    ElMessage.success(t('user.reset_password_success'))
  } catch (error) {
    if (error !== 'cancel' && error !== 'close') {
      ErrorHandler.handle(error, { silent: true })
    }
  }
}

const handleBalanceLogs = (row) => {
  router.push({ path: '/user-balance-logs', query: { user_id: row.id } })
}

const handleDelete = async (row) => {
  await handleDeleteRow(row, loadData)
}

const handleStatusChange = async (row, val) => {
  try {
    await updateUser(row.id, { status: val ? 1 : 0 })
    ElMessage.success(t('common.update_success'))
    loadData()
  } catch (error) {
    ErrorHandler.handle(error)
  }
}

const handleFormSuccess = () => {
  handleClose()
  loadData()
}

const handleExport = async () => {
  if (isExporting.value) return
  isExporting.value = true
  try {
    const response = await exportUsers({ ...searchForm, order_by: 'id:desc' })
    const exportId = response.data?.data?.export_id || response.data?.export_id
    if (!exportId) {
      ElMessage.error(t('export.failed'))
      return
    }
    ElMessage.success(t('export.task_submitted'))
    router.push('/exports')
  } catch (error) {
    if (error.response?.status === 429) {
      ElMessage.warning(t('common.already_queued'))
    } else if (!error.__handled) {
      ErrorHandler.handle(error, { silent: true })
    }
  } finally {
    isExporting.value = false
  }
}
</script>
