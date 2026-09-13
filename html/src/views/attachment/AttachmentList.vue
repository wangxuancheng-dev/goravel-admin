<template>
  <div class="attachment-list">
    <ListPage
      ref="listPageRef"
      page-class="attachment"
      :title="$t('menu.attachment')"
      :show-add-button="false"
      :search-form="searchForm"
      :search-fields="searchFields"
      :initial-search-values="attachmentInitialSearchForm"
      :table-data="tableData"
      :loading="loading"
      :table-columns="tableColumns"
      :table-key="`table-${tableColumns.length}-${JSON.stringify(columnOrder)}`"
      :pagination="pagination"
      show-toolbar
      show-column-setting
      :visible-columns="visibleColumns"
      :all-columns="allColumns"
      :default-visible-columns="defaultVisibleColumns"
      :column-order="columnOrder"
      :fixed-columns="fixedColumns"
      :on-column-setting-confirm="handleColumnSettingConfirm"
      @search="handleSearch"
      @reset="handleReset"
      @refresh="loadData"
      @page-change="loadData"
      @sort-change="handleSortChange"
      @selection-change="handleSelectionChange"
    >
      <template #header-actions>
        <el-upload
          ref="uploadRef"
          :action="uploadAction"
          :headers="uploadHeaders"
          :data="uploadData"
          :before-upload="beforeUpload"
          :on-success="handleUploadSuccess"
          :on-error="handleUploadError"
          :show-file-list="false"
          :multiple="false"
        >
          <template #trigger>
            <el-button type="primary" :disabled="getButtonState('attachment.upload').disabled">
              <el-icon><UploadIcon /></el-icon>
              {{ $t('attachment.upload') }}
            </el-button>
          </template>
        </el-upload>
        <el-button
          type="warning"
          :disabled="getButtonState('attachment.upload').disabled"
          @click="handleCropUpload"
        >
          <el-icon><CropIcon /></el-icon>
          {{ $t('attachment.crop_upload') }}
        </el-button>
        <el-button
          type="success"
          :disabled="getButtonState('attachment.chunk').disabled"
          @click="handleLargeFileUpload"
        >
          <el-icon><UploadIcon /></el-icon>
          {{ $t('attachment.large_file_upload') }}
        </el-button>
        <el-button
          type="info"
          :disabled="getButtonState('attachment_category.index').disabled"
          @click="categoryDialogVisible = true"
        >
          {{ $t('attachment.category_manage') }}
        </el-button>
        <el-button
          type="danger"
          :disabled="selectedRows.length === 0 || getButtonState('attachment.destroy').disabled"
          @click="handleBatchDelete"
        >
          <el-icon><DeleteIcon /></el-icon>
          {{ $t('common.delete_selected') }} ({{ selectedRows.length }})
        </el-button>
      </template>

      <template #filename="{ row }">
          <div class="filename-cell">
            <el-image
              v-if="row.file_type === 'image' && getImageUrl(row)"
              :src="getImageUrl(row)"
              :preview-src-list="[getImageUrl(row)]"
              fit="cover"
              class="filename-thumbnail"
              :preview-teleported="true"
              :lazy="true"
              @load="handleImageLoad(row)"
              @error="handleImageError(row)"
            >
              <template #placeholder>
                <div class="image-placeholder">
                  <el-icon class="is-loading"><Loading /></el-icon>
                </div>
              </template>
              <template #error>
                <div class="image-error">
                  <el-icon><Picture /></el-icon>
                </div>
              </template>
            </el-image>
            <div
              v-else-if="row.file_type === 'image' && getImageLoadingState(row) === 'loading'"
              class="image-placeholder"
            >
              <el-icon class="is-loading"><Loading /></el-icon>
            </div>
            <div
              v-else-if="row.file_type === 'image' && getImageLoadingState(row) === 'error'"
              class="image-error"
            >
              <el-icon><Picture /></el-icon>
            </div>
            <span class="filename-text">{{ row.filename }}</span>
          </div>
        </template>

        <template #display_name="{ row }">
          <el-input
            v-model="row.display_name"
            :placeholder="$t('attachment.display_name_placeholder')"
            size="small"
            @blur="handleUpdateDisplayName(row)"
            @keyup.enter="handleUpdateDisplayName(row)"
          />
        </template>

        <template #category="{ row }">
          <el-select
            v-model="row.category_id"
            size="small"
            style="width: 140px"
            :disabled="getButtonState('attachment.update_category').disabled"
            @change="handleUpdateCategory(row)"
          >
            <el-option
              v-for="opt in categoryOptions"
              :key="opt.value"
              :label="opt.label"
              :value="opt.value"
            />
          </el-select>
        </template>

        <template #is_public="{ row }">
          <el-switch
            v-model="row.is_public"
            :active-value="1"
            :inactive-value="0"
            :disabled="getButtonState('attachment.update_visibility').disabled"
            inline-prompt
            :active-text="$t('attachment.visibility_public_short')"
            :inactive-text="$t('attachment.visibility_private_short')"
            @change="handleUpdateVisibility(row)"
          />
        </template>

        <template #file_type="{ row }">
          <el-tag :type="getFileTypeTagType(row.file_type)">
            {{ getFileTypeLabel(row.file_type) }}
          </el-tag>
        </template>

        <template #disk="{ row }">
          <el-tag size="small" type="info">
            {{ row.disk || '-' }}
          </el-tag>
        </template>

        <template #operation="{ row }">
          <el-button
            v-if="canInlinePreviewAttachment(row)"
            type="primary"
            link
            :loading="previewLoading && previewRow?.id === row.id"
            @click="handlePreview(row)"
          >
            {{ $t('attachment.preview') }}
          </el-button>
          <el-button
            type="success"
            link
            :disabled="downloadingIds.has(row.id)"
            :loading="downloadingIds.has(row.id)"
            @click="handleDownload(row)"
          >
            {{ $t('common.download') }}
          </el-button>
          <el-button
            type="danger"
            link
            :disabled="getButtonState('attachment.destroy').disabled"
            @click="handleDelete(row)"
          >
            {{ $t('common.delete') }}
          </el-button>
        </template>
    </ListPage>

    <el-dialog
      v-model="previewVisible"
      :title="previewTitle"
      width="860px"
      append-to-body
      destroy-on-close
      @closed="handlePreviewClosed"
    >
      <div v-loading="previewLoading" class="preview-container">
        <div v-if="previewKind === 'image' && previewBlobUrl" class="preview-image">
          <img :src="previewBlobUrl" :alt="previewTitle" style="max-width: 100%; max-height: 70vh;" />
        </div>
        <div v-else-if="previewKind === 'video' && previewBlobUrl" class="preview-video">
          <video :src="previewBlobUrl" controls style="max-width: 100%; max-height: 70vh;" />
        </div>
        <div v-else-if="previewKind === 'pdf' && previewBlobUrl" class="preview-document">
          <iframe :src="previewBlobUrl" style="width: 100%; height: 70vh; border: none;" />
        </div>
        <div v-else-if="previewKind === 'text'" class="preview-text">
          <pre>{{ previewText }}</pre>
        </div>
        <el-empty
          v-else-if="!previewLoading"
          :description="$t('attachment.preview_unsupported')"
        />
      </div>
    </el-dialog>

    <AttachmentCategoryDialog
      v-model="categoryDialogVisible"
      @changed="handleCategoryChanged"
    />

    <!-- 大文件上传对话框 -->
    <el-dialog
      v-model="chunkUploadVisible"
      :title="$t('attachment.chunk_upload')"
      width="600px"
      :close-on-click-modal="false"
      :close-on-press-escape="false"
    >
      <div v-if="chunkUploadFile" class="chunk-upload-container">
        <div class="upload-info">
          <p><strong>{{ $t('attachment.filename') }}:</strong> {{ chunkUploadFile.name }}</p>
          <p><strong>{{ $t('attachment.size') }}:</strong> {{ formatFileSize(chunkUploadFile.size) }}</p>
        </div>
        <el-progress 
          :percentage="Math.round(chunkUploadProgress)" 
          :status="chunkUploadStatus"
          :stroke-width="20"
        />
        <div class="upload-status">
          <p v-if="chunkUploadStatus === 'success'">{{ $t('attachment.upload_success') }}</p>
          <p v-else-if="chunkUploadStatus === 'exception'">{{ $t('attachment.upload_failed') }}</p>
          <p v-else>{{ $t('attachment.uploading') }}: {{ Math.round(chunkUploadProgress) }}%</p>
        </div>
        <div class="upload-actions" style="margin-top: 20px; text-align: right;">
          <el-button 
            v-if="chunkUploadStatus !== 'success' && chunkUploadStatus !== 'exception'"
            @click="handleCancelChunkUpload"
          >
            {{ $t('common.cancel') }}
          </el-button>
          <el-button 
            v-if="chunkUploadStatus === 'success'"
            type="primary"
            @click="handleChunkUploadClose"
          >
            {{ $t('common.confirm') }}
          </el-button>
          <el-button 
            v-if="chunkUploadStatus === 'exception'"
            type="primary"
            @click="handleRetryChunkUpload"
          >
            {{ $t('common.retry') }}
          </el-button>
        </div>
      </div>
    </el-dialog>
    <!-- 图片裁剪对话框 -->
    <el-dialog
      v-model="cropDialogVisible"
      :title="$t('attachment.crop_upload')"
      width="800px"
      :close-on-click-modal="false"
      destroy-on-close
    >
      <div class="crop-container" style="height: 500px;">
        <vue-cropper
          ref="cropperRef"
          :img="cropOption.img"
          :output-size="cropOption.size"
          :output-type="cropOption.outputType"
          :info="true"
          :full="cropOption.full"
          :can-move="cropOption.canMove"
          :can-move-box="cropOption.canMoveBox"
          :fixed-box="cropOption.fixedBox"
          :original="cropOption.original"
          :auto-crop="cropOption.autoCrop"
          :auto-crop-width="cropOption.autoCropWidth"
          :auto-crop-height="cropOption.autoCropHeight"
          :center-box="cropOption.centerBox"
          :high="cropOption.high"
          :mode="cropOption.mode"
        />
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-upload
            action=""
            :auto-upload="false"
            :show-file-list="false"
            accept="image/*"
            :on-change="onCropFileChange"
            style="display: inline-block; margin-right: 10px;"
          >
            <el-button>{{ $t('attachment.select_image') }}</el-button>
          </el-upload>
          <el-button @click="cropDialogVisible = false">{{ $t('common.cancel') }}</el-button>
          <el-button type="primary" :loading="cropUploading" @click="handleCropConfirm">
            {{ $t('common.confirm') }}
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, onBeforeUnmount, markRaw, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Upload, Delete, Loading, Picture, Crop } from '@element-plus/icons-vue'
import ListPage from '@/components/ListPage.vue'
import { useListPage } from '@/composables/useListPage'
import { usePermission } from '@/composables/usePermission'
import { useCrud } from '@/composables/useCrud'
import { useColumnSetting } from '@/composables/useColumnSetting'
import { useAttachmentImagePreview } from '@/composables/useAttachmentImagePreview'
import { useAttachmentChunkUpload, useAttachmentUploadConfig } from '@/composables/useAttachmentChunkUpload'
import axios from 'axios'
import {
  getAttachmentList,
  deleteAttachment,
  batchDeleteAttachments,
  updateDisplayName,
  updateCategory,
  updateVisibility,
  getAttachmentPreviewUrl
} from '@/api/attachment'
import AttachmentCategoryDialog from './AttachmentCategoryDialog.vue'
import request from '@/utils/request'
import i18n from '@/i18n'
import Storage from '@/utils/storage'
import {
  canInlinePreviewAttachment,
  attachmentPreviewKind
} from '@/utils/attachmentUrl'
import 'vue-cropper/dist/index.css'
import { VueCropper } from 'vue-cropper'
import {
  attachmentInitialSearchForm,
  transformAttachmentRow,
  createAttachmentSearchFields,
  createAttachmentTableColumns,
  formatAttachmentFileSize,
  getAttachmentFileTypeTagType,
  getAttachmentFileTypeLabel
} from './attachment.config'

