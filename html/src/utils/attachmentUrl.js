/** @typedef {{ id?: number, is_public?: number, file_url?: string }} AttachmentUploadData */

export const ATTACHMENT_PUBLIC_PATH_RE = /\/api\/admin\/public\/images\/(\d+)/
export const ATTACHMENT_PUBLIC_ALIAS_PATH_RE = /\/api\/public\/files\/(\d+)/
export const ATTACHMENT_PRIVATE_PREVIEW_PATH_RE = /\/api\/admin\/attachments\/(\d+)\/preview/

/**
 * Stable path to store in config / rich text (public attachments only).
 * @param {AttachmentUploadData | null | undefined} data
 */
export function toStablePublicAttachmentUrl(data) {
  if (!data?.id) return ''
  if (Number(data.is_public) === 0) return ''
  return `/api/admin/public/images/${data.id}`
}

/**
 * Prefer stable relative path from upload response; fall back to file_url.
 * @param {AttachmentUploadData | null | undefined} data
 */
export function resolveUploadStorageUrl(data) {
  const stable = toStablePublicAttachmentUrl(data)
  if (stable) return stable
  return data?.file_url || ''
}

export function isPublicAttachmentPath(value) {
  const path = String(value || '')
  return ATTACHMENT_PUBLIC_PATH_RE.test(path) || ATTACHMENT_PUBLIC_ALIAS_PATH_RE.test(path)
}

export function isPrivateAttachmentPreviewPath(value) {
  return ATTACHMENT_PRIVATE_PREVIEW_PATH_RE.test(String(value || ''))
}

export function extractAttachmentIdFromPath(value) {
  const path = String(value || '')
  for (const re of [ATTACHMENT_PUBLIC_PATH_RE, ATTACHMENT_PUBLIC_ALIAS_PATH_RE, ATTACHMENT_PRIVATE_PREVIEW_PATH_RE]) {
    const match = path.match(re)
    if (match?.[1]) return Number(match[1])
  }
  return 0
}

const BLOCKED_PREVIEW_EXTENSIONS = new Set(['html', 'htm', 'xhtml', 'svg', 'js', 'mjs', 'cjs', 'xml'])

/**
 * @param {{ mime_type?: string, file_type?: string, extension?: string } | null | undefined} row
 */
function normalizePreviewMeta(row) {
  const mime = String(row?.mime_type || '').toLowerCase().trim()
  const fileType = String(row?.file_type || '').toLowerCase().trim()
  const extension = String(row?.extension || '')
    .toLowerCase()
    .trim()
    .replace(/^\./, '')
  return { mime, fileType, extension }
}

function isBlockedPreviewMimeOrExt(mime, extension) {
  if (
    mime.includes('html') ||
    mime.includes('javascript') ||
    mime.includes('svg') ||
    mime.includes('xml')
  ) {
    return true
  }
  return BLOCKED_PREVIEW_EXTENSIONS.has(extension)
}

/**
 * Mirror backend AttachmentInlinePreviewSafe: pdf, safe text, image, video/mp4|webm|quicktime.
 * @param {{ mime_type?: string, file_type?: string, extension?: string } | null | undefined} row
 */
export function canInlinePreviewAttachment(row) {
  const { mime, extension } = normalizePreviewMeta(row)
  if (isBlockedPreviewMimeOrExt(mime, extension)) return false
  if (!mime) return false

  switch (mime) {
    case 'image/jpeg':
    case 'image/png':
    case 'image/gif':
    case 'image/webp':
    case 'video/mp4':
    case 'video/webm':
    case 'video/quicktime':
    case 'application/pdf':
    case 'text/plain':
    case 'text/csv':
    case 'text/markdown':
    case 'text/x-markdown':
      return true
    default:
      break
  }
  return mime.startsWith('text/')
}

/**
 * @param {{ mime_type?: string, file_type?: string, extension?: string } | null | undefined} row
 * @returns {'image' | 'video' | 'pdf' | 'text' | ''}
 */
export function attachmentPreviewKind(row) {
  if (!canInlinePreviewAttachment(row)) return ''
  const { mime, fileType, extension } = normalizePreviewMeta(row)

  if (
    mime.startsWith('image/') ||
    fileType === 'image' ||
    ['jpg', 'jpeg', 'png', 'gif', 'webp'].includes(extension)
  ) {
    return 'image'
  }
  if (mime === 'application/pdf' || extension === 'pdf') {
    return 'pdf'
  }
  if (
    mime === 'video/mp4' ||
    mime === 'video/webm' ||
    mime === 'video/quicktime' ||
    fileType === 'video' ||
    ['mp4', 'webm', 'mov'].includes(extension)
  ) {
    return 'video'
  }
  if (mime.startsWith('text/') || ['txt', 'csv', 'md', 'markdown'].includes(extension)) {
    return 'text'
  }
  return ''
}
