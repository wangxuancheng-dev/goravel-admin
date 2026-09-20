<template>
  <ListPage
    ref="listPageRef"
    page-class="import"
    :title="embedded ? '' : $t('export.tab_imports')"
    :show-add-button="false"
    :search-form="searchForm"
    :search-fields="searchFields"
    :initial-search-values="initialSearchForm"
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
      <div>
        <el-tag :type="statusTagType(row.status)">
          {{ formatStatus(row.status) }}
        </el-tag>
        <div
          v-if="Number(row.status) === 2 && row.error_msg"
          style="margin-top: 4px; color: #f56c6c; font-size: 12px; word-break: break-all;"
        >
          {{ row.error_msg }}
        </div>
      </div>
    </template>

    <template #operation="{ row }">
      <el-button
        v-if="row.error_file_url"
        type="primary"
        link
        :loading="downloadingIds.has(row.id)"
        @click="handleDownloadError(row)"
      >
        {{ $t('import.download_error') }}
      </el-button>
      <el-button
        type="danger"
        link
        :disabled="getButtonState('import.destroy').disabled"
        @click="handleDelete(row)"
      >
        {{ $t('common.delete') }}
      </el-button>
    </template>
  </ListPage>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import ListPage from '@/components/ListPage.vue'
import { useStandardListPage } from '@/composables/useStandardListPage'
import { getImportList, deleteImport } from '@/api/import'
import { buildAdminAuthHeaders } from '@/utils/authHeaders'

defineProps({
  embedded: { type: Boolean, default: true }
})

const { t } = useI18n()
const listPageRef = ref(null)
const downloadingIds = ref(new Set())

const initialSearchForm = {
  type: '',
  status: '',
  start_time: '',
  end_time: ''
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
  handleDelete,
  getButtonState
} = useStandardListPage({
  fetchApi: getImportList,
  initialSearchForm,
  defaultSort: 'id:desc',
  deleteApi: deleteImport,
  normalizeRows: false,
  transformData: (row) => ({
    id: row.id,
    type: row.type || '',
    status: Number(row.status ?? 0),
    total_rows: Number(row.total_rows ?? 0),
    success_rows: Number(row.success_rows ?? 0),
    failed_rows: Number(row.failed_rows ?? 0),
    error_msg: row.error_msg || '',
    error_file_url: row.error_file_url || '',
    created_at: row.created_at || '',
    admin: row.admin || row.Admin || null
  }),
  tableRef: computed(() => listPageRef.value?.tableRef?.tableRef)
})

const searchFields = computed(() => [
  {
    prop: 'type',
    label: t('export.type'),
    type: 'select',
    options: [
      { label: t('export.types.orders'), value: 'orders' },
      { label: t('export.types.articles'), value: 'articles' }
    ]
  },
  {
    prop: 'status',
    label: t('log.status'),
    type: 'select',
    options: [
      { label: t('log.processing'), value: '0' },
      { label: t('log.success'), value: '1' },
      { label: t('log.failed'), value: '2' }
    ]
  },
  { prop: 'start_time', label: t('log.start_time'), type: 'datetime' },
  { prop: 'end_time', label: t('log.end_time'), type: 'datetime' }
])

const tableColumns = computed(() => [
  { prop: 'id', label: t('table.id'), width: 80, sortable: 'custom' },
  {
    prop: 'type',
    label: t('export.type'),
    width: 120,
    formatter: ({ cellValue }) => {
      const type = cellValue || ''
      if (!type) return '-'
      const key = `export.types.${type}`
      const translated = t(key)
      return translated !== key ? translated : type
    }
  },
  { prop: 'status', label: t('log.status'), width: 140, slot: 'status' },
  { prop: 'total_rows', label: t('import.total_rows'), width: 100 },
  { prop: 'success_rows', label: t('import.success_rows'), width: 100 },
  { prop: 'failed_rows', label: t('import.failed_rows'), width: 100 },
  {
    prop: 'admin',
    label: t('log.admin'),
    width: 140,
    formatter: ({ cellValue }) => cellValue?.username || cellValue?.Username || '-'
  },
  { prop: 'created_at', label: t('table.created_at'), width: 180, sortable: 'custom' },
  { prop: 'operation', label: t('common.operation'), width: 200, slot: 'operation', fixed: 'right' }
])

const formatStatus = (status) => {
  if (Number(status) === 1) return t('log.success')
  if (Number(status) === 0) return t('log.processing')
  return t('log.failed')
}

const statusTagType = (status) => {
  if (Number(status) === 1) return 'success'
  if (Number(status) === 0) return 'warning'
  return 'danger'
}

const handleDownloadError = async (row) => {
  if (!row.error_file_url || downloadingIds.value.has(row.id)) return
  downloadingIds.value.add(row.id)
  try {
    let fullUrl = row.error_file_url
    if (fullUrl.startsWith('/')) {
      const apiBaseURL = import.meta.env.VITE_API_BASE_URL
      const apiPrefix = import.meta.env.VITE_API_PREFIX || '/api/admin'
      if (apiBaseURL) {
        const cleanUrl = fullUrl.replace(/^\/api\/admin/, '')
        fullUrl = `${apiBaseURL.replace(/\/+$/, '')}${apiPrefix.replace(/\/+$/, '')}${cleanUrl}`
      }
    }
    const response = await fetch(fullUrl, {
      method: 'GET',
      headers: buildAdminAuthHeaders()
    })
    if (!response.ok) throw new Error(`HTTP ${response.status}`)
    const blob = await response.blob()
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `import_errors_${row.id}.csv`
    link.click()
    window.URL.revokeObjectURL(url)
    ElMessage.success(t('export.download_success'))
  } catch (e) {
    ElMessage.error(t('export.download_failed'))
  } finally {
    downloadingIds.value.delete(row.id)
  }
}
</script>
