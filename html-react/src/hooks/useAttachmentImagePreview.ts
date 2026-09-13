import { useCallback, useEffect, useRef, useState } from 'react'
import axios from 'axios'
import Storage from '@/utils/storage'
import { getApiBaseURL } from '@/utils/env'
import { isPrivateAttachmentPreviewPath, isPublicAttachmentPath } from '@/utils/attachmentUrl'

type LoadingState = 'loading' | 'loaded' | 'error' | ''

interface MediaRow {
  id?: string | number
  file_type?: string
  file_url?: string
  is_public?: number
}

function isPreviewableMedia(fileType?: string): boolean {
  return fileType === 'image' || fileType === 'video'
}

export function useAttachmentImagePreview() {
  const [urlMap, setUrlMap] = useState<Map<string | number, string>>(new Map())
  const [loadingMap, setLoadingMap] = useState<Map<string | number, LoadingState>>(new Map())
  const loadingRef = useRef<Map<string | number, LoadingState>>(new Map())
  const blobUrlsRef = useRef<string[]>([])

  const setLoading = useCallback((id: string | number, state: LoadingState) => {
    loadingRef.current.set(id, state)
    setLoadingMap((prev) => {
      const next = new Map(prev)
      next.set(id, state)
      return next
    })
  }, [])

  const setUrl = useCallback((id: string | number, url: string) => {
    setUrlMap((prev) => {
      const next = new Map(prev)
      next.set(id, url)
      return next
    })
  }, [])

  const loadImageAsBlob = useCallback(
    async (row: MediaRow) => {
      if (!row || !isPreviewableMedia(row.file_type) || !row.id) return

      const attachmentId = row.id
      const currentState = loadingRef.current.get(attachmentId)
      if (currentState === 'loaded' || currentState === 'loading') return

      const fileUrl = row.file_url
      if (!fileUrl) {
        setLoading(attachmentId, 'error')
        return
      }

      const requiresAuth = Number(row.is_public) === 0 || isPrivateAttachmentPreviewPath(fileUrl)
      setLoading(attachmentId, 'loading')

      if (fileUrl.startsWith('http') && !requiresAuth) {
        setUrl(attachmentId, fileUrl)
        setLoading(attachmentId, 'loaded')
        return
      }

      if (isPublicAttachmentPath(fileUrl) && !import.meta.env.VITE_API_BASE_URL) {
        setUrl(attachmentId, fileUrl)
        setLoading(attachmentId, 'loaded')
        return
      }

      let fullUrl = fileUrl
      if (fileUrl.startsWith('/') && !fileUrl.startsWith('http')) {
        const base = getApiBaseURL().replace(/\/+$/, '')
        const clean = fileUrl.replace(/^\/api\/admin/, '')
        fullUrl = `${base}${clean.startsWith('/') ? clean : `/${clean}`}`
      } else {
        const apiBaseURL = import.meta.env.VITE_API_BASE_URL as string | undefined
        const apiPrefix = (import.meta.env.VITE_API_PREFIX as string | undefined) || '/api/admin'
        if (apiBaseURL) {
          fullUrl = `${apiBaseURL.replace(/\/+$/, '')}${fileUrl}`
        } else {
          fullUrl = `${apiPrefix}${fileUrl.startsWith('/') ? '' : '/'}${fileUrl}`
        }
      }

      try {
        const token = String(Storage.getItem('token', '') ?? '').trim()
        const headers: Record<string, string> = {}
        if (requiresAuth || token) {
          headers.Authorization = `Bearer ${token}`
        }
        const response = await axios.get(fullUrl, {
          responseType: 'blob',
          headers,
        })
        const blobUrl = URL.createObjectURL(new Blob([response.data]))
        blobUrlsRef.current.push(blobUrl)
        setUrl(attachmentId, blobUrl)
        setLoading(attachmentId, 'loaded')
      } catch {
        setLoading(attachmentId, 'error')
        setUrl(attachmentId, '')
      }
    },
    [setLoading, setUrl],
  )

  const getImageUrl = useCallback(
    (row: MediaRow) => {
      if (!row?.id) return ''
      return urlMap.get(row.id) || ''
    },
    [urlMap],
  )

  const getImageLoadingState = useCallback(
    (row: MediaRow): LoadingState => {
      if (!row?.id) return ''
      const state = loadingMap.get(row.id)
      const url = urlMap.get(row.id)
      if (url && state !== 'error') return ''
      if (state === 'error') return 'error'
      if (state === 'loaded') return ''
      return 'loading'
    },
    [loadingMap, urlMap],
  )

  useEffect(() => {
    return () => {
      blobUrlsRef.current.forEach((url) => URL.revokeObjectURL(url))
      blobUrlsRef.current = []
    }
  }, [])

  return {
    loadImageAsBlob,
    getImageUrl,
    getImageLoadingState,
  }
}
