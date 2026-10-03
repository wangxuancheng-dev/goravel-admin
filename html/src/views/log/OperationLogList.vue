<template>
  <div class="operation-log-list">
    <ListPage
      ref="listPageRef"
      page-class="operation-log"
      :title="$t('menu.operation_log')"
      :show-add-button="false"
      :search-form="searchForm"
      :search-fields="searchFields"
      :initial-search-values="operationLogInitialSearchForm"
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

      <template #title="{ row }">
        {{ translateOperationTitle(row.title) }}
      </template>

      <template #request="{ row }">
        <div class="request-params-cell">
          <el-popover
            v-if="hasRequestParams(row.request)"
            placement="top"
            :width="600"
            trigger="hover"
            :title="$t('log.request_params')"
          >
            <template #reference>
              <div class="request-preview">
                <span class="preview-text">{{ getRequestPreview(row.request) }}</span>
              </div>
            </template>
            <div class="request-params-popover">
              <div class="request-params-header">
                <el-button
                  type="primary"
                  link
                  size="small"
                  @click="copyRequestParamsToClipboard(row.request)"
                >
                  <el-icon><DocumentCopy /></el-icon>
                  {{ $t('common.copy') }}
                </el-button>
              </div>
              <pre class="request-params-content">{{ formatRequestParamsFull(row.request) }}</pre>
            </div>
          </el-popover>
          <span v-else class="text-muted">-</span>
        </div>
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
        <el-descriptions-item :label="$t('log.method')">{{ logDetail.method }}</el-descriptions-item>
        <el-descriptions-item :label="$t('log.path')">{{ logDetail.path }}</el-descriptions-item>
        <el-descriptions-item :label="$t('log.ip')">{{ logDetail.ip }}</el-descriptions-item>
        <el-descriptions-item :label="$t('log.status_code')">{{ logDetail.status_code }}</el-descriptions-item>
        <el-descriptions-item :label="$t('log.operation_time')" :span="2">
          {{ logDetail.created_at }}
        </el-descriptions-item>
        <el-descriptions-item :label="$t('log.request_params')" :span="2">
          <div class="request-params-detail">
            <div class="request-params-header">
              <el-button type="primary" link size="small" @click="copyRequestParams">
                <el-icon><DocumentCopy /></el-icon>
                {{ $t('common.copy') }}
              </el-button>
            </div>
            <pre ref="requestParamsPre" class="request-params-content">
              {{ formatRequestParamsFull(logDetail.params || logDetail.request || {}) }}
            </pre>
          </div>
        </el-descriptions-item>
        <el-descriptions-item
          v-if="logDetail.changes?.length"
          :label="$t('log.changes')"
          :span="2"
        >
          <div class="changes-detail">
            <el-table :data="logDetail.changes" border size="small" class="changes-table">
              <el-table-column :label="$t('log.changes_field')" prop="field" width="180" />
              <el-table-column :label="$t('log.changes_old')">
                <template #default="{ row }">
                  <span class="changes-value changes-old">{{ formatChangeValue(row.old) }}</span>
                </template>
              </el-table-column>
              <el-table-column :label="$t('log.changes_new')">
                <template #default="{ row }">
                  <span class="changes-value changes-new">{{ formatChangeValue(row.new) }}</span>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </el-descriptions-item>
      </el-descriptions>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { DocumentCopy } from '@element-plus/icons-vue'
import ListPage from '@/components/ListPage.vue'
import TableActionButtons from '@/components/TableActionButtons.vue'
import { useStandardListPage } from '@/composables/useStandardListPage'
import { useColumnSetting } from '@/composables/useColumnSetting'
import { getMethodOptions } from '@/utils/fieldOptions'
import { validateTimeRange, OPERATION_LOG_MAX_TIME_RANGE_MONTHS } from '@/utils/timeRangeValidator'
import {
  getOperationLogList,
  getOperationLogDetail,
  deleteOperationLog,
  getOperationLogTitleOptions
} from '@/api/log'
import {
  createOperationLogInitialSearchForm,
  transformOperationLogRow,
  createOperationLogSearchFields,
  createOperationLogTableColumns,
  hasRequestParams,
  getRequestPreview,
  formatRequestParamsFull,
  formatChangeValue
} from './operationLog.config'
import { getOperationTitle } from '@/utils/operationTitle'

