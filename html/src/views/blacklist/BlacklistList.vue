<template>
  <ListPage
    ref="listPageRef"
    page-class="blacklist"
    :title="$t('menu.blacklist')"
    :add-button-text="$t('blacklist.add_blacklist')"
    :add-button-disabled="getButtonState('blacklist.store').disabled"
    :search-form="searchForm"
    :search-fields="searchFields"
    :initial-search-values="blacklistInitialSearchForm"
    i18n-prefix="blacklist"
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
    <template #ip="{ row }">
      <div style="word-break: break-all;">
        {{ formatBlacklistIP(row.ip) }}
      </div>
    </template>

    <template #status="{ row }">
      <el-tag :type="rowStatus(row) === 1 ? 'danger' : 'info'">
        {{ rowStatus(row) === 1 ? $t('blacklist.enabled') : $t('blacklist.disabled') }}
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
      <BlacklistForm
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
import ListPage from '@/components/ListPage.vue'
import TableActionButtons from '@/components/TableActionButtons.vue'
import BlacklistForm from './BlacklistForm.vue'
import { useStandardListPage } from '@/composables/useStandardListPage'
import { createCrudActions, rowStatus } from '@/utils/listPageHelpers'
import { getBlacklistList, deleteBlacklist } from '@/api/blacklist'
import {
  blacklistInitialSearchForm,
  createBlacklistSearchFields,
  createBlacklistTableColumns,
  formatBlacklistIP
} from './blacklist.config'

const { t } = useI18n()
const listPageRef = ref(null)

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
  handleFormSuccess,
  handleDelete,
  getButtonState
} = useStandardListPage({
  fetchApi: getBlacklistList,
  initialSearchForm: blacklistInitialSearchForm,
  defaultSort: 'id:desc',
  deleteApi: deleteBlacklist,
  requireSensitiveConfirm: true,
  tableRef: computed(() => listPageRef.value?.tableRef?.tableRef)
})

const searchFields = computed(() => createBlacklistSearchFields(t))
const tableColumns = computed(() => createBlacklistTableColumns(t))

const operationActions = computed(() =>
  createCrudActions(t, 'blacklist', {
    onEdit: handleEdit,
    onDelete: handleDelete
  })
)
</script>