const UploadIcon = markRaw(Upload)
const DeleteIcon = markRaw(Delete)
const CropIcon = markRaw(Crop)

const { t, locale } = useI18n()
const { getButtonState } = usePermission()
const categoryDialogVisible = ref(false)
const categoryOptions = ref([])

const loadCategoryOptions = async () => {
  try {
    const res = await request({ url: '/options', method: 'get', params: { type: 'attachment_category' } })
    categoryOptions.value = res?.data?.options || []
  } catch {
    categoryOptions.value = []
  }
}

const handleCategoryChanged = async () => {
  await loadCategoryOptions()
  loadData()
}

const handleUpdateCategory = async (row) => {
  const attachmentId = row.id
  if (!attachmentId || !row.category_id) return
  try {
    await updateCategory(attachmentId, row.category_id)
    ElMessage.success(t('attachment.update_success'))
  } catch (error) {
    console.error('Update category error:', error)
    if (!error.__handled) {
      ElMessage.error(error.response?.data?.message || error.message || t('attachment.update_failed'))
    }
    loadData()
  }
}

const handleUpdateVisibility = async (row) => {
  const attachmentId = row.id
  if (!attachmentId) return
  try {
    const res = await updateVisibility(attachmentId, Number(row.is_public) === 1)
    if (res?.data?.file_url) {
      row.file_url = res.data.file_url
    }
    ElMessage.success(t('attachment.update_success'))
  } catch (error) {
    console.error('Update visibility error:', error)
    if (!error.__handled) {
      ElMessage.error(error.response?.data?.message || error.message || t('attachment.update_failed'))
    }
    loadData()
  }
}