const { t, te, tm } = useI18n()
const listPageRef = ref(null)
const detailVisible = ref(false)
const logDetail = ref(null)
const requestParamsPre = ref(null)
const titleOptions = ref([])
const operationLogInitialSearchForm = createOperationLogInitialSearchForm()

const {
  pagination,
  tableData,
  loading,
  searchForm,
  loadData,
  handleSearch: baseHandleSearch,
  handleReset,
  handleSortChange,
  handleDelete: handleDeleteRow,
  getButtonState
} = useStandardListPage({
  fetchApi: getOperationLogList,
  initialSearchForm: operationLogInitialSearchForm,
  defaultSort: 'id:desc',
  deleteApi: deleteOperationLog,
  transformData: transformOperationLogRow,
  tableRef: computed(() => listPageRef.value?.tableRef?.tableRef)
})

const translateOperationTitle = (title) => getOperationTitle(t, te, tm, title)

const allTableColumns = computed(() => createOperationLogTableColumns(t))
const {
  tableColumns,
  visibleColumns,
  allColumns,
  defaultVisibleColumns,
  columnOrder,
  fixedColumns,
  handleColumnSettingConfirm
} = useColumnSetting('operation_log', allTableColumns)

const searchFields = computed(() =>
  createOperationLogSearchFields(t, getMethodOptions, titleOptions.value, translateOperationTitle)
)

