import platformRequest from '../utils/platformRequest'
import { normalizeListResponse } from '../utils/normalize'
import {
  clearPlatformSession,
  setPlatformAdmin,
  setPlatformToken
} from '../utils/platformRequest'

export function platformLogin(data) {
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

export function updatePlatformPassword(data) {
  return platformRequest.put('/password', data)
}

export async function getPlatformTenantList(params) {
  const res = await platformRequest.get('/tenants', { params })
  return normalizeListResponse(res)
}

export function getPlatformTenantDetail(id) {
  return platformRequest.get(`/tenants/${id}`)
}

export function createPlatformTenant(data) {
  return platformRequest.post('/tenants', data)
}

export function onboardPlatformTenant(data) {
  return platformRequest.post('/tenants/onboard', data)
}

export function healthInspectPlatformTenants(data = {}) {
  return platformRequest.post('/tenants/health-inspect', data)
}

export function updatePlatformTenant(id, data) {
  return platformRequest.put(`/tenants/${id}`, data)
}

export function updatePlatformTenantStatus(id, status) {
  return platformRequest.put(`/tenants/${id}/status`, { status })
}

export function updatePlatformTenantMaintenance(id, data) {
  return platformRequest.put(`/tenants/${id}/maintenance`, data)
}

export function pingPlatformTenant(id) {
  return platformRequest.post(`/tenants/${id}/ping`)
}

export function migratePlatformTenant(id, data = {}) {
  return platformRequest.post(`/tenants/${id}/migrate`, data)
}

export function seedPlatformTenant(id) {
  return platformRequest.post(`/tenants/${id}/seed`)
}

export function backupPlatformTenant(id, data = {}) {
  return platformRequest.post(`/tenants/${id}/backup`, data)
}

export function restorePlatformTenant(id, data) {
  return platformRequest.post(`/tenants/${id}/restore`, data)
}

export function deletePlatformTenant(id, data) {
  return platformRequest.delete(`/tenants/${id}`, { data })
}

export function undeletePlatformTenant(id) {
  return platformRequest.post(`/tenants/${id}/undelete`)
}

export function forceDeletePlatformTenant(id, data) {
  return platformRequest.delete(`/tenants/${id}/force`, { data })
}

export function purgePlatformTenant(id, data = {}) {
  return platformRequest.post(`/tenants/${id}/purge`, data)
}

export function getPlatformTenantOverview(id) {
  return platformRequest.get(`/tenants/${id}/overview`)
}

export function getPlatformTenantOpLogs(id, params = {}) {
  return platformRequest.get(`/tenants/${id}/op-logs`, { params })
}

export async function getPlatformTenantOpLogList(params) {
  const res = await platformRequest.get('/tenant-op-logs', { params })
  return normalizeListResponse(res)
}

export async function getPlatformLoginLogList(params) {
  const res = await platformRequest.get('/login-logs', { params })
  return normalizeListResponse(res)
}

export function getPlatformLoginLogDetail(id) {
  return platformRequest.get(`/login-logs/${id}`)
}

export async function getPlatformOperationLogList(params) {
  const res = await platformRequest.get('/operation-logs', { params })
  return normalizeListResponse(res)
}

export function getPlatformOperationLogDetail(id) {
  return platformRequest.get(`/operation-logs/${id}`)
}

export async function getPlatformSystemLogList(params) {
  const res = await platformRequest.get('/system-logs', { params })
  return normalizeListResponse(res)
}

export function getPlatformSystemLogDetail(id, params) {
  return platformRequest.get(`/system-logs/${id}`, { params })
}

export function getPlatformSystemLogModuleOptions(params) {
  return platformRequest.get('/system-logs/module-options', { params })
}

export function getPlatformTenantSystemLogSummary(id) {
  return platformRequest.get(`/tenants/${id}/system-log-summary`)
}

export async function getPlatformAdminList(params) {
  const res = await platformRequest.get('/admins', { params })
  return normalizeListResponse(res)
}

export function getPlatformAdminDetail(id) {
  return platformRequest.get(`/admins/${id}`)
}

export function createPlatformAdmin(data) {
  return platformRequest.post('/admins', data)
}

export function updatePlatformAdmin(id, data) {
  return platformRequest.put(`/admins/${id}`, data)
}

export function deletePlatformAdmin(id) {
  return platformRequest.delete(`/admins/${id}`)
}

export function resetPlatformAdminPassword(id, data) {
  return platformRequest.post(`/admins/${id}/reset-password`, data)
}

export function getPlatformTenantLoginLinks(id) {
  return platformRequest.get(`/tenants/${id}/login-links`)
}

export function getPlatformTenantDomains(id) {
  return platformRequest.get(`/tenants/${id}/domains`)
}

export function createPlatformTenantDomain(id, data) {
  return platformRequest.post(`/tenants/${id}/domains`, data)
}

export function verifyPlatformTenantDomain(id, domainId) {
  return platformRequest.post(`/tenants/${id}/domains/${domainId}/verify`)
}

export function setPrimaryPlatformTenantDomain(id, domainId) {
  return platformRequest.put(`/tenants/${id}/domains/${domainId}/primary`)
}

export function disablePlatformTenantDomain(id, domainId) {
  return platformRequest.put(`/tenants/${id}/domains/${domainId}/disable`)
}

export function deletePlatformTenantDomain(id, domainId) {
  return platformRequest.delete(`/tenants/${id}/domains/${domainId}`)
}

export function listPlatformTenantBackups(id) {
  return platformRequest.get(`/tenants/${id}/backups`)
}

export function downloadPlatformTenantBackup(id, name) {
  return platformRequest.get(`/tenants/${id}/backups/download`, {
    params: { name },
    responseType: 'blob'
  })
}

export function prunePlatformTenantBackups(id, data = {}) {
  return platformRequest.post(`/tenants/${id}/backups/prune`, data)
}

export function getPlatformTenantOpsSummary() {
  return platformRequest.get('/tenants/ops-summary')
}

export function getPlatformTenantSettings() {
  return platformRequest.get('/tenants/settings')
}

export function getPlatformTenantQueueStatus() {
  return platformRequest.get('/tenants/queue-status')
}

export function migratePlatformTenantBatch(data) {
  return platformRequest.post('/tenants/migrate-batch', data || { provision_status: 'failed' })
}

export function opsPlatformTenantBatch(data) {
  return platformRequest.post('/tenants/ops-batch', data || {})
}

export function exportPlatformTenants(params = {}) {
  return platformRequest.get('/tenants/export', {
    params,
    responseType: 'blob'
  })
}

export async function completePlatformLogin(res) {
  const token = res?.data?.token
  const admin = res?.data?.admin
  if (token) setPlatformToken(token)
  if (admin) setPlatformAdmin(admin)
  return { token, admin }
}

export async function logoutPlatform() {
  try {
    await platformLogout()
  } catch {
    // ignore
  }
  clearPlatformSession()
}
