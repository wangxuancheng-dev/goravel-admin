<template>
  <ListPage
    ref="listPageRef"
    page-class="export"
    :title="embedded ? '' : $t('menu.export')"
    :show-add-button="false"
    :search-form="searchForm"
    :search-fields="searchFields"
    :initial-search-values="exportInitialSearchForm"
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
    @selection-change="handleSelectionChange"
  >
    <template #header-actions>
      <el-button
        type="danger"
        :disabled="selectedRows.length === 0"
        @click="handleBatchDelete"
      >
        <el-icon><Delete /></el-icon>
        {{ $t('common.delete_selected') }} ({{ selectedRows.length }})
      </el-button>
    </template>

    <template #status="{ row }">
      <div>
        <el-tag :type="getExportStatusTagType(row)">
          {{ formatExportStatus(t, row) }}
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
        type="primary"
        link
        :disabled="downloadingIds.has(row.id) || !isExportCompleted(row)"
        :loading="downloadingIds.has(row.id)"
        @click="handleDownload(row)"
      >
        {{ $t('common.view') }}
      </el-button>
      <el-button
        type="danger"
        link
        :disabled="getButtonState('export.destroy').disabled"
        @click="handleDelete(row)"
      >
        {{ $t('common.delete') }}
      </el-button>
    </template>
  </ListPage>
</template>

<script setup>
import { ref, computed, onActivated } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Delete } from '@element-plus/icons-vue'
import ListPage from '@/components/ListPage.vue'
import { useStandardListPage } from '@/composables/useStandardListPage'
import { getExportList, deleteExport, batchDeleteExports } from '@/api/export'
import { buildAdminAuthHeaders } from '@/utils/authHeaders'
import {
  exportInitialSearchForm,
  transformExportRow,
  createExportSearchFields,
  createExportTableColumns,
  formatExportType,
  formatExportSize,
  formatExportAdmin,
  formatExportErrorMsg,
  formatExportStatus,
  getExportStatusTagType,
  isExportCompleted
} from './export.config'

defineProps({
  embedded: { type: Boolean, default: false }
})

const { t } = useI18n()
const listPageRef = ref(null)
const downloadingIds = ref(new Set())

const {
  pagination,
  tableData,
  loading,
  searchForm,
  selectedRows,
  loadData,
  handleSearch,
  handleReset,
  handleSortChange,
  handleSelectionChange,
  handleDelete: handleDeleteRow,
  handleBatchDelete: handleBatchDeleteRows,
  getButtonState
} = useStandardListPage({
  fetchApi: getExportList,
  initialSearchForm: exportInitialSearchForm,
  defaultSort: 'id:desc',
  deleteApi: deleteExport,
  batchDeleteApi: batchDeleteExports,
  normalizeRows: false,
  transformData: transformExportRow,
  tableRef: computed(() => listPageRef.value?.tableRef?.tableRef)
})

const searchFields = computed(() => createExportSearchFields(t))

const tableColumns = computed(() =>
  createExportTableColumns(t, {
    formatType: ({ cellValue }) => formatExportType(t, cellValue),
    formatSize: formatExportSize,
    formatAdmin: formatExportAdmin,
    formatErrorMsg: formatExportErrorMsg
  })
)

const handleDownload = async (row) => {
  const exportId = row.id
  if (downloadingIds.value.has(exportId)) return

  const url = row.file_url
  if (!url) {
    ElMessage.error(t('export.download_failed') || '无法构造下载链接')
    return
  }

  downloadingIds.value.add(exportId)

  try {
    let fullUrl = url
    if (url.startsWith('/')) {
      const apiBaseURL = import.meta.env.VITE_API_BASE_URL
      const apiPrefix = import.meta.env.VITE_API_PREFIX || '/api/admin'
      if (apiBaseURL) {
        const base = apiBaseURL.replace(/\/+$/, '')
        const cleanUrl = url.replace(/^\/api\/admin/, '')
        fullUrl = `${base}${apiPrefix}${cleanUrl}`
      }
    }

    const response = await fetch(fullUrl, {
      method: 'GET',
      headers: buildAdminAuthHeaders()
    })

    if (!response.ok) {
      if (response.status === 401) {
        ElMessage.error(t('error.unauthorized') || '未授权，请重新登录')
      } else {
        throw new Error(`HTTP error! status: ${response.status}`)
      }
      return
    }

    const contentType = response.headers.get('content-type') || ''
    if (contentType.includes('text/html')) {
      ElMessage.error(t('export.download_failed') || '下载失败：服务器返回了错误内容')
      return
    }

    const blob = await response.blob()
    const contentDisposition = response.headers.get('content-disposition') || ''
    let filename = row.filename || 'export.csv'
    const filenameMatch = contentDisposition.match(/filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/)
    if (filenameMatch?.[1]) {
      filename = filenameMatch[1].replace(/['"]/g, '')
      try {
        filename = decodeURIComponent(filename)
      } catch {
        // keep original filename
      }
    }

    const downloadUrl = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = downloadUrl
    link.download = filename
    link.style.display = 'none'
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    window.URL.revokeObjectURL(downloadUrl)
    ElMessage.success(t('export.download_success') || '下载成功')
  } catch (error) {
    if (!error.__handled) {
      ElMessage.error(error.response?.data?.message || error.message || t('export.download_failed') || '下载失败')
    }
  } finally {
    setTimeout(() => {
      downloadingIds.value.delete(exportId)
    }, 2000)
  }
}

const handleDelete = (row) => handleDeleteRow(row, loadData)

const handleBatchDelete = () => {
  handleBatchDeleteRows(selectedRows.value, () => {
    handleSelectionChange([])
    loadData()
  })
}

onActivated(() => {
  loadData()
})
</script>
