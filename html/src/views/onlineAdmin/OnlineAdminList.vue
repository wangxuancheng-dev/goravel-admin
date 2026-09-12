<template>
  <ListPage
    ref="listPageRef"
    page-class="online-admin"
    :title="t('menu.online_admin')"
    :show-add-button="false"
    :search-form="searchForm"
    :search-fields="searchFields"
    :initial-search-values="onlineAdminInitialSearchForm"
    i18n-prefix="online_admin"
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
    @search="handleSearch"
    @reset="handleReset"
    @refresh="loadData"
    @page-change="loadData"
    @sort-change="handleSortChange"
    @selection-change="handleSelectionChange"
  >
    <template #toolbar-left>
      <el-button
        type="danger"
        :disabled="selectedRows.length === 0 || getButtonState('admin.kick_out').disabled"
        @click="handleBatchKickOut"
      >
        <el-icon><Delete /></el-icon>
        {{ t('online_admin.batch_kick_out') }}
      </el-button>
    </template>

    <template #avatar="{ row }">
      <el-avatar :size="32" :src="row.avatar">
        {{ row.nickname?.charAt(0) || row.username?.charAt(0) || 'U' }}
      </el-avatar>
    </template>

    <template #last_active="{ row }">
      {{ formatOnlineTime(row.last_active) }}
    </template>

    <template #operation="{ row }">
      <el-button
        type="danger"
        link
        :disabled="getButtonState('admin.kick_out').disabled"
        @click="handleKickOut(row)"
      >
        {{ t('online_admin.kick_out') }}
      </el-button>
    </template>
  </ListPage>
</template>

<script setup>
import { ref, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete } from '@element-plus/icons-vue'
import { useI18n } from 'vue-i18n'
import ListPage from '@/components/ListPage.vue'
import { useStandardListPage } from '@/composables/useStandardListPage'
import { useColumnSetting } from '@/composables/useColumnSetting'
import {
  getOnlineAdminList,
  kickOutOnlineAdmin,
  batchKickOutOnlineAdmins
} from '@/api/onlineAdmin'
import { promptSensitiveConfirm } from '@/utils/sensitiveConfirm'
import {
  onlineAdminInitialSearchForm,
  createOnlineAdminSearchFields,
  createOnlineAdminTableColumns,
  formatOnlineTime
} from './onlineAdmin.config'

const { t } = useI18n()
const listPageRef = ref(null)

const allTableColumns = computed(() => createOnlineAdminTableColumns(t))

const {
  tableColumns,
  visibleColumns,
  allColumns,
  defaultVisibleColumns,
  columnOrder,
  fixedColumns,
  handleColumnSettingConfirm
} = useColumnSetting('online_admin', allTableColumns)

const searchFields = computed(() => createOnlineAdminSearchFields(t))

const {
  pagination,
  tableData,
  loading,
  searchForm,
  loadData,
  handleSearch,
  handleReset,
  handleSortChange,
  selectedRows,
  handleSelectionChange,
  getButtonState
} = useStandardListPage({
  fetchApi: getOnlineAdminList,
  initialSearchForm: onlineAdminInitialSearchForm,
  fieldMapping: { last_active: 'last_used_at' },
  defaultSort: 'last_used_at:desc',
  normalizeRows: false,
  tableRef: computed(() => listPageRef.value?.tableRef?.tableRef)
})

const handleKickOut = async (row) => {
  try {
    await ElMessageBox.confirm(
      t('online_admin.kick_out_confirm', { username: row.username }),
      t('common.confirm'),
      { type: 'warning' }
    )
    const confirmCode = await promptSensitiveConfirm(t)
    await kickOutOnlineAdmin(row.id, { confirm_code: confirmCode })
    ElMessage.success(t('online_admin.kick_out_success'))
    loadData()
  } catch {
    // cancelled
  }
}

const handleBatchKickOut = async () => {
  if (!selectedRows.value.length) return
  try {
    await ElMessageBox.confirm(
      t('online_admin.batch_kick_out_confirm', { count: selectedRows.value.length }),
      t('common.confirm'),
      { type: 'warning' }
    )
    const confirmCode = await promptSensitiveConfirm(t)
    await batchKickOutOnlineAdmins(selectedRows.value.map((item) => item.id), {
      confirm_code: confirmCode
    })
    ElMessage.success(t('online_admin.batch_kick_out_success'))
    handleSelectionChange([])
    loadData()
  } catch {
    // cancelled
  }
}
</script>
