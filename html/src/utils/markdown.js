/**
 * Markdown 和 HTML 内容处理工具
 * 
 * 功能：
 * 1. 自动判断内容是 HTML 还是 Markdown
 * 2. 将 Markdown 转换为 HTML
 * 3. 处理图片路径（相对路径转绝对路径）
 * 4. 提取纯文本（用于列表显示）
 * 
 * 使用示例：
 * ```javascript
 * import { renderContent, extractTextFromMarkdown } from '@/utils/markdown'
 * 
 * // 自动判断并渲染
 * const html = renderContent(content, 'auto')
 * 
 * // 强制指定类型
 * const html = renderContent(content, 'html')  // 或 'markdown'
 * 
 * // 提取纯文本
 * const text = extractTextFromMarkdown(content)
 * ```
 */

import createDOMPurify from 'dompurify'
import { marked } from 'marked'
import { getApiBaseURL, resolvePublicAssetUrl } from './env'
import { buildImageFetchUrl } from './publicImage'

// 配置 marked 选项
marked.setOptions({
  breaks: true, // 支持 GitHub 风格的换行
  gfm: true, // 启用 GitHub 风格的 Markdown
  silent: true // 静默模式，不抛出错误
})

const sanitizeOptions = {
  USE_PROFILES: { html: true },
  FORBID_TAGS: ['iframe', 'object', 'embed', 'script', 'style', 'svg', 'form', 'input', 'meta', 'link', 'base'],
  FORBID_ATTR: ['style', 'onerror', 'onload', 'onclick', 'onmouseover']
}

const purifier = typeof window !== 'undefined'
  ? createDOMPurify(window)
  : createDOMPurify

let domPurifyHooksInstalled = false

function ensureDomPurifyHooks() {
  if (domPurifyHooksInstalled || typeof window === 'undefined' || !purifier?.addHook) return
  purifier.addHook('afterSanitizeAttributes', (node) => {
    if (node?.tagName === 'A' && node.getAttribute('target') === '_blank') {
      node.setAttribute('rel', 'noopener noreferrer')
    }
  })
  domPurifyHooksInstalled = true
}

const fallbackForbiddenTags = new Set(sanitizeOptions.FORBID_TAGS)
const dangerousUrlPattern = /^(?:javascript|data|vbscript):/i

