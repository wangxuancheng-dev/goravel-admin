<template>
  <ListPage
    page-class="platform-login-log"
    :title="$t('menu.login_log')"
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
    <template #message="{ row }">
      {{ translateLoginMessage(row.message) }}
    </template>
    <template #operation="{ row }">
      <el-button link type="primary" @click="handleView(row)">{{ $t('common.view') }}</el-button>
    </template>
  </ListPage>

  <el-dialog v-model="detailVisible" :title="$t('log.detail')" width="900px" destroy-on-close>
    <el-descriptions v-if="logDetail" :column="2" border>
      <el-descriptions-item :label="$t('table.id')">{{ logDetail.id }}</el-descriptions-item>
      <el-descriptions-item :label="$t('log.admin')">{{ logDetail.username || '-' }}</el-descriptions-item>
      <el-descriptions-item :label="$t('log.ip')">{{ logDetail.ip || '-' }}</el-descriptions-item>
      <el-descriptions-item :label="$t('log.location')">{{ logDetail.location || '-' }}</el-descriptions-item>
      <el-descriptions-item :label="$t('table.status')">
        <el-tag :type="Number(logDetail.status) === 1 ? 'success' : 'danger'" size="small">
          {{ Number(logDetail.status) === 1 ? $t('log.success') : $t('log.failed') }}
        </el-tag>
      </el-descriptions-item>
      <el-descriptions-item :label="$t('log.login_time')">{{ logDetail.created_at || '-' }}</el-descriptions-item>
      <el-descriptions-item :label="$t('log.user_agent')" :span="2">{{ logDetail.user_agent || '-' }}</el-descriptions-item>
      <el-descriptions-item :label="$t('log.message')" :span="2">{{ translateLoginMessage(logDetail.message) }}</el-descriptions-item>
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
import { getPlatformLoginLogDetail, getPlatformLoginLogList } from '@/api/platform'

const { t } = useI18n()
const initialSearchForm = {
  username: '',
  ip: '',
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
  fetchApi: getPlatformLoginLogList,
  initialSearchForm,
  defaultSort: 'id:desc',
})

const detailVisible = ref(false)
const logDetail = ref(null)

const searchFields = computed(() => [
  { prop: 'username', label: t('log.admin'), type: 'input', width: '140px' },
  { prop: 'ip', label: t('log.ip'), type: 'input', width: '140px' },
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
  { field: 'username', title: t('log.admin'), width: 140, key: 'username' },
  { field: 'ip', title: t('log.ip'), width: 140, key: 'ip' },
  { field: 'location', title: t('log.location'), width: 140, key: 'location' },
  { field: 'status', title: t('table.status'), width: 100, slot: 'status', key: 'status' },
  { field: 'message', title: t('log.message'), minWidth: 160, slot: 'message', key: 'message' },
  { field: 'created_at', title: t('log.login_time'), width: 170, key: 'created_at' },
  { field: 'operation', title: t('table.operation'), width: 90, slot: 'operation', key: 'operation', fixed: 'right' },
])

const translateLoginMessage = (messageKey) => {
  if (!messageKey) return '-'
  const key = `log.${messageKey}`
  const translation = t(key)
  return translation !== key ? translation : messageKey
}

const formatRequest = (raw) => {
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    return String(raw || '')
  }
}

const handleView = async (row) => {
  try {
    const res = await getPlatformLoginLogDetail(row.id)
    logDetail.value = res?.data?.login_log || res?.data || row
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