const { handleDelete: handleDeleteCrud, handleBatchDelete: handleBatchDeleteCrud } = useCrud({
  deleteApi: deleteAttachment,
  batchDeleteApi: batchDeleteAttachments
})

const listPageRef = ref(null)
const uploadRef = ref(null)
const downloadingIds = ref(new Set())
const previewVisible = ref(false)
const previewLoading = ref(false)
const previewRow = ref(null)
const previewKind = ref('')
const previewBlobUrl = ref('')
const previewText = ref('')
const previewTitle = computed(() => {
  const row = previewRow.value
  if (!row) return t('attachment.preview')
  return row.display_name || row.filename || t('attachment.preview')
})

const revokePreviewBlob = () => {
  if (previewBlobUrl.value) {
    window.URL.revokeObjectURL(previewBlobUrl.value)
    previewBlobUrl.value = ''
  }
  previewText.value = ''
  previewKind.value = ''
  previewRow.value = null
}

const handlePreviewClosed = () => {
  revokePreviewBlob()
  previewLoading.value = false
}

const handlePreview = async (row) => {
  if (!canInlinePreviewAttachment(row)) {
    ElMessage.warning(t('attachment.preview_unsupported'))
    return
  }
  const kind = attachmentPreviewKind(row)
  if (!kind) {
    ElMessage.warning(t('attachment.preview_unsupported'))
    return
  }

  revokePreviewBlob()
  previewRow.value = row
  previewKind.value = kind
  previewVisible.value = true
  previewLoading.value = true

  try {
    const previewUrl = getAttachmentPreviewUrl(row.id)
    const token = Storage.getItem('token', '') || ''
    const tokenStr = typeof token === 'string' ? token.trim() : ''
    const currentLocale = locale.value || i18n.global.locale.value || Storage.getItem('language', 'zh-CN') || 'zh-CN'
    const acceptLanguage = currentLocale === 'en-US' ? 'en-US' : 'zh-CN'

    const response = await fetch(previewUrl, {
      method: 'GET',
      headers: {
        Authorization: `Bearer ${tokenStr}`,
        'Accept-Language': acceptLanguage
      }
    })

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }

    const contentType = response.headers.get('content-type') || ''
    if (contentType.includes('text/html') && kind !== 'text') {
      throw new Error('invalid preview content')
    }

    const blob = await response.blob()
    if (kind === 'text') {
      previewText.value = await blob.text()
    } else {
      previewBlobUrl.value = window.URL.createObjectURL(blob)
    }
  } catch (error) {
    console.error('Preview error:', error)
    previewVisible.value = false
    revokePreviewBlob()
    if (!error.__handled) {
      ElMessage.error(t('attachment.preview_failed'))
    }
  } finally {
    previewLoading.value = false
  }
}

