<template>
  <ListPage
    page-class="platform-tenant-op-log"
    :title="$t('menu.tenant_op_log')"
    :show-add-button="false"
    :search-form="searchForm"
    :search-fields="searchFields"
    :initial-search-values="initialSearchForm"
    i18n-prefix="tenant_op_log"
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
  >
    <template #status="{ row }">
      <el-tag :type="statusTag(row.status)" size="small">{{ row.status || '—' }}</el-tag>
    </template>
    <template #message="{ row }">
      <el-tooltip :content="row.message || ''" :disabled="!row.message">
        <span class="msg">{{ shortMsg(row.message) }}</span>
      </el-tooltip>
    </template>
  </ListPage>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import ListPage from '@/components/ListPage.vue'
import { useStandardListPage } from '@/composables/useStandardListPage'
import { getPlatformTenantOpLogList } from '@/api/platform'

const { t } = useI18n()
const route = useRoute()
const initialSearchForm = {
  code: String(route.query.code || ''),
  op: '',
  status: '',
  batch_id: '',
  operator: '',
}

const {
  pagination,
  tableData,
  loading,
  searchForm,
  loadData,
  handleSearch,
  handleReset,
  handleSortChange
} = useStandardListPage({
  fetchApi: getPlatformTenantOpLogList,
  initialSearchForm,
  defaultSort: 'id:desc'
})

const searchFields = computed(() => [
  { prop: 'code', label: t('tenant.code'), type: 'input', width: '140px' },
  {
    prop: 'op',
    label: t('tenant_op_log.op'),
    type: 'select',
    width: '140px',
    options: [
      { label: 'migrate', value: 'migrate' },
      { label: 'seed', value: 'seed' },
      { label: 'backup', value: 'backup' },
      { label: 'restore', value: 'restore' }
    ]
  },
  {
    prop: 'status',
    label: t('tenant_op_log.status'),
    type: 'select',
    width: '140px',
    options: [
      { label: 'queued', value: 'queued' },
      { label: 'running', value: 'running' },
      { label: 'success', value: 'success' },
      { label: 'failed', value: 'failed' }
    ]
  },
  { prop: 'batch_id', label: t('tenant_op_log.batch_id'), type: 'input', width: '180px' },
  { prop: 'operator', label: t('tenant_op_log.operator'), type: 'input', width: '140px' }
])

const tableColumns = computed(() => [
  { field: 'id', title: t('table.id'), width: 70, sortable: true, key: 'id' },
  { field: 'code', title: t('tenant.code'), width: 110, key: 'code' },
  { field: 'op', title: t('tenant_op_log.op'), width: 100, key: 'op' },
  { field: 'status', title: t('tenant_op_log.status'), width: 100, slot: 'status', key: 'status' },
  { field: 'operator_name', title: t('tenant_op_log.operator'), width: 120, key: 'operator_name' },
  { field: 'batch_id', title: t('tenant_op_log.batch_id'), minWidth: 160, key: 'batch_id' },
  { field: 'message', title: t('tenant_op_log.message'), minWidth: 180, slot: 'message', key: 'message' },
  { field: 'started_at', title: t('tenant_op_log.started_at'), width: 170, key: 'started_at' },
  { field: 'finished_at', title: t('tenant_op_log.finished_at'), width: 170, key: 'finished_at' }
])

const statusTag = (status) => {
  switch (status) {
    case 'success':
      return 'success'
    case 'failed':
      return 'danger'
    case 'running':
      return 'warning'
    default:
      return 'info'
  }
}

const shortMsg = (msg) => {
  const s = String(msg || '')
  if (!s) return '—'
  if (s.length <= 48) return s
  return `${s.slice(0, 46)}…`
}
</script>

<style scoped>
.msg {
  font-size: 12px;
  color: #64748b;
}
</style>
