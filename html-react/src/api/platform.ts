import platformRequest, {
  clearPlatformSession,
  setPlatformAdmin,
  setPlatformToken,
} from '@/utils/platformRequest'
import { normalizeListResponse } from '@/utils/normalize'

export function platformLogin(data: { username: string; password: string }) {
  return platformRequest.post('/login', data)
}

export function platformLogout() {
  return platformRequest.post('/logout')
}

export function platformInfo() {
  return platformRequest.get('/info')
}

export function platformHealth() {
  return platformRequest.get('/health')
}

export function updatePlatformPassword(data: {
  old_password: string
  new_password: string
  confirm_password?: string
}) {
  return platformRequest.put('/password', data)
}

export async function getPlatformTenantList(params?: Record<string, unknown>) {
  return normalizeListResponse(await platformRequest.get('/tenants', { params }))
}

export function getPlatformTenantDetail(id: string | number) {
  return platformRequest.get(`/tenants/${id}`)
}

export function createPlatformTenant(data: Record<string, unknown>) {
  return platformRequest.post('/tenants', data)
}

export function updatePlatformTenant(id: string | number, data: Record<string, unknown>) {
  return platformRequest.put(`/tenants/${id}`, data)
}

export function updatePlatformTenantStatus(id: string | number, status: number) {
  return platformRequest.put(`/tenants/${id}/status`, { status })
}

export function pingPlatformTenant(id: string | number) {
  return platformRequest.post(`/tenants/${id}/ping`)
}

export function migratePlatformTenant(id: string | number, data?: { with_seed?: boolean }) {
  return platformRequest.post(`/tenants/${id}/migrate`, data || {})
}

export function seedPlatformTenant(id: string | number) {
  return platformRequest.post(`/tenants/${id}/seed`)
}

export function backupPlatformTenant(id: string | number, data?: { keep?: number }) {
  return platformRequest.post(`/tenants/${id}/backup`, data || {})
}

export function restorePlatformTenant(id: string | number, data: { backup_name: string }) {
  return platformRequest.post(`/tenants/${id}/restore`, data)
}

export function deletePlatformTenant(
  id: string | number,
  data: { confirm_code: string; drop_database?: boolean },
) {
  return platformRequest.delete(`/tenants/${id}`, { data })
}

export function getPlatformTenantOverview(id: string | number) {
  return platformRequest.get(`/tenants/${id}/overview`)
}

export function getPlatformTenantOpLogs(id: string | number, params?: { limit?: number }) {
  return platformRequest.get(`/tenants/${id}/op-logs`, { params })
}

export async function getPlatformTenantOpLogList(params?: Record<string, unknown>) {
  return normalizeListResponse(await platformRequest.get('/tenant-op-logs', { params }))
}

export function getPlatformTenantLoginLinks(id: string | number) {
  return platformRequest.get(`/tenants/${id}/login-links`)
}

export function listPlatformTenantBackups(id: string | number) {
  return platformRequest.get(`/tenants/${id}/backups`)
}

export function prunePlatformTenantBackups(id: string | number, data?: { keep?: number }) {
  return platformRequest.post(`/tenants/${id}/backups/prune`, data || {})
}

export function downloadPlatformTenantBackup(id: string | number, name: string) {
  return platformRequest.get(`/tenants/${id}/backups/download`, {
    params: { name },
    responseType: 'blob',
  })
}

export function getPlatformTenantOpsSummary() {
  return platformRequest.get('/tenants/ops-summary')
}

export function migratePlatformTenantBatch(data?: {
  ids?: Array<string | number>
  provision_status?: string
  with_seed?: boolean
  limit?: number
}) {
  return platformRequest.post('/tenants/migrate-batch', data || { provision_status: 'failed' })
}

export function getPlatformTenantSettings() {
  return platformRequest.get('/tenants/settings')
}

export function getPlatformTenantQueueStatus() {
  return platformRequest.get('/tenants/queue-status')
}

export function opsPlatformTenantBatch(data?: {
  op?: 'migrate' | 'seed' | 'backup'
  ids?: Array<string | number>
  provision_status?: string
  status?: number
  with_seed?: boolean
  keep?: number
  limit?: number
}) {
  return platformRequest.post('/tenants/ops-batch', data || {})
}

export function exportPlatformTenants(params?: Record<string, unknown>) {
  return platformRequest.get('/tenants/export', {
    params,
    responseType: 'blob',
  })
}

export function completePlatformLogin(res: { data?: { token?: string; admin?: unknown } }) {
  if (res?.data?.token) setPlatformToken(res.data.token)
  if (res?.data?.admin) setPlatformAdmin(res.data.admin)
}

export async function logoutPlatform() {
  try {
    await platformLogout()
  } catch {
    // ignore
  }
  clearPlatformSession()
}
