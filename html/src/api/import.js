import request from '../utils/request'
import { createCRUDApi } from '../utils/apiFactory'
import { normalizeListResponse } from '../utils/normalize'

const importApi = createCRUDApi('imports')

export async function getImportList(params) {
  const res = await importApi.list(params)
  return normalizeListResponse(res)
}

export const {
  detail: getImportDetail,
  delete: deleteImport
} = importApi
