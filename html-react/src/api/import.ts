import { createCRUDApi, extendApi } from '@/utils/apiFactory'
import { normalizeListResponse } from '@/utils/normalize'
import request from '@/utils/request'

const baseImportApi = createCRUDApi('imports')

const importApi = extendApi(baseImportApi, {
  downloadError: (id: string | number) =>
    request({
      url: `/imports/${id}/error-file`,
      method: 'get',
      responseType: 'blob',
    }),
})

export async function getImportList(params?: Record<string, unknown>) {
  const res = await importApi.list(params)
  return normalizeListResponse(res)
}

export const getImportDetail = importApi.detail
export const deleteImport = importApi.delete
