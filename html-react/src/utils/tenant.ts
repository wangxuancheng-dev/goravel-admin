/**
 * Tenancy helpers for React admin.
 * Enable with VITE_TENANCY_ENABLED=true or VITE_TENANCY_DRIVER=database
 * (must match backend TENANCY_DRIVER=database).
 */
import Storage from './storage'

const STORAGE_KEY = 'tenant_code'

const DEFAULT_RESERVED = ['www', 'api', 'admin', 'platform', 'static', 'assets']

export function isTenancyEnabled(): boolean {
  const enabled = String(import.meta.env.VITE_TENANCY_ENABLED || '').toLowerCase()
  const driver = String(import.meta.env.VITE_TENANCY_DRIVER || '').toLowerCase()
  return enabled === 'true' || enabled === '1' || driver === 'database'
}

export function getTenantHeaderName(): string {
  return import.meta.env.VITE_TENANCY_HEADER || 'X-Tenant-ID'
}

/** Matches backend TENANCY_BASE_DOMAIN (for {code}.base subdomain detection). */
export function getTenancyBaseDomain(): string {
  return String(import.meta.env.VITE_TENANCY_BASE_DOMAIN || '')
    .trim()
    .toLowerCase()
    .replace(/^\.+|\.+$/g, '')
}

function reservedLabels(): Set<string> {
  const raw = String(import.meta.env.VITE_TENANCY_SUBDOMAIN_RESERVED || '').trim()
  const list = raw
    ? raw.split(',').map((s) => s.trim().toLowerCase()).filter(Boolean)
    : DEFAULT_RESERVED
  return new Set(list)
}

export function getTenantCode(): string {
  return String(Storage.getItem<string>(STORAGE_KEY, '') || '').trim()
}

export function setTenantCode(code: string | undefined | null): string {
  const value = String(code || '').trim().toLowerCase()
  if (!value) {
    Storage.removeItem(STORAGE_KEY)
    return ''
  }
  Storage.setItem(STORAGE_KEY, value)
  return value
}

export function clearTenantCode(): void {
  Storage.removeItem(STORAGE_KEY)
}

/**
 * Admin login path; keep tenant_code on logout/401 so re-login does not lose the hint.
 * Pass an explicit code when calling after clearTenantCode().
 * On a tenant subdomain, plain /login is enough (Host/Origin bind on the API).
 */
export function buildAdminLoginPath(code?: string | null): string {
  const raw = code === undefined || code === null ? getTenantCode() : code
  const value = String(raw || '').trim().toLowerCase()
  if (!value) return '/login'
  if (typeof window !== 'undefined' && resolveTenantCodeFromHostname() === value) {
    return '/login'
  }
  return `/login?tenant_code=${encodeURIComponent(value)}`
}

/** Prefer ?tenant_code= / ?tenant= over stored value. */
export function resolveTenantCodeFromLocation(search = window.location.search): string {
  try {
    const params = new URLSearchParams(search)
    return String(params.get('tenant_code') || params.get('tenant') || '').trim().toLowerCase()
  } catch {
    return ''
  }
}

/**
 * Built-in tenant subdomain: acme.{VITE_TENANCY_BASE_DOMAIN}.
 * Vanity hosts are not mapped to a code here (backend uses Origin/Host + tenant_domains).
 */
export function resolveTenantCodeFromHostname(
  hostname: string = typeof window !== 'undefined' ? window.location.hostname : '',
  baseDomain: string = getTenancyBaseDomain(),
): string {
  const host = String(hostname || '').trim().toLowerCase()
  const base = String(baseDomain || '').trim().toLowerCase()
  if (!host || !base) return ''
  const suffix = `.${base}`
  if (!host.endsWith(suffix)) return ''
  const label = host.slice(0, -suffix.length)
  if (!label || label.includes('.')) return ''
  if (reservedLabels().has(label)) return ''
  return label
}

/** True when the browser host already identifies the tenant (subdomain or vanity). */
export function isHostBoundTenantContext(
  hostname: string = typeof window !== 'undefined' ? window.location.hostname : '',
  baseDomain: string = getTenancyBaseDomain(),
): boolean {
  if (resolveTenantCodeFromHostname(hostname, baseDomain)) return true
  return isLikelyVanityTenantHost(hostname, baseDomain)
}

/**
 * Host outside TENANCY_BASE_DOMAIN (or nested under it) that is not a platform reserved name.
 * Requires VITE_TENANCY_BASE_DOMAIN so we do not treat localhost/dev hosts as vanity.
 */
export function isLikelyVanityTenantHost(
  hostname: string = typeof window !== 'undefined' ? window.location.hostname : '',
  baseDomain: string = getTenancyBaseDomain(),
): boolean {
  const host = String(hostname || '').trim().toLowerCase()
  const base = String(baseDomain || '').trim().toLowerCase()
  if (!host || !base) return false
  if (host === 'localhost' || host === '127.0.0.1') return false
  if (/^\d{1,3}(\.\d{1,3}){3}$/.test(host)) return false
  if (host === base) return false
  for (const label of reservedLabels()) {
    if (host === `${label}.${base}`) return false
  }
  if (host.endsWith(`.${base}`)) {
    const label = host.slice(0, -(base.length + 1))
    return label.includes('.')
  }
  return host.includes('.')
}

/** Storage > query > hostname subdomain. */
export function resolveEffectiveTenantCode(): string {
  return (
    getTenantCode() ||
    resolveTenantCodeFromLocation() ||
    resolveTenantCodeFromHostname()
  )
}

export function applyTenantHeader(headers: Record<string, unknown> | undefined): void {
  if (!headers) return
  const code = resolveEffectiveTenantCode()
  if (!code) return
  headers[getTenantHeaderName()] = code
}

const ATTACHMENT_PUBLIC_HINT = /\/api\/admin\/public\/images\/|\/api\/public\/files\//

/**
 * Append tenant_code query for public asset URLs (img/src cannot send custom headers).
 * Apply whenever a local tenant code exists (same as applyTenantHeader), even if VITE_TENANCY_* is unset.
 */
export function withTenantQuery(url: string): string {
  const value = String(url || '').trim()
  if (!value) return value
  const code = resolveEffectiveTenantCode()
  if (!code) return value
  if (!ATTACHMENT_PUBLIC_HINT.test(value)) return value
  try {
    const abs = value.startsWith('http') ? new URL(value) : new URL(value, 'http://local.invalid')
    if (!abs.searchParams.get('tenant_code') && !abs.searchParams.get('tenant_id')) {
      abs.searchParams.set('tenant_code', code)
    }
    if (value.startsWith('http')) return abs.toString()
    return `${abs.pathname}${abs.search}${abs.hash}`
  } catch {
    const sep = value.includes('?') ? '&' : '?'
    if (/[?&]tenant_code=/.test(value) || /[?&]tenant_id=/.test(value)) return value
    return `${value}${sep}tenant_code=${encodeURIComponent(code)}`
  }
}

/**
 * URL for "back to tenant admin login" from the platform console.
 * Prefer VITE_TENANT_DEMO_LOGIN_URL; hosted demo host falls back to the public tenant demo.
 */
export function getTenantAdminLoginUrl(): string {
  const fromEnv = String(import.meta.env.VITE_TENANT_DEMO_LOGIN_URL || '').trim()
  if (fromEnv) return fromEnv
  if (typeof window !== 'undefined' && window.location.hostname === 'admin.xuancheng888.top') {
    return 'https://acme.xuancheng888.top/login'
  }
  return '/login'
}
