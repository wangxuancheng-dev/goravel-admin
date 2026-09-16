import request from '@/utils/request'
import type { ApiResponse, UserInfoPayload } from '@/types'

export function getProfile() {
  return request({
    url: '/info',
    method: 'get',
  }) as Promise<ApiResponse<UserInfoPayload>>
}

export function updateProfile(data: Record<string, unknown>) {
  return request({
    url: '/profile',
    method: 'put',
    data,
  })
}

export function updatePassword(data: {
  old_password: string
  new_password: string
  confirm_password: string
}) {
  return request({
    url: '/password',
    method: 'put',
    data,
  })
}
