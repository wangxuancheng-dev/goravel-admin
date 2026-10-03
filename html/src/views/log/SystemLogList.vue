<template>
  <div class="system-log-list">
    <ListPage
      ref="listPageRef"
      page-class="system-log"
      :title="$t('menu.system_log')"
      :show-add-button="false"
      :search-form="searchForm"
      :search-fields="searchFields"
      :initial-search-values="systemLogInitialSearchForm"
      i18n-prefix="log"
      :table-data="tableData"
      :loading="loading"
      :table-columns="tableColumns"
      :pagination="pagination"
      show-toolbar
      @search="handleSearch"
      @reset="handleReset"
      @refresh="loadData"
      @page-change="loadData"
      @sort-change="handleSortChange"
      @selection-change="onTableSelectionChange"
    >
      <template #toolbar-left>
        <el-button
          type="danger"
          :disabled="!selectedRows?.length || getButtonState('system_log.batch_delete').disabled"
          @click="handleBatchDelete"
        >
          <el-icon><Delete /></el-icon>
          {{ $t('common.delete_selected') }} ({{ selectedRows?.length || 0 }})
        </el-button>
        <el-button
          :disabled="!selectedRows?.length"
          @click="handleClearSelection"
        >
          {{ $t('common.reset') }}
        </el-button>
      </template>

      <template #level="{ row }">
        <el-tag :type="getSystemLogLevelType(row.level)">
          {{ getSystemLogLevelLabel(t, row.level) }}
        </el-tag>
      </template>

      <template #context="{ row }">
        <el-tooltip
          v-if="row.context"
          :content="formatSystemLogContext(row.context)"
          placement="top"
          effect="dark"
        >
          <div class="context-preview">
            {{ formatSystemLogContextPreview(row.context) }}
          </div>
        </el-tooltip>
        <span v-else>-</span>
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
        <el-descriptions-item :label="$t('log.level')">
          <el-tag :type="getSystemLogLevelType(logDetail.level)">
            {{ getSystemLogLevelLabel(t, logDetail.level) }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item :label="$t('log.module')">
          {{ getModuleLabel(logDetail.module) }}
        </el-descriptions-item>
        <el-descriptions-item :label="$t('log.trace_id')" :span="2">
          {{ logDetail.trace_id || '-' }}
        </el-descriptions-item>
        <el-descriptions-item :label="$t('log.message')" :span="2">
          {{ logDetail.message }}
        </el-descriptions-item>
        <el-descriptions-item :label="$t('log.context')" :span="2">
          <pre v-if="logDetail.context">{{ formatSystemLogContext(logDetail.context) }}</pre>
          <span v-else>-</span>
        </el-descriptions-item>
        <el-descriptions-item :label="$t('log.time')" :span="2">
          {{ logDetail.created_at }}
        </el-descriptions-item>
      </el-descriptions>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Delete } from '@element-plus/icons-vue'
import ListPage from '@/components/ListPage.vue'
import TableActionButtons from '@/components/TableActionButtons.vue'
import { useStandardListPage } from '@/composables/useStandardListPage'
import {
  getSystemLogList,
  getSystemLogModuleOptions,
  getSystemLogDetail,
  deleteSystemLog,
  batchDeleteSystemLogs
} from '@/api/log'
import {
  systemLogInitialSearchForm,
  transformSystemLogRow,
  createSystemLogSearchFields,
  createSystemLogTableColumns,
  getSystemLogLevelType,
  getSystemLogLevelLabel,
  getSystemLogModuleLabel,
  formatSystemLogContext,
  formatSystemLogContextPreview
} from './systemLog.config'

const { t, te } = useI18n()
const listPageRef = ref(null)
const detailVisible = ref(false)
const logDetail = ref(null)
const selectedRows = ref([])
const selectedIds = ref(new Set())
const moduleOptions = ref([])