// 裁剪上传相关
const cropDialogVisible = ref(false)
const cropperRef = ref(null)
const cropUploading = ref(false)
const cropFileName = ref('')
const cropOption = reactive({
  img: '',
  size: 1,
  full: false,
  outputType: 'png',
  canMove: true,
  fixedBox: false,
  original: false,
  canMoveBox: true,
  autoCrop: true,
  // 只有自动截图开启 宽度高度才生效
  autoCropWidth: 200,
  autoCropHeight: 200,
  centerBox: false,
  high: true,
  max: 99999,
  mode: 'contain'
})

// 图片预览
const {
  loadImageAsBlob,
  getImageUrl,
  getImageLoadingState,
  handleImageLoad,
  handleImageError
} = useAttachmentImagePreview()

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
  initDefaultSort
} = useListPage({
  fetchApi: getAttachmentList,
  initialSearchForm: attachmentInitialSearchForm,
  defaultSort: 'id:desc',
  normalizeRows: false,
  transformData: transformAttachmentRow,
  tableRef: computed(() => listPageRef.value?.tableRef?.tableRef),
  onLoadSuccess: () => {
    tableData.value.forEach((row) => {
      if (row.file_type === 'image') {
        nextTick(() => loadImageAsBlob(row))
      }
    })
  }
})

