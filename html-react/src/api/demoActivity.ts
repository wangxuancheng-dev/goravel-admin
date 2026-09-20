import { createCRUDApi, extendApi } from '@/utils/apiFactory'
import { normalizeListResponse } from '@/utils/normalize'
import request from '@/utils/request'

const baseApi = createCRUDApi('demo-activities')

const demoActivityApi = extendApi(baseApi, {
  activeCheck: (id: string | number) =>
    request({
      url: `/demo-activities/${id}/active-check`,
      method: 'get',
    }),
  syncNow: () =>
    request({
      url: '/demo-activities/sync',
      method: 'post',
    }),
})

export async function getDemoActivityList(params?: Record<string, unknown>) {
  return normalizeListResponse(await demoActivityApi.list(params))
}

export const getDemoActivityDetail = demoActivityApi.detail
export const createDemoActivity = demoActivityApi.create
export const updateDemoActivity = demoActivityApi.update
export const deleteDemoActivity = demoActivityApi.delete
export const checkDemoActivityActive = demoActivityApi.activeCheck
export const syncDemoActivities = demoActivityApi.syncNow
