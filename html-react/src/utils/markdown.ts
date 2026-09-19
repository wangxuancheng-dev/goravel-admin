import DOMPurify from 'dompurify'
import { marked } from 'marked'
import { getApiBaseURL, resolvePublicAssetUrl } from '@/utils/env'
import { buildImageFetchUrl } from '@/utils/publicImage'

marked.setOptions({
  breaks: true,
  gfm: true,
})

const sanitizeOptions = {
  USE_PROFILES: { html: true },
  FORBID_TAGS: ['iframe', 'object', 'embed', 'script', 'style', 'svg', 'form', 'input', 'meta', 'link', 'base'],
  FORBID_ATTR: ['style', 'onerror', 'onload', 'onclick', 'onmouseover'],
  ADD_ATTR: ['target'],
}

let domPurifyHooksInstalled = false

function ensureDomPurifyHooks() {
  if (domPurifyHooksInstalled || typeof window === 'undefined') return
  DOMPurify.addHook('afterSanitizeAttributes', (node) => {
    if (!(node instanceof Element)) return
    if (node.tagName === 'A' && node.getAttribute('target') === '_blank') {
      node.setAttribute('rel', 'noopener noreferrer')
    }
  })
  domPurifyHooksInstalled = true
}

const fallbackForbiddenTags = new Set(sanitizeOptions.FORBID_TAGS)
const dangerousUrlPattern = /^(?:javascript|data|vbscript):/i

function fallbackSanitizeHtml(html: string) {
  if (typeof document === 'undefined') return ''
  const template = document.createElement('template')
  template.innerHTML = html
  template.content.querySelectorAll('*').forEach((element) => {
    if (fallbackForbiddenTags.has(element.tagName.toLowerCase())) {
      element.remove()
      return
    }
    Array.from(element.attributes).forEach((attr) => {
      const name = attr.name.toLowerCase()
      const value = attr.value.trim()
      if (name.startsWith('on') || sanitizeOptions.FORBID_ATTR.includes(name)) {
        element.removeAttribute(attr.name)
        return
      }
      if (['href', 'src', 'xlink:href', 'action'].includes(name) && dangerousUrlPattern.test(value)) {
        element.removeAttribute(attr.name)
      }
    })
  })
  return template.innerHTML
}

export function sanitizeHtml(html: string) {
  if (!html) return ''
  ensureDomPurifyHooks()
  const sanitized = DOMPurify.sanitize(html, sanitizeOptions)
  if (sanitized === '' && html.trim() !== '') {
    return fallbackSanitizeHtml(html)
  }
  return sanitized
}

export function detectContentType(content: string): 'html' | 'markdown' {
  if (!content) return 'markdown'

  const htmlTagPattern =
    /<(p|div|span|img|br|hr|h[1-6]|ul|ol|li|table|tr|td|th|thead|tbody|blockquote|pre|code|strong|em|a|iframe)[\s>]/i
  const hasHtmlTags = htmlTagPattern.test(content)
  const htmlStructurePattern = /<[a-z]+[^>]*>.*<\/[a-z]+>/i
  const hasHtmlStructure = htmlStructurePattern.test(content)
  const htmlAttributePattern = /<[^>]+(class|style|src|href|id|data-)=["'][^"']*["'][^>]*>/i
  const hasHtmlAttributes = htmlAttributePattern.test(content)

  if (hasHtmlTags && (hasHtmlStructure || hasHtmlAttributes)) {
    return 'html'
  }

  const markdownPatterns = [
    /^#{1,6}\s+/m,
    /!\[.*?\]\(.*?\)/,
    /\[.*?\]\(.*?\)/,
    /^\s*[-*+]\s+/m,
    /^\s*\d+\.\s+/m,
    /```[\s\S]*?```/,
    /`[^`]+`/,
    /^\s*>\s+/m,
    /\*\*.*?\*\*/,
    /\*.*?\*/,
  ]
  const markdownScore = markdownPatterns.filter((pattern) => pattern.test(content)).length
  if (markdownScore >= 2 && !hasHtmlStructure) {
    return 'markdown'
  }
  return hasHtmlTags ? 'html' : 'markdown'
}

function processImageUrls(html: string) {
  if (!html) return ''
  const baseURL = getApiBaseURL()
  const cleanBaseURL = baseURL ? baseURL.replace(/\/+$/, '') : ''

  const toImageSrc = (path: string) => {
    const publicUrl = resolvePublicAssetUrl(path)
    if (
      publicUrl.startsWith('/') &&
      (publicUrl.includes('/api/admin/public/images/') || publicUrl.includes('/api/public/files/'))
    ) {
      return buildImageFetchUrl(publicUrl)
    }
    if (path.startsWith('/') && path.includes('/api/public/files/')) {
      return buildImageFetchUrl(path)
    }
    const apiPath = publicUrl.startsWith('/') ? publicUrl : path
    return cleanBaseURL ? `${cleanBaseURL}${apiPath}` : apiPath
  }

  html = html.replace(
    /src=["']https?:\/\/[^/]+(\/(?:api\/admin|api\/public)\/[^"']+)["']/gi,
    (_match, path: string) => `src="${toImageSrc(path)}"`,
  )
  html = html.replace(
    /!\[([^\]]*)\]\(https?:\/\/[^/]+(\/(?:api\/admin|api\/public)\/[^)]+)\)/gi,
    (_match, alt: string, path: string) => `![${alt}](${toImageSrc(path)})`,
  )
  html = html.replace(
    /src=["']((?!https?:\/\/)(\/api\/[^"']+))["']/g,
    (_match, path: string) => `src="${toImageSrc(path)}"`,
  )
  html = html.replace(
    /!\[([^\]]*)\]\((?!https?:\/\/)(\/api\/[^)]+)\)/g,
    (_match, alt: string, path: string) => `![${alt}](${toImageSrc(path)})`,
  )
  return html
}

export function markdownToHtml(markdown: string) {
  if (!markdown) return ''
  const html = marked.parse(markdown) as string
  return sanitizeHtml(processImageUrls(html))
}

export function renderContent(content: string, contentType: 'auto' | 'html' | 'markdown' = 'auto') {
  if (!content) return ''
  const type = contentType === 'auto' ? detectContentType(content) : contentType
  if (type === 'html') {
    return sanitizeHtml(processImageUrls(content))
  }
  return markdownToHtml(content)
}

export function extractTextFromMarkdown(content: string) {
  if (!content) return ''
  if (content.includes('<')) {
    const temp = document.createElement('div')
    temp.innerHTML = sanitizeHtml(content)
    return (temp.textContent || temp.innerText || '').trim()
  }
  return content
    .replace(/!\[([^\]]*)\]\([^)]+\)/g, '')
    .replace(/\[([^\]]+)\]\([^)]+\)/g, '$1')
    .replace(/#{1,6}\s+/g, '')
    .replace(/\*\*([^*]+)\*\*/g, '$1')
    .replace(/\*([^*]+)\*/g, '$1')
    .replace(/`([^`]+)`/g, '$1')
    .replace(/```[\s\S]*?```/g, '')
    .replace(/^\s*[-*+]\s+/gm, '')
    .replace(/^\s*\d+\.\s+/gm, '')
    .replace(/>\s+/g, '')
    .replace(/\n{2,}/g, '\n')
    .trim()
}