const searchFields = computed(() => createAttachmentSearchFields(t))
const allTableColumns = computed(() => createAttachmentTableColumns(t))

const {
  tableColumns,
  visibleColumns,
  allColumns,
  defaultVisibleColumns,
  columnOrder,
  fixedColumns,
  handleColumnSettingConfirm
} = useColumnSetting('attachment', allTableColumns)

const formatFileSize = formatAttachmentFileSize
const getFileTypeTagType = getAttachmentFileTypeTagType
const getFileTypeLabel = (fileType) => getAttachmentFileTypeLabel(t, fileType)

const { uploadAction, uploadHeaders, uploadData } = useAttachmentUploadConfig(locale)

const {
  chunkUploadVisible,
  chunkUploadFile,
  chunkUploadProgress,
  chunkUploadStatus,
  beforeUpload,
  handleLargeFileUpload,
  handleCancelChunkUpload,
  handleChunkUploadClose,
  handleRetryChunkUpload
} = useAttachmentChunkUpload({ onUploaded: loadData })

const handleUploadSuccess = () => {
  ElMessage.success(t('attachment.upload_success'))
  loadData()
}

const handleUploadError = () => {
  ElMessage.error(t('attachment.upload_failed'))
}

// 裁剪上传相关方法
const handleCropUpload = () => {
  cropDialogVisible.value = true
  cropOption.img = ''
  cropFileName.value = ''
}

const onCropFileChange = (file) => {
  const isImage = file.raw.type.startsWith('image/')
  if (!isImage) {
    ElMessage.error(t('attachment.only_image_allowed'))
    return
  }
  
  // 检查文件大小（限制10MB）
  const maxSize = 10 * 1024 * 1024
  if (file.size > maxSize) {
    ElMessage.error(t('attachment.file_too_large'))
    return
  }

  cropFileName.value = file.name
  // 读取图片
  const reader = new FileReader()
  reader.onload = (e) => {
    cropOption.img = e.target.result
  }
  reader.readAsDataURL(file.raw)
}

