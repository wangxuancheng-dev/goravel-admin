import { onBeforeUnmount, ref } from 'vue'
import axios from 'axios'
import { buildAdminAuthHeaders } from '@/utils/authHeaders'
import { isPrivateAttachmentPreviewPath, isPublicAttachmentPath } from '@/utils/attachmentUrl'

function isPreviewableMedia(fileType) {
  return fileType === 'image' || fileType === 'video'
}

/**
 * Lazy-load attachment image/video thumbnails as blob URLs when preview requires auth.
 */
export function useAttachmentImagePreview() {
  const imageUrlMap = ref(new Map())
  const imageLoadingMap = ref(new Map())
  const blobUrls = []

  const loadImageAsBlob = async (row) => {
    if (!row || !isPreviewableMedia(row.file_type)) {
      return
    }

    const attachmentId = row.id
    if (!attachmentId) {
      return
    }

    const currentState = imageLoadingMap.value.get(attachmentId)
    if (currentState === 'loaded' || currentState === 'loading') {
      return
    }

    const fileUrl = row.file_url
    if (!fileUrl) {
      imageLoadingMap.value.set(attachmentId, 'error')
      return
    }

    const requiresAuth =
      Number(row.is_public) === 0 ||
      isPrivateAttachmentPreviewPath(fileUrl)

    imageLoadingMap.value.set(attachmentId, 'loading')

    if (fileUrl.startsWith('http') && !requiresAuth) {
      imageUrlMap.value.set(attachmentId, fileUrl)
      imageLoadingMap.value.set(attachmentId, 'loaded')
      return
    }

    if (isPublicAttachmentPath(fileUrl) && !import.meta.env.VITE_API_BASE_URL) {
      imageUrlMap.value.set(attachmentId, fileUrl)
      imageLoadingMap.value.set(attachmentId, 'loaded')
      return
    }

    const apiBaseURL = import.meta.env.VITE_API_BASE_URL
    const apiPrefix = import.meta.env.VITE_API_PREFIX || '/api/admin'
    let fullUrl = fileUrl
    if (apiBaseURL) {
      const base = apiBaseURL.replace(/\/+$/, '')
      fullUrl = `${base}${fileUrl}`
    } else {
      fullUrl = `${apiPrefix}${fileUrl.startsWith('/') ? '' : '/'}${fileUrl}`
    }

    try {
      const headers = buildAdminAuthHeaders()
      const response = await axios.get(fullUrl, {
        responseType: 'blob',
        headers
      })
      const blobUrl = URL.createObjectURL(new Blob([response.data]))
      blobUrls.push(blobUrl)
      imageUrlMap.value.set(attachmentId, blobUrl)
      imageLoadingMap.value.set(attachmentId, 'loaded')
    } catch {
      imageLoadingMap.value.set(attachmentId, 'error')
      imageUrlMap.value.set(attachmentId, '')
    }
  }

  const getImageUrl = (row) => {
    if (!row?.id) {
      return ''
    }
    return imageUrlMap.value.get(row.id) || ''
  }

  const getImageLoadingState = (row) => {
    if (!row?.id) {
      return ''
    }

    const state = imageLoadingMap.value.get(row.id)
    const url = imageUrlMap.value.get(row.id)

    if (url && !state) {
      return ''
    }
    if (url && state === 'loading') {
      return 'loading'
    }
    if (state === 'error') {
      return 'error'
    }
    if (state === 'loaded') {
      return ''
    }
    return 'loading'
  }

  const handleImageLoad = (row) => {
    if (row?.id) {
      imageLoadingMap.value.set(row.id, 'loaded')
    }
  }

  const handleImageError = (row) => {
    if (row?.id) {
      imageLoadingMap.value.set(row.id, 'error')
    }
  }

  onBeforeUnmount(() => {
    blobUrls.forEach((url) => URL.revokeObjectURL(url))
    blobUrls.length = 0
  })

  return {
    loadImageAsBlob,
    getImageUrl,
    getImageLoadingState,
    handleImageLoad,
    handleImageError
  }
}