const getCurrentTimeString = () => {
  const now = new Date()
  const pad = (n) => String(n).padStart(2, '0')
  return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())} ${pad(now.getHours())}:${pad(now.getMinutes())}:${pad(now.getSeconds())}`
}

const validateTimeRangeForSearch = () => {
  if (!searchForm.start_time) return true
  const endTime = searchForm.end_time || getCurrentTimeString()
  const validation = validateTimeRange(searchForm.start_time, endTime, OPERATION_LOG_MAX_TIME_RANGE_MONTHS)
  if (!validation.valid) {
    let errorMessage = validation.error
    if (validation.errorKey) {
      const translationKey = `log.${validation.errorKey}`
      errorMessage = validation.errorParams
        ? t(translationKey, validation.errorParams)
        : t(translationKey)
    }
    ElMessage.warning(errorMessage)
    return false
  }
  return true
}

const isInitialized = ref(false)
watch(
  () => [searchForm.start_time, searchForm.end_time],
  ([newStartTime, newEndTime], [oldStartTime, oldEndTime]) => {
    if (!isInitialized.value) {
      isInitialized.value = true
      return
    }
    if (newStartTime !== oldStartTime || newEndTime !== oldEndTime) {
      if (newStartTime) {
        const endTime = newEndTime || getCurrentTimeString()
        const validation = validateTimeRange(newStartTime, endTime, OPERATION_LOG_MAX_TIME_RANGE_MONTHS)
        if (!validation.valid) {
          let errorMessage = validation.error
          if (validation.errorKey) {
            const translationKey = `log.${validation.errorKey}`
            errorMessage = validation.errorParams
              ? t(translationKey, validation.errorParams)
              : t(translationKey)
          }
          ElMessage.warning(errorMessage)
        }
      }
    }
  }
)

const handleSearch = () => {
  if (!validateTimeRangeForSearch()) return
  baseHandleSearch()
}

const copyRequestParamsToClipboard = async (request) => {
  try {
    await navigator.clipboard.writeText(formatRequestParamsFull(request))
    ElMessage.success(t('common.copy_success') || '复制成功')
  } catch {
    ElMessage.error(t('common.copy_failed') || '复制失败')
  }
}

const copyRequestParams = async () => {
  if (!requestParamsPre.value) return
  const text = requestParamsPre.value.textContent
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success(t('common.copy_success') || '复制成功')
  } catch {
    const textarea = document.createElement('textarea')
    textarea.value = text
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)
    textarea.select()
    try {
      document.execCommand('copy')
      ElMessage.success(t('common.copy_success') || '复制成功')
    } catch {
      ElMessage.error(t('common.copy_failed') || '复制失败')
    }
    document.body.removeChild(textarea)
  }
}

const handleView = async (row) => {
  try {
    const res = await getOperationLogDetail(row.id)
    if (res.data) {
      const log = res.data.operation_log || res.data.log || res.data
      logDetail.value = transformOperationLogRow(log)
      detailVisible.value = true
    }
  } catch (error) {
    console.error('Load operation log detail error:', error)
  }
}

const handleDelete = (row) => handleDeleteRow(row, loadData)

const operationActions = computed(() => [
  {
    key: 'view',
    label: t('common.view'),
    type: 'primary',
    permission: 'operation_log.show',
    handler: handleView
  },
  {
    key: 'delete',
    label: t('common.delete'),
    type: 'danger',
    permission: 'operation_log.destroy',
    handler: handleDelete
  }
])

const loadTitleOptions = async () => {
  try {
    const res = await getOperationLogTitleOptions()
    const mergedSet = new Set()
    if (res.data?.titles && Array.isArray(res.data.titles)) {
      res.data.titles.forEach((title) => {
        if (title && typeof title === 'string') {
          const trimmed = title.trim()
          if (trimmed && trimmed !== 'operation.unknown' && !trimmed.startsWith('operation.')) {
            mergedSet.add(trimmed)
          }
        }
      })
    }
    titleOptions.value = Array.from(mergedSet).sort((a, b) => {
      const labelA = translateOperationTitle(a)
      const labelB = translateOperationTitle(b)
      const locale = t('common.locale') || navigator.language || 'zh-CN'
      return labelA.localeCompare(labelB, locale)
    })
  } catch (error) {
    console.error('Load title options error:', error)
    titleOptions.value = []
  }
}

onMounted(() => {
  loadTitleOptions()
})
</script>

<style scoped>
pre {
  margin: 0;
  padding: var(--space-sm);
  background: var(--bg-color-tertiary);
  border-radius: var(--border-radius-sm);
  overflow-x: auto;
}

.request-params-cell {
  display: flex;
  align-items: center;
  width: 100%;
}

.request-preview {
  width: 100%;
  cursor: pointer;
}

.preview-text {
  font-size: 12px;
  color: var(--text-color-regular);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', 'Consolas', 'source-code-pro', monospace;
  display: block;
}

.text-muted {
  color: var(--text-color-secondary);
}

.request-params-popover {
  max-height: 500px;
  overflow: hidden;
}

.request-params-header {
  display: flex;
  justify-content: flex-end;
  margin-bottom: var(--space-xs);
  padding-bottom: var(--space-xs);
  border-bottom: 1px solid var(--border-color-lighter);
}

.request-params-content {
  margin: 0;
  padding: var(--space-sm);
  background: var(--bg-color-tertiary);
  border-radius: var(--border-radius-sm);
  overflow-x: auto;
  font-size: 12px;
  line-height: 1.6;
  max-height: 400px;
  overflow-y: auto;
  word-break: break-all;
  white-space: pre-wrap;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', 'Consolas', 'source-code-pro', monospace;
}

.request-params-detail {
  width: 100%;
}

.changes-detail,
.changes-table {
  width: 100%;
}

.changes-value {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', 'Consolas', 'source-code-pro', monospace;
  font-size: 12px;
  word-break: break-all;
  white-space: pre-wrap;
}

.changes-old {
  color: var(--el-color-danger);
}

.changes-new {
  color: var(--el-color-success);
}

html.dark pre,
html.dark .request-params-content {
  background: var(--el-bg-color) !important;
  color: var(--el-text-color-regular) !important;
  border: 1px solid var(--el-border-color);
}

html.dark .request-params-header {
  border-bottom-color: var(--el-border-color) !important;
}

html.dark .preview-text {
  color: var(--el-text-color-regular) !important;
}

html.dark .text-muted {
  color: var(--el-text-color-secondary) !important;
}
</style>
