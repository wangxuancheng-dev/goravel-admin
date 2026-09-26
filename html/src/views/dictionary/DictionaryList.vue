<template>
  <ListPage
    ref="listPageRef"
    page-class="dictionary"
    :title="$t('menu.dictionary')"
    :add-button-text="$t('dictionary.add_dictionary')"
    :add-button-disabled="getButtonState('dictionary.store').disabled"
    :search-form="searchForm"
    :search-fields="searchFields"
    :initial-search-values="dictionaryInitialSearchForm"
    i18n-prefix="dictionary"
    :table-data="tableData"
    :loading="loading"
    :table-columns="tableColumns"
    :pagination="pagination"
    show-toolbar
    @add="handleAdd"
    @search="handleSearch"
    @reset="handleReset"
    @refresh="loadData"
    @page-change="loadData"
    @sort-change="handleSortChange"
  >
    <template #is_system="{ row }">
      <el-tag :type="isSystemDictionary(row) ? 'warning' : 'info'" size="small">
        {{ isSystemDictionary(row) ? $t('dictionary.system_yes') : $t('dictionary.system_no') }}
      </el-tag>
    </template>

    <template #status="{ row }">
      <el-tag :type="rowStatus(row) === 1 ? 'success' : 'danger'">
        {{ rowStatus(row) === 1 ? $t('common.enabled') : $t('common.disabled') }}
      </el-tag>
    </template>

    <template #operation="{ row }">
      <TableActionButtons
        :row="row"
        :primary-actions="operationActions"
        :get-button-state="getButtonState"
      />
    </template>

    <template #form>
      <DictionaryForm
        v-model="dialogVisible"
        :edit-id="editId"
        :type-options="typeOptions"
        @success="handleFormSuccess"
      />
    </template>
  </ListPage>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import ListPage from '@/components/ListPage.vue'
import TableActionButtons from '@/components/TableActionButtons.vue'
import DictionaryForm from './DictionaryForm.vue'
import { useStandardListPage } from '@/composables/useStandardListPage'
import { createCrudActions, rowStatus } from '@/utils/listPageHelpers'
import { getDictionaryList, deleteDictionary, getDictionaryTypes } from '@/api/dictionary'
import {
  dictionaryInitialSearchForm,
  createDictionarySearchFields,
  createDictionaryTableColumns,
  isSystemDictionary
} from './dictionary.config'

const { t } = useI18n()
const listPageRef = ref(null)
const typeOptions = ref([])

const loadTypeOptions = async () => {
  try {
    const res = await getDictionaryTypes()
    const types = res?.data?.types || []
    typeOptions.value = types.map((item) => ({ label: item, value: item }))
  } catch (error) {
    console.error('Failed to load dictionary types:', error)
    typeOptions.value = []
  }
}

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
  handleFormSuccess: baseHandleFormSuccess,
  handleDelete: baseHandleDelete,
  getButtonState
} = useStandardListPage({
  fetchApi: getDictionaryList,
  initialSearchForm: dictionaryInitialSearchForm,
  defaultSort: 'id:desc',
  deleteApi: deleteDictionary,
  tableRef: computed(() => listPageRef.value?.tableRef?.tableRef)
})

const handleFormSuccess = async () => {
  await loadTypeOptions()
  await baseHandleFormSuccess()
}

const handleDelete = async (row) => {
  if (isSystemDictionary(row)) {
    ElMessage.warning(t('dictionary.protected_cannot_delete'))
    return
  }
  await baseHandleDelete(row)
  await loadTypeOptions()
}

const searchFields = computed(() => createDictionarySearchFields(t, typeOptions.value))
const tableColumns = computed(() => createDictionaryTableColumns(t))

const operationActions = computed(() =>
  createCrudActions(t, 'dictionary', {
    onEdit: handleEdit,
    onDelete: handleDelete,
    deleteDisabled: (row) => isSystemDictionary(row)
  })
)

onMounted(() => {
  loadTypeOptions()
})
</script>
