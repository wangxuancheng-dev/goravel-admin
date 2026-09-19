import axios from 'axios'
import Storage from '@/utils/storage'
import { getApiPrefix, resolvePublicAssetUrl } from '@/utils/env'
import {
  ATTACHMENT_PUBLIC_ALIAS_PATH_RE,
  isPrivateAttachmentPreviewPath,
  isPublicAttachmentPath,
} from '@/utils/attachmentUrl'
import { applyTenantHeader, withTenantQuery } from '@/utils/tenant'

export const PUBLIC_IMAGE_PATH_RE = /\/api\/admin\/public\/images\/(\d+)/

/** Build a fetchable URL for a stored image path (public preview / relative / absolute). */
export function buildImageFetchUrl(raw: unknown): string {
  const value = String(raw || '').trim()
  if (!value) return ''
  if (value.startsWith('data:') || value.startsWith('blob:')) return value
  if (/^https?:\/\//i.test(value)) return withTenantQuery(value)

  const publicPath = resolvePublicAssetUrl(value)
  const path = publicPath.startsWith('/') ? publicPath : value
  const apiBaseURL = import.meta.env.VITE_API_BASE_URL as string | undefined
  let url = ''
  if (apiBaseURL) {
    url = `${String(apiBaseURL).replace(/\/+$/, '')}${path.startsWith('/') ? path : `/${path}`}`
  } else {
    const prefix = getApiPrefix().startsWith('/') ? getApiPrefix() : `/${getApiPrefix()}`
    if (path.startsWith(prefix) || path.startsWith('/')) url = path
    else url = `${prefix}/${path}`
  }
  return withTenantQuery(url)
}

/**
 * Resolve a displayable image URL for <img>.
 * Public API paths are fetched as blob so they work in dev with VITE_API_BASE_URL.
 */
export async function resolveImageDisplayUrl(raw: unknown): Promise<{ url: string; revoke?: () => void }> {
  const value = String(raw || '').trim()
  if (!value) return { url: '' }

  if (value.startsWith('data:') || value.startsWith('blob:')) {
    return { url: value }
  }

  const resolvedPublic = resolvePublicAssetUrl(value) || ''
  const needsAuth =
    isPrivateAttachmentPreviewPath(value) || isPrivateAttachmentPreviewPath(resolvedPublic)
  const isPublicImage =
    isPublicAttachmentPath(value) ||
    isPublicAttachmentPath(resolvedPublic) ||
    ATTACHMENT_PUBLIC_ALIAS_PATH_RE.test(value) ||
    ATTACHMENT_PUBLIC_ALIAS_PATH_RE.test(resolvedPublic)

  if (/^https?:\/\//i.test(value) && !isPublicImage && !needsAuth) {
    return { url: value }
  }

  const fetchUrl = buildImageFetchUrl(value)
  if (!fetchUrl) return { url: '' }

  try {
    const token = Storage.getItem<string>('token', '') || ''
    const headers: Record<string, string> = {}
    if (needsAuth || token) {
      headers.Authorization = `Bearer ${String(token).trim()}`
    }
    applyTenantHeader(headers)
    const response = await axios.get(fetchUrl, {
      responseType: 'blob',
      headers,
    })
    const blobUrl = URL.createObjectURL(new Blob([response.data]))
    return {
      url: blobUrl,
      revoke: () => URL.revokeObjectURL(blobUrl),
    }
  } catch {
    return { url: withTenantQuery(resolvedPublic || fetchUrl) }
  }
}

/** Rewrite attachment <img> nodes to blob URLs (sends tenant header + auth). */
export async function hydrateContentImages(root: ParentNode | null): Promise<() => void> {
  if (!root || typeof (root as Element).querySelectorAll !== 'function') {
    return () => {}
  }
  const revokers: Array<() => void> = []
  const imgs = Array.from((root as Element).querySelectorAll('img'))
  await Promise.all(
    imgs.map(async (img) => {
      const raw = String(img.getAttribute('src') || '').trim()
      if (!raw || raw.startsWith('data:') || raw.startsWith('blob:')) return
      const resolvedPublic = resolvePublicAssetUrl(raw) || ''
      const needs =
        isPublicAttachmentPath(raw) ||
        isPublicAttachmentPath(resolvedPublic) ||
        isPrivateAttachmentPreviewPath(raw) ||
        isPrivateAttachmentPreviewPath(resolvedPublic) ||
        ATTACHMENT_PUBLIC_ALIAS_PATH_RE.test(raw) ||
        ATTACHMENT_PUBLIC_ALIAS_PATH_RE.test(resolvedPublic)
      if (!needs) return
      const { url, revoke } = await resolveImageDisplayUrl(raw)
      if (!url) return
      img.setAttribute('src', url)
      if (typeof revoke === 'function') revokers.push(revoke)
    }),
  )
  return () => {
    revokers.forEach((fn) => {
      try {
        fn()
      } catch {
        /* ignore */
      }
    })
  }
}

export const WEBSITE_CONFIG_UPDATED_EVENT = 'website-config-updated'

export function notifyWebsiteConfigUpdated() {
  if (typeof window !== 'undefined') {
    window.dispatchEvent(new CustomEvent(WEBSITE_CONFIG_UPDATED_EVENT))
  }
}
