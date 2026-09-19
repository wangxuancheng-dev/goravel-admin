<template>
  <div>
    <el-alert
      type="info"
      :closable="false"
      show-icon
      class="scope-hint"
      :title="scopeHint"
    />
    <ListPage
      page-class="platform-system-log"
      :title="$t('menu.system_log')"
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
      <template #level="{ row }">
        <el-tag :type="getSystemLogLevelType(row.level)" size="small">
          {{ getSystemLogLevelLabel(t, row.level) }}
        </el-tag>
      </template>
      <template #context="{ row }">
        <el-tooltip
          v-if="row.context"
          :content="formatSystemLogContext(row.context)"
          placement="top"
          effect="dark"
        >
          <div class="context-preview">{{ formatSystemLogContextPreview(row.context) }}</div>
        </el-tooltip>
        <span v-else>-</span>
      </template>
      <template #operation="{ row }">
        <el-button link type="primary" @click="handleView(row)">{{ $t('common.view') }}</el-button>
      </template>
    </ListPage>

    <el-dialog v-model="detailVisible" :title="$t('log.detail')" width="900px" destroy-on-close>
      <el-descriptions v-if="logDetail" :column="2" border>
        <el-descriptions-item :label="$t('table.id')">{{ logDetail.id }}</el-descriptions-item>
        <el-descriptions-item :label="$t('log.level')">
          <el-tag :type="getSystemLogLevelType(logDetail.level)" size="small">
            {{ getSystemLogLevelLabel(t, logDetail.level) }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item :label="$t('log.module')">{{ logDetail.module || '-' }}</el-descriptions-item>
        <el-descriptions-item :label="$t('log.trace_id')">{{ logDetail.trace_id || '-' }}</el-descriptions-item>
        <el-descriptions-item :label="$t('log.message')" :span="2">{{ logDetail.message }}</el-descriptions-item>
        <el-descriptions-item :label="$t('log.context')" :span="2">
          <pre v-if="logDetail.context" class="context-full">{{ formatSystemLogContext(logDetail.context) }}</pre>
          <span v-else>-</span>
        </el-descriptions-item>
        <el-descriptions-item :label="$t('log.time')" :span="2">{{ logDetail.created_at || '-' }}</el-descriptions-item>
      </el-descriptions>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import ListPage from '@/components/ListPage.vue'
import { useStandardListPage } from '@/composables/useStandardListPage'
import {
  getPlatformSystemLogDetail,
  getPlatformSystemLogList,
  getPlatformTenantList,
} from '@/api/platform'
import {
  formatSystemLogContext,
  formatSystemLogContextPreview,
  getSystemLogLevelLabel,
  getSystemLogLevelType,
  systemLogInitialSearchForm,
  transformSystemLogRow,
} from '@/views/log/systemLog.config'

const { t } = useI18n()
const route = useRoute()

const tenantOptions = ref([])
const initialSearchForm = {
  tenant_code: '',
  ...systemLogInitialSearchForm,
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
  fetchApi: getPlatformSystemLogList,
  initialSearchForm,
  defaultSort: 'id:desc',
  transformData: transformSystemLogRow,
})

const detailVisible = ref(false)
const logDetail = ref(null)

const scopeHint = computed(() => {
  const code = searchForm.tenant_code
  return code
    ? t('platform.system_log_scope_tenant', { code })
    : t('platform.system_log_scope_platform')
})

const searchFields = computed(() => [
  {
    prop: 'tenant_code',
    label: t('tenant.code'),
    type: 'select',
    width: '160px',
    clearable: true,
    options: [
      { label: t('platform.system_log_platform_opt'), value: '' },
      ...tenantOptions.value,
    ],
  },
  {
    prop: 'level',
    label: t('log.level'),
    type: 'select',
    width: '120px',
    options: [
      { label: 'error', value: 'error' },
      { label: 'warning', value: 'warning' },
      { label: 'info', value: 'info' },
      { label: 'debug', value: 'debug' },
    ],
  },
  { prop: 'module', label: t('log.module'), type: 'input', width: '140px' },
  { prop: 'trace_id', label: t('log.trace_id'), type: 'input', width: '160px' },
  { prop: 'message', label: t('log.message'), type: 'input', width: '160px' },
])

const tableColumns = computed(() => [
  { field: 'id', title: t('table.id'), width: 70, sortable: true, key: 'id' },
  { field: 'level', title: t('log.level'), width: 100, slot: 'level', key: 'level' },
  { field: 'module', title: t('log.module'), width: 140, key: 'module' },
  { field: 'trace_id', title: t('log.trace_id'), minWidth: 160, key: 'trace_id' },
  { field: 'message', title: t('log.message'), minWidth: 220, key: 'message' },
  { field: 'context', title: t('log.context'), minWidth: 160, slot: 'context', key: 'context' },
  { field: 'created_at', title: t('log.time'), width: 170, key: 'created_at' },
  { field: 'operation', title: t('table.operation'), width: 90, slot: 'operation', key: 'operation', fixed: 'right' },
])

const loadTenants = async () => {
  try {
    const res = await getPlatformTenantList({ page: 1, page_size: 200, order_by: 'code:asc' })
    tenantOptions.value = (res?.data?.list || []).map((row) => ({
      label: row.code,
      value: row.code,
    }))
  } catch (e) {
    console.error(e)
  }
}

const handleView = async (row) => {
  try {
    const res = await getPlatformSystemLogDetail(row.id, {
      tenant_code: searchForm.tenant_code || undefined,
    })
    logDetail.value = transformSystemLogRow(res?.data?.log || res?.data || row)
    detailVisible.value = true
  } catch (e) {
    console.error(e)
  }
}

onMounted(async () => {
  await loadTenants()
  const code = typeof route.query.code === 'string' ? route.query.code : ''
  if (code) {
    searchForm.tenant_code = code
    await handleSearch()
  }
})

watch(
  () => route.query.code,
  async (code) => {
    if (typeof code === 'string' && code !== searchForm.tenant_code) {
      searchForm.tenant_code = code
      await handleSearch()
    }
  },
)
</script>

<style scoped>
.scope-hint {
  margin-bottom: 12px;
}
.context-preview {
  max-width: 220px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.context-full {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 280px;
  overflow: auto;
}
</style>
