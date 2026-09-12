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

export function platformLogout() {
  return platformRequest.post('/logout')
}

export function platformInfo() {
  return platformRequest.get('/info')
}

export function platformHealth() {
  return platformRequest.get('/health')
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

export function updatePlatformTenant(id, data) {
  return platformRequest.put(`/tenants/${id}`, data)
}

export function updatePlatformTenantStatus(id, status) {
  return platformRequest.put(`/tenants/${id}/status`, { status })
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

export function backupPlatformTenant(id) {
  return platformRequest.post(`/tenants/${id}/backup`)
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
