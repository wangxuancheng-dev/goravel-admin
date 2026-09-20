<template>
  <ListPage
    ref="listPageRef"
    page-class="demo-activity"
    :title="$t('menu.demo_activity')"
    :add-button-text="$t('demo_activity.add')"
    :add-button-disabled="getButtonState('demo_activity.store').disabled"
    :search-form="searchForm"
    :search-fields="searchFields"
    :initial-search-values="initialSearchForm"
    i18n-prefix="demo_activity"
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
    <template #toolbar-right>
      <el-button
        :disabled="getButtonState('demo_activity.sync').disabled"
        :loading="syncing"
        @click="handleSync"
      >
        {{ $t('demo_activity.sync_now') }}
      </el-button>
    </template>

    <template #schedule_type="{ row }">
      {{ typeLabel(row.schedule_type) }}
    </template>

    <template #window="{ row }">
      <span v-if="row.schedule_type === 'daily'">{{ row.daily_start || '-' }} ~ {{ row.daily_end || '-' }}</span>
      <span v-else-if="row.schedule_type === 'weekly'">
        {{ row.weekdays || '-' }} @ {{ row.daily_start || '-' }}-{{ row.daily_end || '-' }}
      </span>
      <span v-else-if="row.schedule_type === 'monthly'">
        <template v-if="row.month_days">{{ row.month_days }} @ {{ row.daily_start || '-' }}-{{ row.daily_end || '-' }}</template>
        <template v-else>D{{ row.month_day_start }}~D{{ row.month_day_end }}</template>
      </span>
      <span v-else-if="row.schedule_type === 'yearly'">{{ row.year_start || '-' }} ~ {{ row.year_end || '-' }}</span>
      <span v-else>{{ formatDemoTime(row.start_at) }} ~ {{ formatDemoTime(row.end_at) }}</span>
    </template>

    <template #status="{ row }">
      <el-tag :type="statusTagType(row.status)">
        {{ statusLabel(row.status) }}
      </el-tag>
    </template>

    <template #enabled="{ row }">
      <el-tag :type="row.enabled !== false ? 'success' : 'info'">
        {{ row.enabled !== false ? $t('common.enabled') : $t('common.disabled') }}
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
      <DemoActivityForm
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
import { ElMessage, ElMessageBox } from 'element-plus'
import ListPage from '@/components/ListPage.vue'
import TableActionButtons from '@/components/TableActionButtons.vue'
import DemoActivityForm from './DemoActivityForm.vue'
import { useStandardListPage } from '@/composables/useStandardListPage'
import { createCrudActions } from '@/utils/listPageHelpers'
import {
  getDemoActivityList,
  deleteDemoActivity,
  checkDemoActivityActive,
  syncDemoActivities
} from '@/api/demoActivity'

const { t } = useI18n()

const formatDemoTime = (value) => {
  if (!value) return '-'
  const s = String(value).trim()
  if (/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/.test(s)) return s
  const d = new Date(s)
  if (Number.isNaN(d.getTime())) return s
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}
const listPageRef = ref(null)
const syncing = ref(false)
const initialSearchForm = { title: '' }

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
  fetchApi: getDemoActivityList,
  initialSearchForm,
  defaultSort: 'id:desc',
  deleteApi: deleteDemoActivity,
  tableRef: computed(() => listPageRef.value?.tableRef?.tableRef)
})

const searchFields = computed(() => [
  { prop: 'title', label: t('demo_activity.title'), type: 'input' }
])

const tableColumns = computed(() => [
  { field: 'id', title: t('table.id'), width: 80, sortable: true },
  { field: 'title', title: t('demo_activity.title'), minWidth: 140 },
  { field: 'schedule_type', title: t('demo_activity.schedule_type'), width: 100, slot: 'schedule_type' },
  { field: 'window', title: t('demo_activity.window'), minWidth: 220, slot: 'window', sortable: false },
  { field: 'status', title: t('demo_activity.status'), width: 110, slot: 'status' },
  { field: 'enabled', title: t('common.enabled'), width: 90, slot: 'enabled' },
  { field: 'operation', title: t('table.operation'), width: 260, slot: 'operation', fixed: 'right', sortable: false }
])

const statusLabel = (status) => {
  if (status === 1) return t('demo_activity.status_running')
  if (status === 2) return t('demo_activity.status_ended')
  return t('demo_activity.status_pending')
}

const statusTagType = (status) => {
  if (status === 1) return 'success'
  if (status === 2) return 'info'
  return ''
}

const typeLabel = (v) => {
  if (v === 'daily') return t('demo_activity.type_daily')
  if (v === 'weekly') return t('demo_activity.type_weekly')
  if (v === 'monthly') return t('demo_activity.type_monthly')
  if (v === 'yearly') return t('demo_activity.type_yearly')
  return t('demo_activity.type_once')
}

const handleCheck = async (row) => {
  try {
    const res = await checkDemoActivityActive(row.id)
    const data = res?.data || {}
    await ElMessageBox.alert(
      [
        `${t('demo_activity.stored_status')}: ${statusLabel(data.stored_status)}`,
        `${t('demo_activity.desired_status')}: ${statusLabel(data.desired_status)}`,
        `${t('demo_activity.is_active_live')}: ${data.is_active_live ? t('common.yes') : t('common.no')}`
      ].join('\n'),
      t('demo_activity.active_check')
    )
  } catch (e) {
    // global handler
  }
}

const handleSync = async () => {
  syncing.value = true
  try {
    const res = await syncDemoActivities()
    ElMessage.success(t('demo_activity.sync_success', { count: res?.data?.updated ?? 0 }))
    loadData()
  } finally {
    syncing.value = false
  }
}

const operationActions = computed(() => [
  ...createCrudActions(t, 'demo_activity', {
    onEdit: handleEdit,
    onDelete: handleDelete
  }),
  {
    key: 'check',
    label: t('demo_activity.active_check'),
    permission: 'demo_activity.active_check',
    handler: handleCheck
  }
])
</script>
