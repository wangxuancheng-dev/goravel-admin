<template>
  <TreeListPage
    ref="treeListPageRef"
    page-class="<<.ModuleName>>"
    :title="$t('menu.<<.ModuleName>>')"
    :add-button-text="$t('<<.ModuleName>>.add_<<.ModuleName>>')"
    :add-button-disabled="getButtonState('<<.ModuleName>>.store').disabled"
    :search-form="searchForm"
    :search-fields="searchFields"
    :initial-search-values="<<.ModuleNameCamel>>InitialSearchForm"
    i18n-prefix="<<.ModuleName>>"
    :table-data="tableData"
    :loading="loading"
    :table-columns="tableColumns"
    :table-key="`table-${tableColumns.length}-${JSON.stringify(columnOrder)}`"
    :default-expand-all="isExpanded"
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
  >
    <template #header-extra>
      <el-button v-if="!hasSearch" @click="handleToggleExpand">
        <el-icon><component :is="isExpanded ? Fold : Expand" /></el-icon>
        {{ isExpanded ? $t('menu_management.collapse_all') : $t('menu_management.expand_all') }}
      </el-button>
    </template>

    <<range .ListFields>>
    <<- if and .ShowInList (eq .Name "status")>>
    <template #status="{ row }">
      <el-tag :type="Number(row.status ?? 1) === 1 ? 'success' : 'danger'">
        {{ Number(row.status ?? 1) === 1 ? $t('common.enabled') : $t('common.disabled') }}
      </el-tag>
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
      <<if .HasEdit>>
      <el-button
        type="primary"
        link
        :disabled="getButtonState('<<.ModuleName>>.update').disabled"
        @click="handleEdit(row)"
      >
        {{ $t('common.edit') }}
      </el-button>
      <<end>>
      <<if .HasDelete>>
      <el-button
        type="danger"
        link
        :disabled="getButtonState('<<.ModuleName>>.destroy').disabled"
        @click="handleDelete(row)"
      >
        {{ $t('common.delete') }}
      </el-button>
      <<end>>
    </template>

    <template #form>
      <<printf "<%s" .ModelName>>Form
        v-model="dialogVisible"
        :edit-id="editId"
        @success="handleFormSuccess"
      />
    </template>
  </TreeListPage>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Fold, Expand } from '@element-plus/icons-vue'
import TreeListPage from '@/components/TreeListPage.vue'
import <<.ModelName>>Form from './<<.ModelName>>Form.vue'
import { useTreeListPage } from '@/composables/useTreeListPage'
import { usePermission } from '@/composables/usePermission'
import { useCrud } from '@/composables/useCrud'
import { useElTableColumns } from '@/composables/useElTableColumns'
import { useColumnSetting } from '@/composables/useColumnSetting'
<<if .HasRichTextInList>>
import { extractTextFromMarkdown } from '@/utils/markdown'
<<end>>
import {
  get<<.ModelName>>List
  <<if .HasDelete>>, delete<<.ModelName>><<end>>
} from '@/api/<<.ModuleName>>'
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
const { getButtonState } = usePermission()
const treeListPageRef = ref(null)
const isExpanded = ref(false)

<<if .HasDelete>>
const {
  dialogVisible,
  editId,
  handleAdd,
  handleEdit,
  handleFormSuccess: onFormSuccess,
  handleDelete: handleDeleteCrud
} = useCrud({
  deleteApi: delete<<.ModelName>>
})
<<else>>
const dialogVisible = ref(false)
const editId = ref(null)
const handleAdd = () => {
  editId.value = null
  dialogVisible.value = true
}
const handleEdit = (row) => {
  editId.value = row.id
  dialogVisible.value = true
}
const onFormSuccess = (reload) => {
  dialogVisible.value = false
  editId.value = null
  if (reload) reload()
}
const handleDeleteCrud = () => {}
<<end>>

const {
  searchForm,
  tableData,
  loading,
  loadData,
  handleSearch,
  handleReset
} = useTreeListPage({
  fetchApi: get<<.ModelName>>List,
  initialSearchForm: <<.ModuleNameCamel>>InitialSearchForm,
  buildParams: build<<.ModelName>>ListParams,
})

const hasSearch = computed(() => {
  return Object.values(searchForm).some((value) => {
    if (Array.isArray(value)) return value.length > 0
    return value !== '' && value != null
  })
})

const searchFields = computed(() => create<<.ModelName>>SearchFields(t))

const allTableColumns = computed(() => create<<.ModelName>>TableColumns(t))

const {
  tableColumns: tableColumnsConfig,
  visibleColumns,
  allColumns,
  defaultVisibleColumns,
  columnOrder,
  fixedColumns,
  handleColumnSettingConfirm
} = useColumnSetting('<<.ModuleName>>', allTableColumns)

const tableColumns = useElTableColumns(tableColumnsConfig, visibleColumns, columnOrder, fixedColumns)

const handleFormSuccess = () => onFormSuccess(loadData)
const handleDelete = (row) => handleDeleteCrud(row, loadData)

const handleToggleExpand = () => {
  isExpanded.value = !isExpanded.value
  const elTable = treeListPageRef.value?.tableRef
  if (!elTable) return

  const toggleNode = (rows) => {
    if (!Array.isArray(rows)) return
    rows.forEach((row) => {
      elTable.toggleRowExpansion(row, isExpanded.value)
      if (row.children?.length) toggleNode(row.children)
    })
  }
  toggleNode(tableData.value)
}

onMounted(() => {
  loadData()
})
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
