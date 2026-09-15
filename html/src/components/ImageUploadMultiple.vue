<template>
  <div class="image-upload-multiple">
    <el-upload
      :action="uploadAction"
      :headers="uploadHeaders"
      :data="uploadData"
      :before-upload="beforeUpload"
      :on-success="handleUploadSuccess"
      :on-error="handleUploadError"
      :on-remove="handleRemove"
      :on-preview="handlePreview"
      :file-list="fileList"
      :multiple="true"
      :limit="limit"
      list-type="picture-card"
      accept="image/*"
    >
      <el-icon><Plus /></el-icon>
    </el-upload>

    <el-dialog v-model="previewVisible" width="520px">
      <img :src="previewImage" alt="preview" style="width: 100%" />
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, ref, watch, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import { buildAdminAuthHeaders } from '@/utils/authHeaders'

const props = defineProps({
  modelValue: {
    type: Array,
    default: () => []
  },
  limit: {
    type: Number,
    default: 9
  },
  minCount: {
    type: Number,
    default: 0
  },
  maxSizeMB: {
    type: Number,
    default: 10
  }
})

const emit = defineEmits(['update:modelValue', 'change'])

const previewVisible = ref(false)
const previewImage = ref('')
const fileList = ref([])
const isInternalSyncing = ref(false)

const normalizeUrl = (url) => {
  if (!url) return ''
  if (url.startsWith('http') || url.startsWith('blob:') || url.startsWith('data:')) return url

  const apiBaseURL = import.meta.env.VITE_API_BASE_URL
  const apiPrefix = import.meta.env.VITE_API_PREFIX || '/api/admin'
  if (apiBaseURL) {
    const base = apiBaseURL.replace(/\/+$/, '')
    const prefix = apiPrefix.startsWith('/') ? apiPrefix : `/${apiPrefix}`
    if (url.startsWith(prefix)) return `${base}${url}`
    return `${base}${prefix}${url.startsWith('/') ? '' : '/'}${url}`
  }

  const prefix = apiPrefix.startsWith('/') ? apiPrefix : `/${apiPrefix}`
  if (url.startsWith(prefix)) return url
  return `${prefix}${url.startsWith('/') ? '' : '/'}${url}`
}

const isTransientUrl = (url = '') => String(url).startsWith('blob:') || String(url).startsWith('data:')

const toSubmitUrl = (url = '') => {
  const value = String(url || '').trim()
  if (!value || isTransientUrl(value)) return ''
  if (value.startsWith('/')) return value

  if (value.startsWith('http')) {
    const apiPrefix = import.meta.env.VITE_API_PREFIX || '/api/admin'
    const normalizedPrefix = apiPrefix.startsWith('/') ? apiPrefix : `/${apiPrefix}`
    try {
      const parsed = new URL(value)
      if (parsed.pathname.startsWith(normalizedPrefix)) {
        return `${parsed.pathname}${parsed.search || ''}`
      }
      return value
    } catch {
      return value
    }
  }

  return value
}

const syncFileList = (urls) => {
  const list = Array.isArray(urls) ? urls : []
  fileList.value = list
    .filter(Boolean)
    .map((url, idx) => {
      const rawUrl = String(url)
      const normalized = normalizeUrl(rawUrl)
      return {
        uid: `init-${idx}-${rawUrl}`,
        name: `image-${idx + 1}`,
        url: normalized,
        rawUrl
      }
    })
}

watch(
  () => props.modelValue,
  (val) => {
    if (isInternalSyncing.value) {
      isInternalSyncing.value = false
      return
    }
    syncFileList(val)
  },
  { immediate: true, deep: true }
)

const uploadAction = computed(() => {
  const apiBaseURL = import.meta.env.VITE_API_BASE_URL
  const apiPrefix = import.meta.env.VITE_API_PREFIX || '/api/admin'
  if (apiBaseURL) {
    const base = apiBaseURL.replace(/\/+$/, '')
    const prefix = apiPrefix.startsWith('/') ? apiPrefix : `/${apiPrefix}`
    return `${base}${prefix}/attachments/upload`
  }
  return `${apiPrefix}/attachments/upload`
})