const handleCropConfirm = () => {
  if (!cropOption.img) {
    ElMessage.warning(t('attachment.please_select_image'))
    return
  }

  cropUploading.value = true
  cropperRef.value.getCropBlob((blob) => {
    if (!blob) {
      cropUploading.value = false
      return
    }

    // 创建 File 对象
    const file = new File([blob], cropFileName.value || 'cropped-image.png', {
      type: blob.type
    })

    const formData = new FormData()
    formData.append('file', file)

    // 获取上传 URL 和 Headers
    const action = uploadAction.value
    const headers = uploadHeaders.value
    
    // 使用 axios 上传
    axios.post(action, formData, {
      headers: {
        ...headers,
        'Content-Type': 'multipart/form-data'
      }
    }).then(() => {
      ElMessage.success(t('attachment.upload_success'))
      cropDialogVisible.value = false
      loadData()
    }).catch((error) => {
      console.error('Crop upload error:', error)
      ElMessage.error(t('attachment.upload_failed'))
    }).finally(() => {
      cropUploading.value = false
    })
  })
}

const handleUpdateDisplayName = async (row) => {
  const attachmentId = row.id
  if (!attachmentId) return
  
  const displayName = row.display_name || ''
  
  try {
    await updateDisplayName(attachmentId, displayName)
    ElMessage.success(t('attachment.update_success'))
  } catch (error) {
    console.error('Update display name error:', error)
    if (!error.__handled) {
      const errorMessage = error.response?.data?.message || error.message || t('attachment.update_failed')
      ElMessage.error(errorMessage)
    }
    // 重新加载数据以恢复原值
    loadData()
  }
}

