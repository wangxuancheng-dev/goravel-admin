import { useCallback, useRef, useState } from 'react'
import { App } from 'antd'
import { useTranslation } from 'react-i18next'
import {
  getChunkProgress,
  initChunkUpload,
  mergeChunks,
  uploadAttachment,
  uploadChunk,
} from '@/api/attachment'
import Storage from '@/utils/storage'
import { useUnhandledError } from '@/hooks/useUnhandledError'
import {
  ATTACHMENT_CHUNK_SIZE,
  ATTACHMENT_LARGE_FILE_THRESHOLD,
} from '@/pages/attachment/attachment.config'
import type { ApiError } from '@/types'

type ChunkStatus = '' | 'success' | 'exception'

export function useAttachmentChunkUpload(
  onUploaded?: () => Promise<void> | void,
  chunkUploadSupported = true,
) {
  const { t } = useTranslation()
  const { message } = App.useApp()
  const showError = useUnhandledError()

  const [visible, setVisible] = useState(false)
  const [file, setFile] = useState<File | null>(null)
  const [progress, setProgress] = useState(0)
  const [status, setStatus] = useState<ChunkStatus>('')
  const chunkIdRef = useRef('')
  const cancelledRef = useRef(false)

  const reset = useCallback(() => {
    cancelledRef.current = true
    setVisible(false)
    setFile(null)
    chunkIdRef.current = ''
    setProgress(0)
    setStatus('')
  }, [])

  const runChunkUpload = useCallback(
    async (uploadFile: File, useExistingChunkID = false) => {
      setFile(uploadFile)
      setVisible(true)
      setProgress(0)
      setStatus('')
      cancelledRef.current = false

      try {
        const totalSize = uploadFile.size
        if (!totalSize || totalSize <= 0) {
          message.error(t('attachment.invalid_file_size'))
          setVisible(false)
          setFile(null)
          return
        }

        const totalChunks = Math.ceil(totalSize / ATTACHMENT_CHUNK_SIZE)
        if (!totalChunks || !Number.isFinite(totalChunks)) {
          message.error(t('attachment.invalid_chunk_calculation'))
          setVisible(false)
          setFile(null)
          return
        }

        if (!useExistingChunkID || !chunkIdRef.current) {
          try {
            const initRes = await initChunkUpload(
              uploadFile.name,
              totalSize,
              ATTACHMENT_CHUNK_SIZE,
              totalChunks,
            )
            chunkIdRef.current = initRes.data?.chunk_id || ''
            Storage.setItem(`chunk_${chunkIdRef.current}`, {
              filename: uploadFile.name,
              total_size: totalSize,
              chunk_size: ATTACHMENT_CHUNK_SIZE,
              total_chunks: totalChunks,
              created_at: Date.now(),
            })
          } catch (error) {
            const err = error as ApiError
            showError(error, t('common.operation_failed'))
            if (err.errorCode === 'chunk_upload_only_local_storage') {
              setVisible(false)
              setFile(null)
            }
            throw error
          }
        }

        let uploadedChunksSet = new Set<number>()
        if (chunkIdRef.current && !cancelledRef.current) {
          try {
            const progressRes = await getChunkProgress(chunkIdRef.current, totalChunks)
            const uploadedIndices = progressRes.data?.uploaded_chunks || []
            uploadedChunksSet = new Set(uploadedIndices)
            if (uploadedChunksSet.size > 0) {
              message.info(
                t('attachment.resume_upload', {
                  count: uploadedChunksSet.size,
                  total: totalChunks,
                }),
              )
            }
          } catch {
            // continue fresh
          }
        }

        const chunks: Array<{ index: number; chunk: Blob; uploaded: boolean }> = []
        for (let i = 0; i < totalChunks; i++) {
          const start = i * ATTACHMENT_CHUNK_SIZE
          const end = Math.min(start + ATTACHMENT_CHUNK_SIZE, totalSize)
          chunks.push({
            index: i,
            chunk: uploadFile.slice(start, end),
            uploaded: uploadedChunksSet.has(i),
          })
        }

        const pendingChunks = chunks.filter((c) => !c.uploaded)
        const alreadyUploadedCount = chunks.length - pendingChunks.length
        if (alreadyUploadedCount > 0) {
          setProgress(Math.round((alreadyUploadedCount / totalChunks) * 100))
        }

        const chunkProgressMap = new Map<number, number>()
        for (let i = 0; i < totalChunks; i++) {
          chunkProgressMap.set(i, uploadedChunksSet.has(i) ? 1 : 0)
        }

        const updateTotalProgress = () => {
          if (cancelledRef.current) return
          let total = 0
          for (let i = 0; i < totalChunks; i++) {
            total += chunkProgressMap.get(i) || 0
          }
          setProgress(Math.min(Math.round((total / totalChunks) * 100), 99))
        }

        for (const chunkData of pendingChunks) {
          if (cancelledRef.current || chunkData.uploaded) continue
          await uploadChunk(chunkIdRef.current, chunkData.index, chunkData.chunk, (p) => {
            if (!cancelledRef.current) {
              chunkProgressMap.set(chunkData.index, p / 100)
              updateTotalProgress()
            }
          })
          if (!cancelledRef.current) {
            chunkProgressMap.set(chunkData.index, 1)
            updateTotalProgress()
          }
        }

        if (cancelledRef.current) return

        await mergeChunks(
          chunkIdRef.current,
          uploadFile.name,
          uploadFile.type || 'application/octet-stream',
          totalChunks,
        )

        if (cancelledRef.current) return

        setStatus('success')
        setProgress(100)
        message.success(t('attachment.upload_success'))

        try {
          if (chunkIdRef.current) {
            Storage.removeItem(`chunk_${chunkIdRef.current}`)
          }
        } catch {
          // ignore
        }

        await onUploaded?.()
      } catch (error) {
        if (cancelledRef.current) return
        setStatus('exception')
        showError(error, t('attachment.upload_failed'))
      }
    },
    [message, onUploaded, showError, t],
  )

  /** Returns false when upload is handled (chunk) so Ant Upload should not proceed. */
  const beforeUpload = useCallback(
    (uploadFile: File) => {
      const maxSize = 100 * 1024 * 1024
      if (uploadFile.size > maxSize) {
        message.error(t('attachment.file_too_large'))
        return false
      }
      if (uploadFile.size > ATTACHMENT_LARGE_FILE_THRESHOLD) {
        if (!chunkUploadSupported) {
          message.warning(t('attachment.chunk_upload_only_local_storage'))
          return false
        }
        void runChunkUpload(uploadFile, false)
        return false
      }
      return true
    },
    [chunkUploadSupported, message, runChunkUpload, t],
  )

  const handleNormalUpload = useCallback(
    async (uploadFile: File) => {
      if (!beforeUpload(uploadFile)) return
      try {
        await uploadAttachment(uploadFile)
        message.success(t('attachment.upload_success'))
        await onUploaded?.()
      } catch (error) {
        showError(error, t('attachment.upload_failed'))
      }
    },
    [beforeUpload, message, onUploaded, showError, t],
  )

  const handleLargeFileUpload = useCallback(() => {
    if (!chunkUploadSupported) {
      message.warning(t('attachment.chunk_upload_only_local_storage'))
      return
    }
    const input = document.createElement('input')
    input.type = 'file'
    input.onchange = (e) => {
      const selected = (e.target as HTMLInputElement).files?.[0]
      if (selected) void runChunkUpload(selected, true)
    }
    input.click()
  }, [chunkUploadSupported, message, runChunkUpload, t])

  const handleCancel = useCallback(() => {
    reset()
  }, [reset])

  const handleClose = useCallback(async () => {
    reset()
    await onUploaded?.()
  }, [onUploaded, reset])

  const handleRetry = useCallback(() => {
    if (file && chunkIdRef.current) {
      void runChunkUpload(file, true)
    } else if (file) {
      void runChunkUpload(file)
    }
  }, [file, runChunkUpload])

  return {
    visible,
    file,
    progress,
    status,
    beforeUpload,
    handleNormalUpload,
    handleLargeFileUpload,
    startChunkUpload: (uploadFile: File) => runChunkUpload(uploadFile, false),
    handleCancel,
    handleClose,
    handleRetry,
  }
}
