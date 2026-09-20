import request from '@/utils/request'
import { createCRUDApi, extendApi } from '@/utils/apiFactory'
import { normalizeListResponse } from '@/utils/normalize'

const baseAdminApi = createCRUDApi('admins')

const adminApi = extendApi(baseAdminApi, {
  export: (params?: Record<string, unknown>) =>
    request({
      url: '/admins/export',
      method: 'post',
      data: params,
    }),
  resetPassword: (
    id: string | number,
    data: { password: string; confirm_code?: string },
  ) =>
    request({
      url: `/admins/${id}/password`,
      method: 'put',
      data,
    }),
  kickOutUser: (id: string | number) =>
    request({
      url: `/admins/${id}/tokens`,
      method: 'delete',
    }),
  unbindGoogleAuth: (id: string | number, data?: Record<string, unknown>) =>
    request({
      url: `/admins/${id}/unbind-google-auth`,
      method: 'post',
      data,
    }),
  resetGoogleAuth: (id: string | number) =>
    request({
      url: `/admins/${id}/reset-google-auth`,
      method: 'post',
    }),
})

export async function getAdminList(params?: Record<string, unknown>) {
  const res = await adminApi.list(params)
  return normalizeListResponse(res)
}

export const {
  detail: getAdminDetail,
  create: createAdmin,
  update: updateAdmin,
  delete: deleteAdmin,
  export: exportAdmin,
  resetPassword,
  kickOutUser,
  unbindGoogleAuth: unbindAdminGoogleAuth,
  resetGoogleAuth: resetAdminGoogleAuth,
} = adminApi
