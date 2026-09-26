import { createCRUDApi } from '../utils/apiFactory'
import { normalizeListResponse } from '../utils/normalize'

const allowlistApi = createCRUDApi('allowlists')

export async function getAllowlistList(params) {
  const res = await allowlistApi.list(params)
  return normalizeListResponse(res)
}

export const getAllowlistDetail = allowlistApi.detail
export const createAllowlist = allowlistApi.create
export const updateAllowlist = allowlistApi.update
export const deleteAllowlist = allowlistApi.delete
