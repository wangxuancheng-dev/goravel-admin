<template>
  <ListPage
    page-class="platform-operation-log"
    :title="$t('menu.operation_log')"
    :show-add-button="false"
    :search-form="searchForm"
    :search-fields="searchFields"
    :initial-search-values="initialSearchForm"
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
  >
    <template #status="{ row }">
      <el-tag :type="Number(row.status) === 1 ? 'success' : 'danger'" size="small">
        {{ Number(row.status) === 1 ? $t('log.success') : $t('log.failed') }}
      </el-tag>
    </template>
    <template #title="{ row }">
      {{ translateTitle(row.title) }}
    </template>
    <template #operation="{ row }">
      <el-button link type="primary" @click="handleView(row)">{{ $t('common.view') }}</el-button>
    </template>
  </ListPage>

  <el-dialog v-model="detailVisible" :title="$t('log.detail')" width="900px" destroy-on-close>
    <el-descriptions v-if="logDetail" :column="2" border>
      <el-descriptions-item :label="$t('table.id')">{{ logDetail.id }}</el-descriptions-item>
      <el-descriptions-item :label="$t('log.admin')">{{ logDetail.username || '-' }}</el-descriptions-item>
      <el-descriptions-item :label="$t('log.method')">{{ logDetail.method || '-' }}</el-descriptions-item>
      <el-descriptions-item :label="$t('log.title')">{{ translateTitle(logDetail.title) }}</el-descriptions-item>
      <el-descriptions-item :label="$t('log.path')" :span="2">{{ logDetail.path || '-' }}</el-descriptions-item>
      <el-descriptions-item :label="$t('log.ip')">{{ logDetail.ip || '-' }}</el-descriptions-item>
      <el-descriptions-item :label="$t('table.status')">
        <el-tag :type="Number(logDetail.status) === 1 ? 'success' : 'danger'" size="small">
          {{ Number(logDetail.status) === 1 ? $t('log.success') : $t('log.failed') }}
        </el-tag>
      </el-descriptions-item>
      <el-descriptions-item :label="$t('log.operation_time')">{{ logDetail.created_at || '-' }}</el-descriptions-item>
      <el-descriptions-item label="Duration">{{ logDetail.duration ?? '-' }} ms</el-descriptions-item>
      <el-descriptions-item :label="$t('log.user_agent')" :span="2">{{ logDetail.user_agent || '-' }}</el-descriptions-item>
      <el-descriptions-item :label="$t('log.request')" :span="2">
        <pre v-if="logDetail.request" class="request-preview">{{ formatRequest(logDetail.request) }}</pre>
        <span v-else>-</span>
      </el-descriptions-item>
    </el-descriptions>
  </el-dialog>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ListPage from '@/components/ListPage.vue'
import { useStandardListPage } from '@/composables/useStandardListPage'
import { getPlatformOperationLogDetail, getPlatformOperationLogList } from '@/api/platform'
import { translatePlatformOpTitle } from '@/utils/platformOpTitle'

const { t, te } = useI18n()
const translateTitle = (title) => translatePlatformOpTitle(t, te, title)
const initialSearchForm = {
  username: '',
  method: '',
  path: '',
  status: '',
}

const {
  pagination,
  tableData,
  loading,
  searchForm,
  loadData,
  handleSearch,
  handleReset,
  handleSortChange,
} = useStandardListPage({
  fetchApi: getPlatformOperationLogList,
  initialSearchForm,
  defaultSort: 'id:desc',
})

const detailVisible = ref(false)
const logDetail = ref(null)

const searchFields = computed(() => [
  { prop: 'username', label: t('log.admin'), type: 'input', width: '140px' },
  {
    prop: 'method',
    label: t('log.method'),
    type: 'select',
    width: '120px',
    options: [
      { label: 'POST', value: 'POST' },
      { label: 'PUT', value: 'PUT' },
      { label: 'PATCH', value: 'PATCH' },
      { label: 'DELETE', value: 'DELETE' },
    ],
  },
  { prop: 'path', label: t('log.path'), type: 'input', width: '180px' },
  {
    prop: 'status',
    label: t('table.status'),
    type: 'select',
    width: '120px',
    options: [
      { label: t('log.success'), value: '1' },
      { label: t('log.failed'), value: '0' },
    ],
  },
])

const tableColumns = computed(() => [
  { field: 'id', title: t('table.id'), width: 70, sortable: true, key: 'id' },
  { field: 'username', title: t('log.admin'), width: 120, key: 'username' },
  { field: 'method', title: t('log.method'), width: 90, key: 'method' },
  { field: 'title', title: t('log.title'), minWidth: 180, slot: 'title', key: 'title' },
  { field: 'path', title: t('log.path'), minWidth: 200, key: 'path' },
  { field: 'ip', title: t('log.ip'), width: 130, key: 'ip' },
  { field: 'status', title: t('table.status'), width: 90, slot: 'status', key: 'status' },
  { field: 'duration', title: 'ms', width: 80, key: 'duration' },
  { field: 'created_at', title: t('log.operation_time'), width: 170, key: 'created_at' },
  { field: 'operation', title: t('table.operation'), width: 90, slot: 'operation', key: 'operation', fixed: 'right' },
])

const formatRequest = (raw) => {
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    return String(raw || '')
  }
}

const handleView = async (row) => {
  try {
    const res = await getPlatformOperationLogDetail(row.id)
    logDetail.value = res?.data?.operation_log || res?.data || row
    detailVisible.value = true
  } catch (e) {
    console.error(e)
  }
}
</script>

<style scoped>
.request-preview {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 280px;
  overflow: auto;
}
</style>
