import { createCRUDApi, extendApi } from '@/utils/apiFactory'
import { normalizeListResponse } from '@/utils/normalize'
import { getApiBaseURL } from '@/utils/env'
import request from '@/utils/request'
import Storage from '@/utils/storage'
import type { ApiResponse } from '@/types'

const baseAttachmentApi = createCRUDApi('attachments')

const attachmentApi = extendApi(baseAttachmentApi, {
  upload: (file: File | Blob, onProgress?: (percent: number) => void) => {
    const formData = new FormData()
    formData.append('file', file)
    return request({
      url: '/attachments/upload',
      method: 'post',
      data: formData,
      headers: { 'Content-Type': 'multipart/form-data' },
      onUploadProgress: (event) => {
        if (onProgress && event.total) {
          onProgress(Math.round((event.loaded * 100) / event.total))
        }
      },
    })
  },
  batchDelete: (ids: Array<string | number>) =>
    request({
      url: '/attachments/batch-delete',
      method: 'post',
      data: { ids },
    }),
  updateDisplayName: (id: string | number, displayName: string) =>
    request({
      url: `/attachments/${id}/display-name`,
      method: 'put',
      data: { display_name: displayName },
    }),
  updateCategory: (id: string | number, categoryId: string | number) =>
    request({
      url: `/attachments/${id}/category`,
      method: 'put',
      data: { category_id: categoryId },
    }),
  updateVisibility: (id: string | number, isPublic: boolean) =>
    request({
      url: `/attachments/${id}/visibility`,
      method: 'put',
      data: { is_public: isPublic ? 1 : 0 },
    }),
})

export async function getAttachmentList(params?: Record<string, unknown>) {
  const res = await attachmentApi.list(params)
  return normalizeListResponse(res)
}

export const deleteAttachment = attachmentApi.delete
export const batchDeleteAttachments = attachmentApi.batchDelete
export const uploadAttachment = attachmentApi.upload
export const updateDisplayName = attachmentApi.updateDisplayName
export const updateCategory = attachmentApi.updateCategory
export const updateVisibility = attachmentApi.updateVisibility

type ChunkAction = 'init' | 'upload' | 'merge' | 'progress'

function chunkUpload(
  action: ChunkAction,
  data: Record<string, unknown> = {},
  onProgress?: (percent: number) => void,
) {
  const isGet = action === 'progress'
  const config: Record<string, unknown> = {
    url: '/attachments/chunk',
    method: isGet ? 'get' : 'post',
    timeout: 60000,
    ...(isGet ? { params: { action, ...data } } : { data: { action, ...data } }),
  }

  if (action === 'upload') {
    const formData = new FormData()
    formData.append('action', 'upload')
    formData.append('chunk_id', String(data.chunk_id ?? ''))
    formData.append('chunk_index', String(data.chunk_index ?? ''))
    formData.append('chunk', data.chunk as Blob)
    config.data = formData
    config.headers = { 'Content-Type': 'multipart/form-data' }
    if (onProgress) {
      config.onUploadProgress = (progressEvent: { loaded: number; total?: number }) => {
        if (progressEvent.total) {
          onProgress(Math.round((progressEvent.loaded * 100) / progressEvent.total))
        }
      }
    }
  }

  return request(config as never)
}

export function initChunkUpload(
  filename: string,
  totalSize: number,
  chunkSize: number,
  totalChunks: number,
) {
  return chunkUpload('init', {
    filename,
    total_size: totalSize,
    chunk_size: chunkSize,
    total_chunks: totalChunks,
  }) as Promise<ApiResponse<{ chunk_id: string }>>
}

export function uploadChunk(
  chunkID: string,
  chunkIndex: number,
  chunk: Blob,
  onProgress?: (percent: number) => void,
) {
  return chunkUpload(
    'upload',
    {
      chunk_id: chunkID,
      chunk_index: chunkIndex,
      chunk,
    },
    onProgress,
  )
}

export function mergeChunks(
  chunkID: string,
  filename: string,
  mimeType: string,
  totalChunks?: number,
) {
  let chunks = totalChunks
  if (!chunks) {
    try {
      const chunkInfo = Storage.getItem<{ total_chunks?: number }>(`chunk_${chunkID}`, null)
      if (chunkInfo && typeof chunkInfo === 'object') {
        chunks = chunkInfo.total_chunks
      }
    } catch {
      // ignore
    }
  }
  if (!chunks || chunks <= 0) {
    return Promise.reject(new Error('Total chunks is required'))
  }
  return chunkUpload('merge', {
    chunk_id: chunkID,
    filename,
    mime_type: mimeType,
    total_chunks: chunks,
  })
}

export function getChunkProgress(chunkID: string, totalChunks?: number) {
  if (!chunkID) {
    return Promise.reject(new Error('Chunk ID is empty'))
  }
  let chunks = totalChunks
  if (!chunks) {
    try {
      const chunkInfo = Storage.getItem<{ total_chunks?: number }>(`chunk_${chunkID}`, null)
      if (chunkInfo && typeof chunkInfo === 'object') {
        chunks = chunkInfo.total_chunks
      }
    } catch {
      // ignore
    }
  }
  if (!chunks || chunks <= 0) {
    return Promise.reject(new Error('Total chunks is required'))
  }
  return chunkUpload('progress', {
    chunk_id: chunkID,
    total_chunks: chunks,
  }) as Promise<ApiResponse<{ uploaded_chunks?: number[]; progress?: number }>>
}

/** Relative SSE path for chunk upload progress (nice-to-have). */
export function createUploadProgressSSE(
  chunkID: string,
  totalChunks: number,
  options: { interval?: number } = {},
) {
  const { interval = 500 } = options
  return `/attachments/upload/progress?chunk_id=${chunkID}&total_chunks=${totalChunks}&interval=${interval}`
}

/** Absolute download URL for authenticated blob fetch. */
export function getAttachmentDownloadUrl(id: string | number) {
  const base = getApiBaseURL().replace(/\/+$/, '')
  return `${base}/attachments/${id}/download`
}

/** Absolute inline preview URL for authenticated blob fetch. */
export function getAttachmentPreviewUrl(id: string | number) {
  const base = getApiBaseURL().replace(/\/+$/, '')
  return `${base}/attachments/${id}/preview`
}
