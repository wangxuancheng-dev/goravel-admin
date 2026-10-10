import request from '../utils/request'

// Read-only plan summary for the current tenant (GET /entitlements/me)
export function getMyPlan() {
  return request({
    url: '/entitlements/me',
    method: 'get'
  })
}
