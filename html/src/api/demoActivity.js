import request from '../utils/request'
import { createCRUDApi, extendApi } from '../utils/apiFactory'
import { normalizeListResponse } from '../utils/normalize'

const baseApi = createCRUDApi('demo-activities')

const demoActivityApi = extendApi(baseApi, {
  activeCheck: (id) =>
    request({
      url: `/demo-activities/${id}/active-check`,
      method: 'get'
    }),
  syncNow: () =>
    request({
      url: '/demo-activities/sync',
      method: 'post'
    })
})

export async function getDemoActivityList(params) {
  const res = await demoActivityApi.list(params)
  return normalizeListResponse(res)
}

export const getDemoActivityDetail = demoActivityApi.detail
export const createDemoActivity = demoActivityApi.create
export const updateDemoActivity = demoActivityApi.update
export const deleteDemoActivity = demoActivityApi.delete
export const checkDemoActivityActive = demoActivityApi.activeCheck
export const syncDemoActivities = demoActivityApi.syncNow
