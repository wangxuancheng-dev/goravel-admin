import { createCRUDApi, extendApi } from '@/utils/apiFactory'
import { normalizeListResponse } from '@/utils/normalize'
import request from '@/utils/request'
import type { ApiResponse, PaginatedData } from '@/types'

const tenantApi = extendApi(createCRUDApi('tenants'), {
  updateStatus: (id: string | number, data: { status: number }) =>
    request.put(`/tenants/${id}/status`, data),
})

export async function getTenantList(params?: Record<string, unknown>) {
  return normalizeListResponse(await tenantApi.list(params)) as Promise<
    ApiResponse<PaginatedData>
  >
}

export const getTenantDetail = tenantApi.detail
export const createTenant = tenantApi.create

export function updateTenantStatus(id: string | number, status: number) {
  return tenantApi.updateStatus(id, { status })
}