const handleDownload = async (row) => {
  // 防止重复点击
  const attachmentId = row.id
  if (downloadingIds.value.has(attachmentId)) {
    return // 如果正在下载，直接返回
  }
  
  // 标记为正在下载
  downloadingIds.value.add(attachmentId)
  
  try {
    // 构建下载 URL
    const apiBaseURL = import.meta.env.VITE_API_BASE_URL
    const apiPrefix = import.meta.env.VITE_API_PREFIX || '/api/admin'
    
    let downloadUrl = `${apiPrefix}/attachments/${attachmentId}/download`
    
    if (apiBaseURL) {
      const base = apiBaseURL.replace(/\/+$/, '')
      downloadUrl = `${base}${downloadUrl}`
    }
    
    // 获取 token
    const token = Storage.getItem('token', '') || ''
    const tokenStr = typeof token === 'string' ? token.trim() : ''
    
    // 获取当前语言设置
    const currentLocale = locale.value || i18n.global.locale.value || Storage.getItem('language', 'zh-CN') || 'zh-CN'
    const acceptLanguage = currentLocale === 'en-US' ? 'en-US' : 'zh-CN'
    
    // 使用 fetch 请求下载文件，这样可以携带认证 token
    const response = await fetch(downloadUrl, {
      method: 'GET',
      headers: {
        'Authorization': `Bearer ${tokenStr}`,
        'Accept-Language': acceptLanguage
      }
    })
    
    if (!response.ok) {
      if (response.status === 401) {
        ElMessage.error(t('error.unauthorized') || '未授权，请重新登录')
      } else {
        const errorText = await response.text()
        console.error('Download error response:', errorText)
        throw new Error(`HTTP error! status: ${response.status}`)
      }
      return
    }
    
    // 检查响应类型，确保是文件而不是 HTML
    const contentType = response.headers.get('content-type') || ''
    if (contentType.includes('text/html')) {
      const htmlContent = await response.text()
      console.error('Received HTML instead of file:', htmlContent.substring(0, 200))
      ElMessage.error(t('attachment.download_failed') || '下载失败：服务器返回了错误内容')
      return
    }
    
    // 获取文件内容
    const blob = await response.blob()
    
    // 从响应头获取文件名，如果没有则使用记录中的文件名
    const contentDisposition = response.headers.get('content-disposition') || ''
    let filename = row.filename || row.Filename || 'attachment'
    
    // 尝试从 Content-Disposition 头中提取文件名
    const filenameMatch = contentDisposition.match(/filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/)
    if (filenameMatch && filenameMatch[1]) {
      filename = filenameMatch[1].replace(/['"]/g, '')
      // 处理 URL 编码的文件名
      try {
        filename = decodeURIComponent(filename)
      } catch (e) {
        // 如果解码失败，使用原始文件名
      }
    }
    
    // 创建下载链接
    const downloadUrlObj = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = downloadUrlObj
    link.download = filename
    link.style.display = 'none'
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
    
    // 释放 URL 对象
    window.URL.revokeObjectURL(downloadUrlObj)
    
    ElMessage.success(t('attachment.download_success') || '下载成功')
  } catch (error) {
    console.error('Download error:', error)
    if (!error.__handled) {
      const errorMessage = error.response?.data?.message || error.message || t('attachment.download_failed') || '下载失败'
      ElMessage.error(errorMessage)
    }
  } finally {
    // 下载完成或失败后，延迟移除标记（防止短时间内重复点击）
    setTimeout(() => {
      downloadingIds.value.delete(attachmentId)
    }, 2000) // 2秒内不允许重复点击
  }
}

const handleDelete = (row) => handleDeleteCrud(row, loadData)

const handleBatchDelete = () => {
  handleBatchDeleteCrud(selectedRows.value, () => {
    handleSelectionChange([])
    loadData()
  })
}

onMounted(() => {
  initDefaultSort()
  loadCategoryOptions()
  loadData()
})

onBeforeUnmount(() => {
  revokePreviewBlob()
})

// 刷新由菜单的「是否缓存」设置控制：no_cache=1 时每次进入会 remount 触发 onMounted 刷新
</script>

<style scoped>


.preview-container {
  text-align: center;
}

.preview-image img {
  border-radius: var(--border-radius-sm);
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
}

.preview-video video {
  border-radius: var(--border-radius-sm);
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
}

.preview-document iframe {
  border-radius: var(--border-radius-sm);
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
}

.preview-text pre {
  max-height: 70vh;
  overflow: auto;
  text-align: left;
  white-space: pre-wrap;
  word-break: break-word;
  margin: 0;
  padding: 12px;
  background: var(--bg-color-tertiary, #f5f7fa);
  border-radius: var(--border-radius-sm);
  font-size: 13px;
  line-height: 1.5;
}

.chunk-upload-container {
  padding: 20px;
}

.upload-info {
  margin-bottom: 20px;
}

.upload-info p {
  margin: 8px 0;
}

.upload-status {
  margin-top: var(--space-sm);
  text-align: center;
  color: var(--text-color-regular);
}

.upload-actions {
  margin-top: 20px;
}

.filename-cell {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
}

.filename-thumbnail {
  width: 50px;
  height: 50px;
  cursor: pointer;
  border-radius: var(--border-radius-sm);
  flex-shrink: 0;
  border: 1px solid var(--border-color-light);
}

.image-placeholder {
  width: 50px;
  height: 50px;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: var(--bg-color-tertiary);
  border-radius: var(--border-radius-sm);
  border: 1px solid var(--border-color-light);
}

.image-placeholder .el-icon {
  font-size: 20px;
  color: var(--text-color-secondary);
}

.image-error {
  width: 50px;
  height: 50px;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: #fef0f0;
  border-radius: var(--border-radius-sm);
  border: 1px solid #fde2e2;
}

.image-error .el-icon {
  font-size: 20px;
  color: #f56c6c;
}

.filename-text {
  flex: 1;
  word-break: break-all;
}
</style>

