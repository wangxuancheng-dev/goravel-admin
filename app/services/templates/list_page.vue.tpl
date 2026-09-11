<template>
  <ListPage
    ref="listPageRef"
    page-class="<<.ModuleName>>"
    :title="$t('menu.<<.ModuleName>>')"
    <<if .HasCreate>>
    :add-button-text="$t('common.add')"
    :add-button-disabled="getButtonState('<<.ModuleName>>.store').disabled"
    <<else>>
    :show-add-button="false"
    <<end>>
    :search-form="searchForm"
    :search-fields="searchFields"
    :initial-search-values="<<.ModuleNameCamel>>InitialSearchForm"
    i18n-prefix="<<.ModuleName>>"
    :table-data="tableData"
    :loading="loading"
    :table-columns="tableColumns"
    :pagination="pagination"
    :show-toolbar="<<.ShowToolbar>>"
    @add="handleAdd"
    @search="handleSearch"
    @reset="handleReset"
    @refresh="loadData"
    @page-change="loadData"
    @sort-change="handleSortChange"
    <<if .EnableBatchActions>>
    @selection-change="handleSelectionChange"
    <<end>>
  >
    <<if .EnableBatchActions>>
    <template #toolbar-left>
      <<if .HasDelete>>
      <el-button
        v-if="hasSelection"
        type="danger"
        :disabled="getButtonState('<<.ModuleName>>.destroy').disabled"
        @click="handleBatchDelete"
      >
        {{ `${$t('common.batch_delete')} (${selectedIds.length})` }}
      </el-button>
      <<end>>
    </template>
    <<end>>

    <<if .HasExport>>
    <template #extra-buttons>
      <el-button
        type="success"
        :disabled="getButtonState('<<.ModuleName>>.export').disabled || isExporting"
        :loading="isExporting"
        @click="handleExport"
      >
        {{ $t('common.export') }}
      </el-button>
    </template>
    <<end>>

    <<range .ListFields>>
    <<- if and .ShowInList (eq .Name "status") (eq .FormType "switch")>>
    <template #status="{ row }">
      <el-switch
        :model-value="Number(row.status ?? 1) === 1"
        :disabled="getButtonState('<<$.ModuleName>>.update').disabled"
        @change="(val) => handleStatusChange(row, val)"
      />
    </template>
    <<- else if and .ShowInList (eq .FormType "image-upload")>>
    <template #<<.Name>>="{ row }">
      <el-image
        v-if="row.<<.Name>>"
        :src="row.<<.Name>>"
        :preview-src-list="[row.<<.Name>>]"
        fit="cover"
        style="width: 60px; height: 60px; border-radius: 4px; cursor: pointer;"
        lazy
      />
      <span v-else>-</span>
    </template>
    <<- else if and .ShowInList (or (eq .FormType "editor") (eq .FormType "markdown"))>>
    <template #<<.Name>>="{ row }">
      <div class="text-truncate" :title="extractTextFromMarkdown(row.<<.Name>>)">
        {{ extractTextFromMarkdown(row.<<.Name>>).slice(0, 120) || '-' }}
      </div>
    </template>
    <<- else if and .ShowInList .Relation>>
    <template #<<.Name>>="{ row }">
      {{ get<<.Relation.JsonName>>DisplayName(row.<<.Relation.JsonName>> || row.<<.Name>>) }}
    </template>
    <<- end>>
    <<- end>>

    <template #operation="{ row }">
      <TableActionButtons
        :row="row"
        :primary-actions="operationActions"
        :get-button-state="getButtonState"
      />
    </template>

    <template #form>
      <<printf "<%s" .ModelName>>Form
        v-model="dialogVisible"
        :edit-id="editId"
        @success="handleFormSuccess"
      />
    </template>
  </ListPage>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import ListPage from '@/components/ListPage.vue'
import TableActionButtons from '@/components/TableActionButtons.vue'
import <<.ModelName>>Form from './<<.ModelName>>Form.vue'
import { useStandardListPage } from '@/composables/useStandardListPage'
import { createCrudActions } from '@/utils/listPageHelpers'
<<if .HasRichTextInList>>
import { extractTextFromMarkdown } from '@/utils/markdown'
<<end>>
<<if .HasExport>>
import { export<<.ModelName>> } from '@/api/<<.ModuleName>>'
<<end>>
import {
  get<<.ModelName>>List,
  <<if .HasDelete>>delete<<.ModelName>>,<<end>>
  <<if .HasEdit>>update<<.ModelName>>,<<end>>
} from '@/api/<<.ModuleName>>'
import logger from '@/utils/logger'
import ErrorHandler from '@/utils/errorHandler'
import {
  <<.ModuleNameCamel>>InitialSearchForm,
  build<<.ModelName>>ListParams,
  create<<.ModelName>>SearchFields,
  create<<.ModelName>>TableColumns,
<<range .ListFields>>
<<- if and .ShowInList .Relation>>
  get<<.Relation.JsonName>>DisplayName,
<<- end>>
<<- end>>
} from './<<.ModuleNameCamel>>.config'

