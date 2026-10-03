<template>
  <div class="login-log-list">
    <ListPage
    ref="listPageRef"
    page-class="login-log"
    :title="$t('menu.login_log')"
    :show-add-button="false"
    :search-form="searchForm"
    :search-fields="searchFields"
    :initial-search-values="loginLogInitialSearchForm"
    i18n-prefix="log"
    :table-data="tableData"
    :loading="loading"
    :table-columns="tableColumns"
    :table-key="`table-${tableColumns.length}-${JSON.stringify(columnOrder)}`"
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
  >
    <template #admin="{ row }">
      {{ row.admin?.username || '-' }}
    </template>

    <template #status="{ row }">
      <el-tag :type="(row.status ?? 1) === 1 ? 'success' : 'danger'">
        {{ (row.status ?? 1) === 1 ? $t('log.success') : $t('log.failed') }}
      </el-tag>
    </template>

    <template #message="{ row }">
      {{ translateLoginMessage(row.message || '') }}
    </template>

    <template #operation="{ row }">
      <TableActionButtons
        :row="row"
        :primary-actions="operationActions"
        :get-button-state="getButtonState"
      />
    </template>
  </ListPage>

  <el-dialog v-model="detailVisible" :title="$t('log.detail')" width="1100px">
    <el-descriptions v-if="logDetail" :column="2" border>
      <el-descriptions-item :label="$t('table.id')">{{ logDetail.id }}</el-descriptions-item>
      <el-descriptions-item :label="$t('log.admin')">{{ logDetail.admin?.username }}</el-descriptions-item>
      <el-descriptions-item :label="$t('log.ip')">{{ logDetail.ip }}</el-descriptions-item>
      <el-descriptions-item :label="$t('log.location')">{{ logDetail.location || '-' }}</el-descriptions-item>
      <el-descriptions-item :label="$t('table.status')">
        <el-tag :type="logDetail.status === 1 ? 'success' : 'danger'">
          {{ logDetail.status === 1 ? $t('log.success') : $t('log.failed') }}
        </el-tag>
      </el-descriptions-item>
      <el-descriptions-item :label="$t('log.user_agent')">{{ logDetail.user_agent }}</el-descriptions-item>
      <el-descriptions-item :label="$t('log.login_time')">{{ logDetail.created_at }}</el-descriptions-item>
      <el-descriptions-item :label="$t('log.message')" :span="2">
        {{ translateLoginMessage(logDetail.message || '') }}
      </el-descriptions-item>
      <el-descriptions-item :label="$t('log.request')" :span="2">
        <pre v-if="logDetail.request" class="request-preview-content">{{ formatRequestPreview(logDetail.request) }}</pre>
        <span v-else>-</span>
      </el-descriptions-item>
    </el-descriptions>
  </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Delete, View } from '@element-plus/icons-vue'
import ListPage from '@/components/ListPage.vue'
import TableActionButtons from '@/components/TableActionButtons.vue'
import { useStandardListPage } from '@/composables/useStandardListPage'
import { useColumnSetting } from '@/composables/useColumnSetting'
import { getLoginLogList, getLoginLogDetail, deleteLoginLog } from '@/api/log'
import {
  loginLogInitialSearchForm,
  transformLoginLogRow,
  createLoginLogSearchFields,
  createLoginLogTableColumns,
  formatRequestPreview
} from './loginLog.config'

const { t } = useI18n()
const listPageRef = ref(null)
const detailVisible = ref(false)
const logDetail = ref(null)

const {
  pagination,
  tableData,
  loading,
  searchForm,
  loadData,
  handleSearch,
  handleReset,
  handleSortChange,
  handleDelete: handleDeleteRow,
  getButtonState
} = useStandardListPage({
  fetchApi: getLoginLogList,
  initialSearchForm: loginLogInitialSearchForm,
  defaultSort: 'id:desc',
  deleteApi: deleteLoginLog,
  transformData: transformLoginLogRow,
  tableRef: computed(() => listPageRef.value?.tableRef?.tableRef)
})

const searchFields = computed(() => createLoginLogSearchFields(t))

const allTableColumns = computed(() => createLoginLogTableColumns(t))

const {
  tableColumns,
  visibleColumns,
  allColumns,
  defaultVisibleColumns,
  columnOrder,
  fixedColumns,
  handleColumnSettingConfirm
} = useColumnSetting('login_log', allTableColumns)

const handleView = async (row) => {
  try {
    const res = await getLoginLogDetail(row.id)
    if (res.data) {
      const log = res.data.login_log || res.data.log || res.data
      logDetail.value = transformLoginLogRow(log)
      detailVisible.value = true
    }
  } catch (error) {
    console.error('Load login log detail error:', error)
  }
}

const handleDelete = (row) => handleDeleteRow(row, loadData)

const operationActions = computed(() => [
  {
    key: 'view',
    label: t('common.view'),
    type: 'primary',
    permission: 'login_log.show',
    icon: View,
    handler: handleView
  },
  {
    key: 'delete',
    label: t('common.delete'),
    type: 'danger',
    permission: 'login_log.destroy',
    icon: Delete,
    handler: handleDelete
  }
])

const translateLoginMessage = (messageKey) => {
  if (!messageKey) return '-'
  const translation = t(`log.${messageKey}`, null)
  return translation !== `log.${messageKey}` ? translation : messageKey
}
</script>

<style scoped>
.request-preview-content {
  white-space: pre-wrap;
  word-wrap: break-word;
  max-height: 300px;
  overflow-y: auto;
  background: var(--bg-color-tertiary);
  padding: var(--space-sm);
  border-radius: var(--border-radius-sm);
  margin: 0;
}

html.dark .request-preview-content {
  background: var(--el-bg-color) !important;
  color: var(--el-text-color-regular) !important;
  border: 1px solid var(--el-border-color);
}
</style>
