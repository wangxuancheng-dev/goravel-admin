<template>
  <ListPage
    ref="listPageRef"
    page-class="allowlist"
    :title="$t('menu.allowlist')"
    :add-button-text="$t('allowlist.add_allowlist')"
    :add-button-disabled="getButtonState('allowlist.store').disabled"
    :search-form="searchForm"
    :search-fields="searchFields"
    :initial-search-values="allowlistInitialSearchForm"
    i18n-prefix="allowlist"
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
    <template #toolbar-left>
      <el-alert
        v-if="clientIp"
        type="info"
        :closable="false"
        show-icon
        :title="t('allowlist.client_ip_tip', { ip: clientIp })"
        style="margin-bottom: 8px; width: 100%;"
      />
    </template>
    <template #ip="{ row }">
      <div style="word-break: break-all;">
        {{ formatAllowlistIP(row.ip) }}
      </div>
    </template>

    <template #status="{ row }">
      <el-tag :type="rowStatus(row) === 1 ? 'success' : 'info'">
        {{ rowStatus(row) === 1 ? $t('allowlist.enabled') : $t('allowlist.disabled') }}
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
      <AllowlistForm
        v-model="dialogVisible"
        :edit-id="editId"
        @success="handleFormSuccess"
      />
    </template>
  </ListPage>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ListPage from '@/components/ListPage.vue'
import TableActionButtons from '@/components/TableActionButtons.vue'
import AllowlistForm from './AllowlistForm.vue'
import { useStandardListPage } from '@/composables/useStandardListPage'
import { createCrudActions, rowStatus } from '@/utils/listPageHelpers'
import { getAllowlistList, deleteAllowlist } from '@/api/allowlist'
import {
  allowlistInitialSearchForm,
  createAllowlistSearchFields,
  createAllowlistTableColumns,
  formatAllowlistIP
} from './allowlist.config'

const { t } = useI18n()
const listPageRef = ref(null)
const clientIp = ref('')

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
  fetchApi: getAllowlistList,
  initialSearchForm: allowlistInitialSearchForm,
  defaultSort: 'id:desc',
  deleteApi: deleteAllowlist,
  requireSensitiveConfirm: true,
  tableRef: computed(() => listPageRef.value?.tableRef?.tableRef)
})

const searchFields = computed(() => createAllowlistSearchFields(t))
const tableColumns = computed(() => createAllowlistTableColumns(t))

const operationActions = computed(() =>
  createCrudActions(t, 'allowlist', {
    onEdit: handleEdit,
    onDelete: handleDelete
  })
)

onMounted(async () => {
  try {
    const res = await getAllowlistList({ page: 1, page_size: 1 })
    clientIp.value = String(res?.data?.client_ip || '')
  } catch {
    clientIp.value = ''
  }
})
</script>