const { t } = useI18n()
const router = useRouter()
const listPageRef = ref(null)
<<if .HasExport>>
const isExporting = ref(false)
<<end>>

const {
  pagination,
  tableData,
  loading,
  searchForm,
  selectedIds,
  dialogVisible,
  editId,
  loadData,
  handleSearch,
  handleReset,
  handleSortChange,
  handleSelectionChange,
  handleAdd,
  handleEdit,
  handleFormSuccess,
  handleDelete,
  getButtonState
} = useStandardListPage({
  fetchApi: get<<.ModelName>>List,
  initialSearchForm: <<.ModuleNameCamel>>InitialSearchForm,
  buildParams: build<<.ModelName>>ListParams,
  defaultSort: 'id:desc',
  <<if .HasDelete>>deleteApi: delete<<.ModelName>>,<<end>>
  tableRef: computed(() => listPageRef.value?.tableRef?.tableRef),
  normalizeRows: false
})

const hasSelection = computed(() => selectedIds.value.length > 0)
const searchFields = computed(() => create<<.ModelName>>SearchFields(t))
const tableColumns = computed(() =>
  create<<.ModelName>>TableColumns(t, { enableBatchActions: <<if .EnableBatchActions>>true<<else>>false<<end>> })
)

<<if .HasEdit>>
<<range .ListFields>>
<<- if and .ShowInList (eq .Name "status") (eq .FormType "switch")>>
const handleStatusChange = async (row, newStatus) => {
  try {
    const statusValue = newStatus ? 1 : 0
    await update<<$.ModelName>>(row.id, { status: statusValue })
    ElMessage.success(newStatus ? t('common.enabled') : t('common.disabled'))
    const item = tableData.value.find((item) => item.id === row.id)
    if (item) item.status = statusValue
  } catch (error) {
    logger.error('Status change error:', error)
    loadData()
    if (!error.__handled) {
      ElMessage.error(error.message || t('common.operation_failed'))
    }
  }
}
<<- end>>
<<- end>>
<<end>>

const operationActions = computed(() =>
  createCrudActions(t, '<<.ModuleName>>', {
    <<if .HasEdit>>onEdit: handleEdit,<<end>>
    <<if .HasDelete>>onDelete: handleDelete<<end>>
  })
)

<<if .HasDelete>>
const handleBatchDelete = async () => {
  if (!selectedIds.value.length) return
  try {
    await ElMessageBox.confirm(
      t('common.batch_delete_confirm', { count: selectedIds.value.length }),
      t('common.warning'),
      { type: 'warning' }
    )
    await Promise.all(selectedIds.value.map((id) => delete<<.ModelName>>(id)))
    ElMessage.success(t('common.operation_success'))
    await loadData()
  } catch (error) {
    if (error === 'cancel' || error === 'close') return
    logger.error('Batch delete error:', error)
  }
}
<<end>>

<<if .HasExport>>
const handleExport = async () => {
  if (isExporting.value) return
  isExporting.value = true
  try {
    const response = await export<<.ModelName>>(searchForm)
<<- if .ExportAsync>>
    const exportId = response?.data?.export_id || response?.data?.id
    ElMessage.success(exportId ? t('export.task_submitted') : t('common.operation_success'))
    router.push('/exports')
<<- else>>
    const fileUrl = response?.data?.file_url
    if (fileUrl) {
      window.open(fileUrl, '_blank')
      ElMessage.success(t('export.success'))
    } else {
      ElMessage.success(t('export.success'))
      router.push('/exports')
    }
<<- end>>
  } catch (error) {
    logger.error('Export error:', error)
    if (error.response?.status === 429) {
      ElMessage.warning(t('common.already_queued'))
    } else if (!error.__handled) {
      ElMessage.error(t('export.failed'))
      ErrorHandler.handle(error, { silent: true })
    }
  } finally {
    isExporting.value = false
  }
}
<<end>>
</script>
<<if .HasRichTextInList>>
<style scoped>
.text-truncate {
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
<<end>>
