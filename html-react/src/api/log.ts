import request from '@/utils/request'
import { normalizeListResponse } from '@/utils/normalize'
import type { ApiResponse, PaginatedData } from '@/types'

export async function getOperationLogList(params?: Record<string, unknown>) {
  const res = await request({ url: '/operation-logs', method: 'get', params })
  return normalizeListResponse(res as unknown as ApiResponse<PaginatedData>) as ApiResponse<PaginatedData>
}

export function getOperationLogTitleOptions() {
  return request({ url: '/operation-logs/title-options', method: 'get' }) as Promise<ApiResponse>
}

export function getOperationLogDetail(id: string | number) {
  return request({ url: `/operation-logs/${id}`, method: 'get' }) as Promise<ApiResponse>
}

export function deleteOperationLog(id: string | number) {
  return request({ url: `/operation-logs/${id}`, method: 'delete' })
}

export function batchDeleteOperationLogs(ids: Array<string | number>) {
  return request({ url: '/operation-logs/batch-delete', method: 'post', data: { ids } })
}

export function cleanOperationLogs(params: Record<string, unknown> = {}) {
  return request({ url: '/operation-logs/clean', method: 'post', params })
}

export async function getLoginLogList(params?: Record<string, unknown>) {
  const res = await request({ url: '/login-logs', method: 'get', params })
  return normalizeListResponse(res as unknown as ApiResponse<PaginatedData>) as ApiResponse<PaginatedData>
}

export function getLoginLogDetail(id: string | number) {
  return request({ url: `/login-logs/${id}`, method: 'get' }) as Promise<ApiResponse>
}

export function deleteLoginLog(id: string | number) {
  return request({ url: `/login-logs/${id}`, method: 'delete' })
}

export function batchDeleteLoginLogs(ids: Array<string | number>) {
  return request({ url: '/login-logs/batch-delete', method: 'post', data: { ids } })
}

export function cleanLoginLogs(params: Record<string, unknown> = {}) {
  return request({ url: '/login-logs/clean', method: 'post', params })
}

export function refreshLoginLogLocation(id?: string | number) {
  return request({
    url: '/login-logs/refresh-location',
    method: 'post',
    data: id ? { id } : { fill_empty: true },
  }) as Promise<ApiResponse>
}

export async function getSystemLogList(params?: Record<string, unknown>) {
  const res = await request({ url: '/system-logs', method: 'get', params })
  return normalizeListResponse(res as unknown as ApiResponse<PaginatedData>) as ApiResponse<PaginatedData>
}

export function getSystemLogModuleOptions() {
  return request({ url: '/system-logs/module-options', method: 'get' }) as Promise<ApiResponse>
}

export function getSystemLogDetail(id: string | number) {
  return request({ url: `/system-logs/${id}`, method: 'get' }) as Promise<ApiResponse>
}

export function deleteSystemLog(id: string | number) {
  return request({ url: `/system-logs/${id}`, method: 'delete' })
}

export function batchDeleteSystemLogs(ids: Array<string | number>) {
  return request({ url: '/system-logs/batch-delete', method: 'post', data: { ids } })
}

export function cleanSystemLogs(params: Record<string, unknown> = {}) {
  return request({ url: '/system-logs/clean', method: 'post', params })
}
