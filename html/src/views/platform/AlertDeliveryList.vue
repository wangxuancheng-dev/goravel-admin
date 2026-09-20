<template>
  <ListPage
    page-class="platform-alert-delivery"
    :title="$t('menu.platform_alert')"
    :show-add-button="false"
    :search-form="searchForm"
    :search-fields="searchFields"
    :initial-search-values="initialSearchForm"
    i18n-prefix="platform_alert"
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
      <el-tag :type="statusType(row.status)" size="small">{{ row.status || '-' }}</el-tag>
    </template>
    <template #actions="{ row }">
      <el-button
        v-if="isOwner && row.channel === 'webhook'"
        link
        type="primary"
        @click="onRetry(row)"
      >
        {{ $t('platform_alert.retry') }}
      </el-button>
      <span v-else>-</span>
    </template>
  </ListPage>
</template>

<script setup>
import { computed } from 'vue'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'
import ListPage from '@/components/ListPage.vue'
import { useStandardListPage } from '@/composables/useStandardListPage'
import { getPlatformAlertDeliveryList, retryPlatformAlertDelivery } from '@/api/platform'
import { isPlatformOwner } from '@/utils/platformRequest'

const { t } = useI18n()
const isOwner = isPlatformOwner()
const initialSearchForm = { event: '', status: '', channel: '', tenant_code: '' }

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
  fetchApi: getPlatformAlertDeliveryList,
  initialSearchForm,
  defaultSort: 'id:desc'
})

const searchFields = computed(() => [
  { prop: 'tenant_code', label: t('tenant.code'), type: 'input', width: '140px' },
  {
    prop: 'channel',
    label: t('platform_alert.channel'),
    type: 'select',
    width: '120px',
    options: [
      { label: 'webhook', value: 'webhook' },
      { label: 'mail', value: 'mail' }
    ]
  },
  {
    prop: 'status',
    label: t('platform_alert.status'),
    type: 'select',
    width: '120px',
    options: [
      { label: 'success', value: 'success' },
      { label: 'failed', value: 'failed' },
      { label: 'pending', value: 'pending' }
    ]
  },
  { prop: 'event', label: t('platform_alert.event'), type: 'input', width: '160px' }
])

const tableColumns = computed(() => [
  { field: 'id', title: t('table.id'), width: 70, sortable: true, key: 'id' },
  { field: 'channel', title: t('platform_alert.channel'), width: 90, key: 'channel' },
  { field: 'event', title: t('platform_alert.event'), width: 160, key: 'event' },
  { field: 'tenant_code', title: t('tenant.code'), width: 110, key: 'tenant_code' },
  { field: 'op', title: t('tenant_op_log.op'), width: 90, key: 'op' },
  { field: 'status', title: t('platform_alert.status'), width: 100, slot: 'status', key: 'status' },
  { field: 'http_status', title: t('platform_alert.http_status'), width: 90, key: 'http_status' },
  { field: 'attempt', title: t('platform_alert.attempt'), width: 80, key: 'attempt' },
  { field: 'target_masked', title: t('platform_alert.target'), key: 'target_masked' },
  { field: 'error_message', title: t('platform_alert.error'), key: 'error_message' },
  { field: 'created_at', title: t('table.created_at'), width: 170, key: 'created_at' },
  { field: 'actions', title: t('table.actions'), width: 100, slot: 'actions', key: 'actions' }
])

function statusType(status) {
  if (status === 'success') return 'success'
  if (status === 'failed') return 'danger'
  return 'info'
}

async function onRetry(row) {
  try {
    await retryPlatformAlertDelivery(row.id)
    ElMessage.success(t('platform_alert.retry_success'))
    loadData()
  } catch (e) {
    ElMessage.error(e?.message || t('common.operation_failed'))
  }
}
</script>