function fallbackSanitizeHtml(html) {
  if (typeof document === 'undefined') {
    return ''
  }

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

/**
 * 净化即将用于 v-html 的内容，防止脚本、事件属性和危险标签进入 DOM。
 * @param {string} html - HTML 字符串
 * @returns {string} 净化后的 HTML
 */
export function sanitizeHtml(html) {
  if (!html) return ''
  ensureDomPurifyHooks()
  if (purifier?.isSupported === false) {
    return fallbackSanitizeHtml(html)
  }
  const sanitized = purifier.sanitize(html, sanitizeOptions)
  if (sanitized === '' && String(html).trim() !== '') {
    return fallbackSanitizeHtml(html)
  }
  return sanitized
}

/**
 * 判断内容是 HTML 还是 Markdown
 * @param {string} content - 内容文本
 * @returns {'html'|'markdown'} 内容类型
 */
export function detectContentType(content) {
  if (!content) return 'markdown'
  
  // 检查是否包含常见的 HTML 标签（排除 markdown 可能产生的简单标签）
  const htmlTagPattern = /<(p|div|span|img|br|hr|h[1-6]|ul|ol|li|table|tr|td|th|thead|tbody|blockquote|pre|code|strong|em|a|iframe)[\s>]/i
  const hasHtmlTags = htmlTagPattern.test(content)
  
  // 检查是否包含完整的 HTML 结构（如 <p>...</p>）
  const htmlStructurePattern = /<[a-z]+[^>]*>.*<\/[a-z]+>/i
  const hasHtmlStructure = htmlStructurePattern.test(content)
  
  // 检查是否包含 HTML 属性（如 class, style, src 等）
  const htmlAttributePattern = /<[^>]+(class|style|src|href|id|data-)=["'][^"']*["'][^>]*>/i
  const hasHtmlAttributes = htmlAttributePattern.test(content)
  
  // 如果同时满足多个 HTML 特征，认为是 HTML
  if (hasHtmlTags && (hasHtmlStructure || hasHtmlAttributes)) {
    return 'html'
  }
  
  // 检查是否包含明显的 markdown 语法
  const markdownPatterns = [
    /^#{1,6}\s+/m,           // 标题
    /!\[.*?\]\(.*?\)/,       // 图片
    /\[.*?\]\(.*?\)/,        // 链接
    /^\s*[-*+]\s+/m,         // 无序列表
    /^\s*\d+\.\s+/m,         // 有序列表
    /```[\s\S]*?```/,       // 代码块
    /`[^`]+`/,               // 行内代码
    /^\s*>\s+/m,             // 引用
    /\*\*.*?\*\*/,           // 加粗
    /\*.*?\*/,               // 斜体
  ]
  
  const markdownScore = markdownPatterns.filter(pattern => pattern.test(content)).length
  
  // 如果 markdown 特征明显多于 HTML 特征，认为是 markdown
  if (markdownScore >= 2 && !hasHtmlStructure) {
    return 'markdown'
  }
  
  // 默认：如果有 HTML 标签，认为是 HTML；否则认为是 markdown
  return hasHtmlTags ? 'html' : 'markdown'
}

/**
 * 将 markdown 转换为 HTML，并处理图片路径
 * @param {string} markdown - markdown 文本
 * @returns {string} HTML 字符串
 */
export function markdownToHtml(markdown) {
  if (!markdown) return ''
  
  // 先转换 markdown 为 HTML
  let html = marked.parse(markdown)
  
  // 处理图片路径
  html = processImageUrls(html)
  
  return sanitizeHtml(html)
}

/**
 * 渲染内容（自动判断 HTML 或 Markdown）
 * @param {string} content - 内容文本
 * @param {string} contentType - 可选：强制指定内容类型 'html' | 'markdown' | 'auto'
 * @returns {string} HTML 字符串
 */
export function renderContent(content, contentType = 'auto') {
  if (!content) return ''
  
  let type = contentType
  if (type === 'auto') {
    type = detectContentType(content)
  }
  
  if (type === 'html') {
    // 处理 HTML 中的图片路径后统一净化，再交给 v-html 渲染。
    return sanitizeHtml(processImageUrls(content))
  } else {
    // 转换为 HTML 并处理图片路径
    return markdownToHtml(content)
  }
}

/**
 * 处理 HTML 中的图片 URL，将相对路径转换为绝对路径
 * @param {string} html - HTML 字符串
 * @returns {string} 处理后的 HTML
 */
export function processImageUrls(html) {
  if (!html) return ''
  
  const baseURL = getApiBaseURL()
  const cleanBaseURL = baseURL ? baseURL.replace(/\/+$/, '') : ''

  const toImageSrc = (path) => {
    const publicUrl = resolvePublicAssetUrl(path)
    if (
      publicUrl.startsWith('/') &&
      (publicUrl.includes('/api/admin/public/images/') || publicUrl.includes('/api/public/files/'))
    ) {
      // Absolute API URL when VITE_API_BASE_URL is set; always append tenant_code for <img>.
      return buildImageFetchUrl(publicUrl)
    }
    if (path.startsWith('/') && path.includes('/api/public/files/')) {
      return buildImageFetchUrl(path)
    }
    const apiPath = publicUrl.startsWith('/') ? publicUrl : path
    return cleanBaseURL ? `${cleanBaseURL}${apiPath}` : apiPath
  }

  // 通知/富文本里可能存了带域名的绝对地址，先还原为路径再按规则输出
  html = html.replace(
    /src=["']https?:\/\/[^/]+(\/(?:api\/admin|api\/public)\/[^"']+)["']/gi,
    (match, path) => `src="${toImageSrc(path)}"`
  )
  html = html.replace(
    /!\[([^\]]*)\]\(https?:\/\/[^/]+(\/(?:api\/admin|api\/public)\/[^)]+)\)/gi,
    (match, alt, path) => `![${alt}](${toImageSrc(path)})`
  )
  
  // 相对路径：公开图片同源，其余 API 资源拼 baseURL
  html = html.replace(
    /src=["']((?!https?:\/\/)(\/api\/[^"']+))["']/g,
    (match, path) => `src="${toImageSrc(path)}"`
  )
  
  html = html.replace(
    /!\[([^\]]*)\]\((?!https?:\/\/)(\/api\/[^)]+)\)/g,
    (match, alt, path) => `![${alt}](${toImageSrc(path)})`
  )
  
  return html
}

/**
 * 从 markdown 或 HTML 中提取纯文本（用于列表显示）
 * @param {string} content - markdown 或 HTML 内容
 * @returns {string} 纯文本
 */
export function extractTextFromMarkdown(content) {
  if (!content) return ''
  
  // 如果是 HTML，先提取文本
  if (content.includes('<')) {
    const temp = document.createElement('div')
    temp.innerHTML = sanitizeHtml(content)
    return temp.textContent || temp.innerText || ''
  }
  
  // 如果是 markdown，移除 markdown 语法
  return content
    .replace(/!\[([^\]]*)\]\([^)]+\)/g, '') // 移除图片
    .replace(/\[([^\]]+)\]\([^)]+\)/g, '$1') // 移除链接，保留文本
    .replace(/#{1,6}\s+/g, '') // 移除标题标记
    .replace(/\*\*([^*]+)\*\*/g, '$1') // 移除加粗
    .replace(/\*([^*]+)\*/g, '$1') // 移除斜体
    .replace(/`([^`]+)`/g, '$1') // 移除行内代码
    .replace(/```[\s\S]*?```/g, '') // 移除代码块
    .replace(/^\s*[-*+]\s+/gm, '') // 移除列表标记
    .replace(/^\s*\d+\.\s+/gm, '') // 移除有序列表标记
    .replace(/>\s+/g, '') // 移除引用标记
    .replace(/\n{2,}/g, '\n') // 合并多个换行
    .trim()
}
