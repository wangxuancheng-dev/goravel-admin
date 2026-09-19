/** Resolve API base URL from Vite env (same contract as Vue frontend). */
export function getApiBaseURL(): string {
  const apiBaseURL = import.meta.env.VITE_API_BASE_URL
  const apiPrefix = import.meta.env.VITE_API_PREFIX || '/api/admin'

  if (apiBaseURL) {
    const base = apiBaseURL.replace(/\/+$/, '')
    const prefix = apiPrefix.startsWith('/') ? apiPrefix : `/${apiPrefix}`
    return `${base}${prefix}`
  }

  return apiPrefix
}

export function getApiPrefix(): string {
  return import.meta.env.VITE_API_PREFIX || '/api/admin'
}

const PUBLIC_ATTACHMENT_PATH_RE = /\/api\/(?:admin\/public\/images|public\/files)\/\d+/

/** Public attachment paths stay relative for same-origin access (logo, notification body). */
export function resolvePublicAssetUrl(raw: unknown): string {
  if (!raw) return ''
  const value = String(raw).trim()
  if (!value) return ''

  let path = value
  if (/^https?:\/\//i.test(value)) {
    try {
      const parsed = new URL(value)
      path = `${parsed.pathname}${parsed.search}${parsed.hash}`
    } catch {
      path = value.replace(/^https?:\/\/[^/]+/i, '')
    }
  }

  const prefix = getApiPrefix().startsWith('/') ? getApiPrefix() : `/${getApiPrefix()}`
  if (!path.startsWith('/')) {
    path = path.startsWith(prefix.replace(/^\//, '')) ? `/${path}` : `${prefix}/${path}`
  }
  if (
    !PUBLIC_ATTACHMENT_PATH_RE.test(path) &&
    !path.includes('/api/admin/public/images/') &&
    !path.includes('/api/public/files/')
  ) {
    return value
  }
  return path
}

export function getWsBaseURL(): string {
  const apiBaseURL = import.meta.env.VITE_API_BASE_URL || `${window.location.protocol}//${window.location.host}`
  return apiBaseURL.replace(/^http/, 'ws').replace(/\/+$/, '')
}