const uploadHeaders = computed(() => buildAdminAuthHeaders())

const uploadData = computed(() => ({}))

const buildUrlsFromUploadFiles = (uploadFiles = []) => {
  return (Array.isArray(uploadFiles) ? uploadFiles : [])
    .map((f) => {
      const raw = f?.rawUrl || f?.response?.data?.file_url || f?.response?.data?.preview_url || f?.url || ''
      return String(raw || '')
    })
    .filter(Boolean)
}

const emitUrls = (urls) => {
  const finalUrls = (Array.isArray(urls) ? urls : fileList.value.map((f) => f.rawUrl || f.url))
    .filter(Boolean)
    .map((url) => toSubmitUrl(url))
    .filter((url) => !isTransientUrl(url))
  isInternalSyncing.value = true
  emit('update:modelValue', finalUrls)
  emit('change', finalUrls)
  nextTick(() => {
    isInternalSyncing.value = false
  })
}

const syncFileListFromUploadFiles = (uploadFiles = []) => {
  fileList.value = (Array.isArray(uploadFiles) ? uploadFiles : [])
    .map((f) => {
      const raw = f?.rawUrl || f?.response?.data?.file_url || f?.response?.data?.preview_url || f?.url || ''
      return {
        uid: f?.uid,
        name: f?.name || '',
        url: normalizeUrl(String(raw || '')),
        rawUrl: String(raw || '')
      }
    })
    .filter((f) => !!f.url)
    .map((f, idx) => ({
      uid: f.uid || `upload-${idx}-${f.rawUrl}`,
      name: f.name || `image-${idx + 1}`,
      url: f.url,
      rawUrl: f.rawUrl
    }))
}

const beforeUpload = (file) => {
  if (!file.type.startsWith('image/')) {
    ElMessage.error('只能上传图片文件')
    return false
  }
  if (file.size > props.maxSizeMB * 1024 * 1024) {
    ElMessage.error(`图片大小不能超过 ${props.maxSizeMB}MB`)
    return false
  }
  return true
}

const handleUploadSuccess = (response, uploadFile, uploadFiles) => {
  if (response?.code !== 200 || !response?.data) {
    ElMessage.error(response?.message || '上传失败')
    return
  }
  const raw = response.data.file_url || response.data.preview_url || ''
  const url = normalizeUrl(raw)
  uploadFile.url = url
  uploadFile.rawUrl = String(raw || '')

  const nextUploadFiles = (Array.isArray(uploadFiles) ? uploadFiles : []).map((f) => {
    if (f.uid === uploadFile.uid) return { ...f, url, rawUrl: String(raw || '') }
    return f
  })

  syncFileListFromUploadFiles(nextUploadFiles)
  const persistentUrls = buildUrlsFromUploadFiles(nextUploadFiles)
    .map((item) => toSubmitUrl(item))
    .filter((item) => !isTransientUrl(item))
    .slice(0, props.limit)
  emitUrls([...new Set(persistentUrls)])
  ElMessage.success('上传成功')
}

const handleUploadError = () => {
  ElMessage.error('上传失败')
}

const handleRemove = (uploadFile, uploadFiles) => {
  syncFileListFromUploadFiles(uploadFiles)
  emitUrls(buildUrlsFromUploadFiles(uploadFiles))

  const currentCount = Array.isArray(uploadFiles) ? uploadFiles.filter((f) => !!f?.url).length : 0
  if (props.minCount > 0 && currentCount < props.minCount) {
    ElMessage.warning(`至少保留 ${props.minCount} 张图片`)
  }
}

const handlePreview = (file) => {
  previewImage.value = file.url || ''
  previewVisible.value = true
}
</script>