const getTable = () => listPageRef.value?.tableRef?.tableRef

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
  handleBatchDelete: handleBatchDeleteRows,
  getButtonState
} = useStandardListPage({
  fetchApi: getSystemLogList,
  initialSearchForm: systemLogInitialSearchForm,
  defaultSort: 'id:desc',
  deleteApi: deleteSystemLog,
  batchDeleteApi: batchDeleteSystemLogs,
  transformData: transformSystemLogRow,
  tableRef: computed(() => getTable()),
  onSearch: () => {
    selectedRows.value = []
    selectedIds.value.clear()
  },
  onReset: () => {
    selectedRows.value = []
    selectedIds.value.clear()
  },
  onLoadSuccess: () => {
    nextTick(() => {
      const table = getTable()
      if (table && selectedIds.value.size > 0) {
        tableData.value
          .filter((row) => selectedIds.value.has(row.id))
          .forEach((row) => table.setCheckboxRow(row, true))
        selectedRows.value = table.getCheckboxRecords() || []
      }
    })
  }
})

const getModuleLabel = (module) => getSystemLogModuleLabel(t, te, module)
const searchFields = computed(() => createSystemLogSearchFields(t, moduleOptions.value))
const tableColumns = computed(() => createSystemLogTableColumns(t, getModuleLabel))

const loadModuleOptions = async () => {
  try {
    const res = await getSystemLogModuleOptions()
    const modules = Array.isArray(res?.data?.modules) ? res.data.modules : []
    moduleOptions.value = modules.map((module) => ({
      label: getModuleLabel(module),
      value: module
    }))
  } catch (error) {
    console.error('Load system log module options error:', error)
    moduleOptions.value = []
  }
}

const handleView = async (row) => {
  try {
    const res = await getSystemLogDetail(row.id)
    if (res?.data) {
      const log = res.data.system_log || res.data.log || res.data
      if (log) {
        logDetail.value = transformSystemLogRow(log)
        detailVisible.value = true
      }
    }
  } catch (error) {
    console.error('Load system log detail error:', error)
    ElMessage.error(error.response?.data?.message || error.message || t('error.default'))
  }
}

const handleDelete = (row) => handleDeleteRow(row, loadData)

const operationActions = computed(() => [
  {
    key: 'view',
    label: t('common.view'),
    type: 'primary',
    permission: 'system_log.show',
    handler: handleView
  },
  {
    key: 'delete',
    label: t('common.delete'),
    type: 'danger',
    permission: 'system_log.destroy',
    handler: handleDelete
  }
])

const onTableSelectionChange = () => {
  const table = getTable()
  if (!table) return

  const currentSelected = table.getCheckboxRecords() || []
  selectedRows.value = currentSelected

  tableData.value.forEach((row) => selectedIds.value.delete(row.id))
  currentSelected.forEach((row) => selectedIds.value.add(row.id))
}

const handleBatchDelete = () => {
  handleBatchDeleteRows(selectedRows.value, () => {
    selectedRows.value = []
    selectedIds.value.clear()
    loadData()
  })
}

const handleClearSelection = () => {
  selectedRows.value = []
  selectedIds.value.clear()
  const table = getTable()
  if (!table) return

  table.clearCheckboxRow?.()
  table.setAllCheckboxRow?.(false)
  table.clearCheckboxReserve?.()
  tableData.value.forEach((row) => table.setCheckboxRow?.(row, false))
}

onMounted(() => {
  loadModuleOptions()
})
</script>

<style scoped>
pre {
  margin: 0;
  padding: var(--space-sm);
  background: var(--bg-color-tertiary);
  border-radius: var(--border-radius-sm);
  overflow-x: auto;
  max-height: 400px;
  overflow-y: auto;
}

html.dark pre {
  background: var(--el-bg-color) !important;
  color: var(--el-text-color-regular) !important;
  border: 1px solid var(--el-border-color);
}

.context-preview {
  max-width: 200px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  cursor: pointer;
  color: var(--el-color-primary);
}
</style>
