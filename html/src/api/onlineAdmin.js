import request from '../utils/request'
import { normalizeListResponse } from '../utils/normalize'

export async function getOnlineAdminList(params) {
  const res = await request({
    url: '/online-admins',
    method: 'get',
    params
  })
  return normalizeListResponse(res)
}

export function kickOutOnlineAdmin(id, data) {
  return request({
    url: `/online-admins/${id}`,
    method: 'delete',
    ...(data ? { data } : {})
  })
}

export function batchKickOutOnlineAdmins(tokenIds, data = {}) {
  return request({
    url: '/online-admins/batch-kick-out',
    method: 'post',
    data: {
      token_ids: tokenIds.join(','),
      ...data
    }
  })
}
