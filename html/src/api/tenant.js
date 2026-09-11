import { createCRUDApi, extendApi } from '../utils/apiFactory'
import { normalizeListResponse } from '../utils/normalize'
import request from '../utils/request'

const tenantApi = extendApi(createCRUDApi('tenants'), {
  updateStatus: (id, data) => request.put(`/tenants/${id}/status`, data)
})

export async function getTenantList(params) {
  const res = await tenantApi.list(params)
  return normalizeListResponse(res)
}

export const getTenantDetail = tenantApi.detail
export const createTenant = tenantApi.create
export function updateTenantStatus(id, status) {
  return tenantApi.updateStatus(id, { status })
}
