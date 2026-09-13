import request from '../utils/request'
import Storage from '../utils/storage'
import { createCRUDApi, extendApi } from '../utils/apiFactory'
import { normalizeListResponse } from '../utils/normalize'

const baseAttachmentApi = createCRUDApi('attachments')

const attachmentApi = extendApi(baseAttachmentApi, {
  upload: (file, onProgress) => {
    const formData = new FormData()
    formData.append('file', file)
    return request({
      url: '/attachments/upload',
      method: 'post',
      data: formData,
      headers: {
        'Content-Type': 'multipart/form-data'
      },
      onUploadProgress: (progressEvent) => {
        if (onProgress && progressEvent.total) {
          onProgress(Math.round((progressEvent.loaded * 100) / progressEvent.total))
        }
      }
    })
  },
  batchDelete: (ids) => {
    return request({
      url: '/attachments/batch-delete',
      method: 'post',
      data: { ids }
    })
  },
  updateDisplayName: (id, displayName) => {
    return request({
      url: `/attachments/${id}/display-name`,
      method: 'put',
      data: { display_name: displayName }
    })
  },
  updateCategory: (id, categoryId) => {
    return request({
      url: `/attachments/${id}/category`,
      method: 'put',
      data: { category_id: categoryId }
    })
  },
  updateVisibility: (id, isPublic) => {
    return request({
      url: `/attachments/${id}/visibility`,
      method: 'put',
      data: { is_public: isPublic ? 1 : 0 }
    })
  }
})

export async function getAttachmentList(params) {
  const res = await attachmentApi.list(params)
  return normalizeListResponse(res)
}

export const {
  delete: deleteAttachment,
  batchDelete: batchDeleteAttachments,
  upload: uploadFile,
  updateDisplayName,
  updateCategory,
  updateVisibility
} = attachmentApi

export function chunkUpload(action, data = {}, onProgress) {
  const isGet = action === 'progress'
  const config = {
    url: '/attachments/chunk',
    method: isGet ? 'get' : 'post',
    timeout: 60000,
    ...(isGet ? { params: { action, ...data } } : { data: { action, ...data } })
  }

  if (action === 'upload') {
    const formData = new FormData()
    formData.append('action', 'upload')
    formData.append('chunk_id', data.chunk_id)
    formData.append('chunk_index', data.chunk_index)
    formData.append('chunk', data.chunk)
    config.data = formData
    config.headers = { 'Content-Type': 'multipart/form-data' }
    if (onProgress) {
      config.onUploadProgress = (progressEvent) => {
        if (progressEvent.total) {
          onProgress(Math.round((progressEvent.loaded * 100) / progressEvent.total))
        }
      }
    }
  }

  return request(config)
}

export function initChunkUpload(filename, totalSize, chunkSize, totalChunks) {
  return chunkUpload('init', {
    filename,
    total_size: totalSize,
    chunk_size: chunkSize,
    total_chunks: totalChunks
  })
}

export function uploadChunk(chunkID, chunkIndex, chunk, onProgress) {
  return chunkUpload('upload', {
    chunk_id: chunkID,
    chunk_index: chunkIndex,
    chunk
  }, onProgress)
}

export function mergeChunks(chunkID, filename, mimeType, totalChunks) {
  if (!totalChunks) {
    try {
      const chunkInfo = Storage.getItem(`chunk_${chunkID}`, null)
      if (chunkInfo && typeof chunkInfo === 'object') {
        totalChunks = chunkInfo.total_chunks
      }
    } catch {
      // ignore storage read errors
    }
  }
  if (!totalChunks || totalChunks <= 0) {
    return Promise.reject(new Error('Total chunks is required'))
  }
  return chunkUpload('merge', {
    chunk_id: chunkID,
    filename,
    mime_type: mimeType,
    total_chunks: totalChunks
  })
}

export function getChunkProgress(chunkID, totalChunks) {
  if (!chunkID) {
    return Promise.reject(new Error('Chunk ID is empty'))
  }
  if (!totalChunks) {
    try {
      const chunkInfo = Storage.getItem(`chunk_${chunkID}`, null)
      if (chunkInfo && typeof chunkInfo === 'object') {
        totalChunks = chunkInfo.total_chunks
      }
    } catch {
      // ignore storage read errors
    }
  }
  if (!totalChunks || totalChunks <= 0) {
    return Promise.reject(new Error('Total chunks is required'))
  }
  return chunkUpload('progress', { chunk_id: chunkID, total_chunks: totalChunks })
}

export function createUploadProgressSSE(chunkID, totalChunks, options = {}) {
  const { interval = 500 } = options
  return `/attachments/upload/progress?chunk_id=${chunkID}&total_chunks=${totalChunks}&interval=${interval}`
}

/** Absolute authenticated URL for attachment download. */
export function getAttachmentDownloadUrl(id) {
  const apiBaseURL = import.meta.env.VITE_API_BASE_URL
  const apiPrefix = import.meta.env.VITE_API_PREFIX || '/api/admin'
  let url = `${apiPrefix}/attachments/${id}/download`
  if (apiBaseURL) {
    url = `${String(apiBaseURL).replace(/\/+$/, '')}${url}`
  }
  return url
}

/** Absolute authenticated URL for attachment inline preview. */
export function getAttachmentPreviewUrl(id) {
  const apiBaseURL = import.meta.env.VITE_API_BASE_URL
  const apiPrefix = import.meta.env.VITE_API_PREFIX || '/api/admin'
  let url = `${apiPrefix}/attachments/${id}/preview`
  if (apiBaseURL) {
    url = `${String(apiBaseURL).replace(/\/+$/, '')}${url}`
  }
  return url
}
