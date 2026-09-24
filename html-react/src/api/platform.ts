import platformRequest, {
  clearPlatformSession,
  setPlatformAdmin,
  setPlatformToken,
} from '@/utils/platformRequest'
import { normalizeListResponse } from '@/utils/normalize'

export function platformLogin(data: {
  username: string
  password: string
  captcha_id?: string
  captcha_answer?: string
  google_code?: string
}) {
  return platformRequest.post('/login', data)
}

export function getPlatformLoginCaptcha() {
  return platformRequest.get('/login/captcha')
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

export function getPlatformOpsOverview() {
  return platformRequest.get('/ops/overview')
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

export function onboardPlatformTenant(data: Record<string, unknown>) {
  return platformRequest.post('/tenants/onboard', data)
}

export function healthInspectPlatformTenants(data?: { limit?: number; alert?: boolean }) {
  return platformRequest.post('/tenants/health-inspect', data || {})
}

export function updatePlatformTenant(id: string | number, data: Record<string, unknown>) {
  return platformRequest.put(`/tenants/${id}`, data)
}

export function updatePlatformTenantStatus(id: string | number, status: number) {
  return platformRequest.put(`/tenants/${id}/status`, { status })
}

export function updatePlatformTenantMaintenance(
  id: string | number,
  data: { maintenance: boolean; maintenance_message?: string },
) {
  return platformRequest.put(`/tenants/${id}/maintenance`, data)
}

export function pingPlatformTenant(id: string | number) {
  return platformRequest.post(`/tenants/${id}/ping`)
}

export function migratePlatformTenant(id: string | number, data?: { with_seed?: boolean }) {
  return platformRequest.post(`/tenants/${id}/migrate`, data || {})
}

export function rollbackPlatformTenant(id: string | number, data?: { step?: number; batch?: number }) {
  return platformRequest.post(`/tenants/${id}/rollback`, data || {})
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
  data: {
    confirm_code: string
    drop_database?: boolean
    purge_objects?: boolean
    purge_backups?: boolean
    purge_files?: boolean
  },
) {
  return platformRequest.delete(`/tenants/${id}`, { data })
}

export function undeletePlatformTenant(id: string | number) {
  return platformRequest.post(`/tenants/${id}/undelete`)
}

export function forceDeletePlatformTenant(
  id: string | number,
  data: {
    confirm_code: string
    purge_objects?: boolean
    purge_backups?: boolean
    purge_files?: boolean
  },
) {
  return platformRequest.delete(`/tenants/${id}/force`, { data })
}

export function purgePlatformTenant(
  id: string | number,
  data?: { purge_objects?: boolean; purge_backups?: boolean; purge_files?: boolean },
) {
  return platformRequest.post(`/tenants/${id}/purge`, data || {})
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

export async function getPlatformLoginLogList(params?: Record<string, unknown>) {
  return normalizeListResponse(await platformRequest.get('/login-logs', { params }))
}

export function getPlatformLoginLogDetail(id: string | number) {
  return platformRequest.get(`/login-logs/${id}`)
}

export async function getPlatformOperationLogList(params?: Record<string, unknown>) {
  return normalizeListResponse(await platformRequest.get('/operation-logs', { params }))
}

export function getPlatformOperationLogDetail(id: string | number) {
  return platformRequest.get(`/operation-logs/${id}`)
}

export async function getPlatformSystemLogList(params?: Record<string, unknown>) {
  return normalizeListResponse(await platformRequest.get('/system-logs', { params }))
}

export function getPlatformSystemLogDetail(id: string | number, params?: Record<string, unknown>) {
  return platformRequest.get(`/system-logs/${id}`, { params })
}

export function getPlatformSystemLogModuleOptions(params?: Record<string, unknown>) {
  return platformRequest.get('/system-logs/module-options', { params })
}

export function getPlatformTenantSystemLogSummary(id: string | number) {
  return platformRequest.get(`/tenants/${id}/system-log-summary`)
}

export async function getPlatformAdminList(params?: Record<string, unknown>) {
  return normalizeListResponse(await platformRequest.get('/admins', { params }))
}

export function getPlatformAdminDetail(id: string | number) {
  return platformRequest.get(`/admins/${id}`)
}

export function createPlatformAdmin(data: {
  username: string
  password: string
  name?: string
  role?: string
}) {
  return platformRequest.post('/admins', data)
}

export function updatePlatformAdmin(
  id: string | number,
  data: { name?: string; role?: string; status?: number },
) {
  return platformRequest.put(`/admins/${id}`, data)
}

export function deletePlatformAdmin(id: string | number) {
  return platformRequest.delete(`/admins/${id}`)
}

export function resetPlatformAdminPassword(
  id: string | number,
  data: { password: string; confirm_password?: string },
) {
  return platformRequest.post(`/admins/${id}/reset-password`, data)
}

export function getPlatformTenantLoginLinks(id: string | number) {
  return platformRequest.get(`/tenants/${id}/login-links`)
}

export function getPlatformTenantDomains(id: string | number) {
  return platformRequest.get(`/tenants/${id}/domains`)
}

export function createPlatformTenantDomain(
  id: string | number,
  data: { host: string; ssl_mode?: string; is_primary?: boolean },
) {
  return platformRequest.post(`/tenants/${id}/domains`, data)
}

export function verifyPlatformTenantDomain(id: string | number, domainId: string | number) {
  return platformRequest.post(`/tenants/${id}/domains/${domainId}/verify`)
}

export function setPrimaryPlatformTenantDomain(id: string | number, domainId: string | number) {
  return platformRequest.put(`/tenants/${id}/domains/${domainId}/primary`)
}

export function disablePlatformTenantDomain(id: string | number, domainId: string | number) {
  return platformRequest.put(`/tenants/${id}/domains/${domainId}/disable`)
}

export function deletePlatformTenantDomain(id: string | number, domainId: string | number) {
  return platformRequest.delete(`/tenants/${id}/domains/${domainId}`)
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
  op?: 'migrate' | 'seed' | 'backup' | 'rollback'
  ids?: Array<string | number>
  provision_status?: string
  last_op?: string
  last_op_status?: string
  status?: number
  with_seed?: boolean
  keep?: number
  limit?: number
  step?: number
  batch?: number
}) {
  return platformRequest.post('/tenants/ops-batch', data || {})
}

export function exportPlatformTenants(params?: Record<string, unknown>) {
  return platformRequest.get('/tenants/export', {
    params,
    responseType: 'blob',
  })
}

export function getPlatformTenantAdmins(id: string | number) {
  return platformRequest.get(`/tenants/${id}/admins`)
}

export function resetPlatformTenantAdminPassword(
  id: string | number,
  adminId: string | number,
  data: { password: string; confirm_password?: string },
) {
  return platformRequest.post(`/tenants/${id}/admins/${adminId}/reset-password`, data)
}

export function unlockPlatformTenantAdmin(id: string | number, data: { username: string }) {
  return platformRequest.post(`/tenants/${id}/admins/unlock`, data)
}

export function resetPlatformTenantAdmin2FA(id: string | number, adminId: string | number) {
  return platformRequest.post(`/tenants/${id}/admins/${adminId}/reset-2fa`)
}

export function getPlatformTenantAuditSummary(id: string | number) {
  return platformRequest.get(`/tenants/${id}/audit-summary`)
}

export async function getPlatformAlertDeliveryList(params?: Record<string, unknown>) {
  return normalizeListResponse(await platformRequest.get('/alert-deliveries', { params }))
}

export function retryPlatformAlertDelivery(id: string | number) {
  return platformRequest.post(`/alert-deliveries/${id}/retry`)
}

export function getPlatformSecurityStatus() {
  return platformRequest.get('/security/2fa/status')
}

export function getPlatformSecurityQRCode() {
  return platformRequest.get('/security/2fa/qrcode')
}

export function bindPlatformSecurity2FA(data: { secret: string; code: string }) {
  return platformRequest.post('/security/2fa/bind', data)
}

export function unbindPlatformSecurity2FA(data: { code: string }) {
  return platformRequest.post('/security/2fa/unbind', data)
}

export function updatePlatformAllowedIPs(data: { allowed_ips: string }) {
  return platformRequest.put('/security/allowed-ips', data)
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
