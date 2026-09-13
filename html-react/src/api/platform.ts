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

export function backupPlatformTenant(id: string | number) {
  return platformRequest.post(`/tenants/${id}/backup`)
}

export function listPlatformTenantBackups(id: string | number) {
  return platformRequest.get(`/tenants/${id}/backups`)
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
