import { ref, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import Storage from '@/utils/storage'
import { buildAdminAuthHeaders } from '@/utils/authHeaders'
import {
  initChunkUpload,
  uploadChunk,
  mergeChunks,
  getChunkProgress
} from '@/api/attachment'
import { ATTACHMENT_CHUNK_SIZE, ATTACHMENT_LARGE_FILE_THRESHOLD } from '@/views/attachment/attachment.config'

/**
 * Chunked/large file upload dialog state and handlers.
 */
export function useAttachmentChunkUpload({ onUploaded }) {
  const { t } = useI18n()

  const chunkUploadVisible = ref(false)
  const chunkUploadFile = ref(null)
  const chunkUploadProgress = ref(0)
  const chunkUploadStatus = ref('')
  const chunkUploadChunkID = ref('')
  const chunkUploadChunks = ref([])
  const chunkUploadCancelled = ref(false)

  const CHUNK_SIZE = ATTACHMENT_CHUNK_SIZE
  const LARGE_FILE_THRESHOLD = ATTACHMENT_LARGE_FILE_THRESHOLD

  const handleLargeFileUpload = () => {
    const input = document.createElement('input')
    input.type = 'file'
    input.onchange = (e) => {
      const file = e.target.files?.[0]
      if (file) {
        handleChunkUpload(file, true)
      }
    }
    input.click()
  }

  const beforeUpload = (file) => {
    const maxSize = 100 * 1024 * 1024
    if (file.size > maxSize) {
      ElMessage.error(t('attachment.file_too_large'))
      return false
    }
    if (file.size > LARGE_FILE_THRESHOLD) {
      handleChunkUpload(file, false)
      return false
    }
    return true
  }

  const handleChunkUpload = async (file, _isLargeFileButton = false, useExistingChunkID = false) => {
    chunkUploadFile.value = file
    chunkUploadVisible.value = true
    chunkUploadProgress.value = 0
    chunkUploadStatus.value = ''
    chunkUploadCancelled.value = false

    try {
      const totalSize = file.size
      if (!totalSize || totalSize <= 0) {
        ElMessage.error(t('attachment.invalid_file_size'))
        chunkUploadVisible.value = false
        chunkUploadFile.value = null
        return
      }

      const totalChunks = Math.ceil(totalSize / CHUNK_SIZE)
      if (!totalChunks || !isFinite(totalChunks)) {
        ElMessage.error(t('attachment.invalid_chunk_calculation'))
        chunkUploadVisible.value = false
        chunkUploadFile.value = null
        return
      }

      if (!useExistingChunkID || !chunkUploadChunkID.value) {
        try {
          const initRes = await initChunkUpload(file.name, totalSize, CHUNK_SIZE, totalChunks)
          chunkUploadChunkID.value = initRes.data.chunk_id
          try {
            Storage.setItem(`chunk_${chunkUploadChunkID.value}`, {
              filename: file.name,
              total_size: totalSize,
              chunk_size: CHUNK_SIZE,
              total_chunks: totalChunks,
              created_at: Date.now()
            })
          } catch {
            // ignore storage errors
          }
        } catch (error) {
          const errorCode = error.errorCode || error.response?.data?.error_code || ''
          if (!error.__handled) {
            ElMessage.error(error.response?.data?.message || error.message || t('common.operation_failed'))
          }
          if (errorCode === 'chunk_upload_only_local_storage') {
            chunkUploadVisible.value = false
            chunkUploadFile.value = null
          }
          throw error
        }
      }

      let uploadedChunksSet = new Set()
      if (chunkUploadChunkID.value && !chunkUploadCancelled.value) {
        try {
          const progressRes = await getChunkProgress(chunkUploadChunkID.value, totalChunks)
          const uploadedIndices = progressRes.data?.uploaded_chunks || []
          uploadedChunksSet = new Set(uploadedIndices)
          if (uploadedChunksSet.size > 0) {
            ElMessage.info(t('attachment.resume_upload', { count: uploadedChunksSet.size, total: totalChunks }))
          }
        } catch {
          // continue fresh upload
        }
      }

      const chunks = []
      for (let i = 0; i < totalChunks; i++) {
        const start = i * CHUNK_SIZE
        const end = Math.min(start + CHUNK_SIZE, totalSize)
        chunks.push({ index: i, chunk: file.slice(start, end), uploaded: uploadedChunksSet.has(i) })
      }
      chunkUploadChunks.value = chunks

      const pendingChunks = chunks.filter((chunk) => !chunk.uploaded)
      const alreadyUploadedCount = chunks.length - pendingChunks.length
      if (alreadyUploadedCount > 0) {
        chunkUploadProgress.value = Math.round((alreadyUploadedCount / totalChunks) * 100)
      }

      const chunkProgressMap = new Map()
      for (let i = 0; i < totalChunks; i++) {
        chunkProgressMap.set(i, uploadedChunksSet.has(i) ? 1 : 0)
      }

      const updateTotalProgress = () => {
        if (chunkUploadCancelled.value) {
          return
        }
        let totalProgress = 0
        for (let i = 0; i < totalChunks; i++) {
          totalProgress += chunkProgressMap.get(i) || 0
        }
        chunkUploadProgress.value = Math.min(Math.round((totalProgress / totalChunks) * 100), 99)
      }

      const uploadChunkWithProgress = async (chunkData) => {
        if (chunkUploadCancelled.value || chunkData.uploaded) {
          return
        }
        await uploadChunk(chunkUploadChunkID.value, chunkData.index, chunkData.chunk, (progress) => {
          if (!chunkUploadCancelled.value) {
            chunkProgressMap.set(chunkData.index, progress / 100)
            updateTotalProgress()
          }
        })
        if (!chunkUploadCancelled.value) {
          chunkProgressMap.set(chunkData.index, 1)
          updateTotalProgress()
        }
      }

      for (let i = 0; i < pendingChunks.length; i++) {
        await uploadChunkWithProgress(pendingChunks[i])
      }

      if (chunkUploadCancelled.value) {
        return
      }

      await mergeChunks(
        chunkUploadChunkID.value,
        file.name,
        file.type || 'application/octet-stream',
        totalChunks
      )

      if (chunkUploadCancelled.value) {
        return
      }

      chunkUploadStatus.value = 'success'
      chunkUploadProgress.value = 100
      ElMessage.success(t('attachment.upload_success'))

      try {
        if (chunkUploadChunkID.value) {
          Storage.removeItem(`chunk_${chunkUploadChunkID.value}`)
        }
      } catch {
        // ignore
      }

      if (onUploaded) {
        await onUploaded()
      }
    } catch (error) {
      if (chunkUploadCancelled.value) {
        return
      }
      chunkUploadStatus.value = 'exception'
      if (!error.__handled) {
        ElMessage.error(error.response?.data?.message || error.message || t('attachment.upload_failed'))
      }
    }
  }

  const resetChunkUpload = () => {
    chunkUploadCancelled.value = true
    chunkUploadVisible.value = false
    chunkUploadFile.value = null
    chunkUploadChunkID.value = ''
    chunkUploadChunks.value = []
  }

  const handleCancelChunkUpload = () => {
    resetChunkUpload()
  }

  const handleChunkUploadClose = async () => {
    resetChunkUpload()
    if (onUploaded) {
      await onUploaded()
    }
  }

  const handleRetryChunkUpload = () => {
    if (chunkUploadFile.value && chunkUploadChunkID.value) {
      handleChunkUpload(chunkUploadFile.value, false, true)
    } else if (chunkUploadFile.value) {
      handleChunkUpload(chunkUploadFile.value)
    }
  }

  return {
    chunkUploadVisible,
    chunkUploadFile,
    chunkUploadProgress,
    chunkUploadStatus,
    beforeUpload,
    handleLargeFileUpload,
    handleCancelChunkUpload,
    handleChunkUploadClose,
    handleRetryChunkUpload
  }
}

export function useAttachmentUploadConfig(locale) {
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

  const uploadData = computed(() => {
    return {}
  })

  return {
    uploadAction,
    uploadHeaders,
    uploadData
  }
}
