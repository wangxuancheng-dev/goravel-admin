import { normalizeListResponse } from '@/utils/normalize'
import request from '@/utils/request'

export async function getOnlineAdminList(params?: Record<string, unknown>) {
  const res = await request({
    url: '/online-admins',
    method: 'get',
    params,
  })
  return normalizeListResponse(res)
}

export function kickOutOnlineAdmin(id: string | number, data?: Record<string, unknown>) {
  return request({
    url: `/online-admins/${id}`,
    method: 'delete',
    ...(data ? { data } : {}),
  })
}

export function batchKickOutOnlineAdmins(
  tokenIds: Array<string | number>,
  data?: Record<string, unknown>,
) {
  return request({
    url: '/online-admins/batch-kick-out',
    method: 'post',
    data: { token_ids: tokenIds.join(','), ...data },
  })
}
